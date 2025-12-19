package controller

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/21strive/redifu"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"os"
	"redifu-example/internal/service"
	"redifu-example/pkg/logger"
	"redifu-example/request"
	"strings"
)

type TicketSeeder interface {
	SeedTimeline(int64, string) error
	SeedTimelineByReporter(int64, string, string) error
	SeedSorted() error
	SeedSortedByReporter(string) error
	SeedByRandId(string) error
}

type CUDController struct {
	ticketService *service.TicketService
}

func (cud *CUDController) CreateTicket(c *fiber.Ctx) error {
	var reqBody request.CreateTicketRequest
	if err := c.BodyParser(&reqBody); err != nil {
		return logger.Error(c, fiber.StatusBadRequest, err, "T100", "CreateTicket.BodyParser")
	}

	errCreate := cud.ticketService.Create(reqBody.Description, reqBody.ReporterUUID)
	if errCreate != nil {
		return logger.Error(c, fiber.StatusInternalServerError, errCreate, "T500", "CreateTicket.Create")
	}

	return c.SendStatus(fiber.StatusCreated)
}

func (cud *CUDController) PatchTicket(c *fiber.Ctx) error {
	var reqBody request.UpdateTicketDescriptionRequest
	if err := c.BodyParser(&reqBody); err != nil {
		return logger.Error(c, fiber.StatusBadRequest, err, "T100", "UpdateTicketDescription.BodyParser")
	}

	errUpdate := cud.ticketService.UpdateDescription(reqBody.TicketUUID, reqBody.Description)
	if errUpdate != nil {
		return logger.Error(c, fiber.StatusInternalServerError, errUpdate, "T500", "UpdateTicketDescription.Update")
	}
	return c.SendStatus(fiber.StatusOK)
}

func (cud *CUDController) ResolveTicket(c *fiber.Ctx) error {
	var reqBody request.UpdateTicketDescriptionRequest
	if err := c.BodyParser(&reqBody); err != nil {
		return logger.Error(c, fiber.StatusBadRequest, err, "T100", "UpdateTicketDescription.BodyParser")
	}

	errResolve := cud.ticketService.ResolveTicket(reqBody.TicketUUID)
	if errResolve != nil {
		return logger.Error(c, fiber.StatusInternalServerError, errResolve, "T500", "UpdateTicketDescription.Resolve")
	}

	return c.SendStatus(fiber.StatusOK)
}

func (cud *CUDController) DeleteTicket(c *fiber.Ctx) error {
	ticketUUID := c.Params("ticketUUID")
	if ticketUUID == "" {
		return logger.Error(c, fiber.StatusBadRequest, fmt.Errorf("ticketUUID is empty"), "T100", "DeleteTicket.Params")
	}

	errDelete := cud.ticketService.Delete(ticketUUID)
	if errDelete != nil {
		return logger.Error(c, fiber.StatusInternalServerError, errDelete, "T500", "DeleteTicket.Delete")
	}
	return c.SendStatus(fiber.StatusOK)
}

func NewCUDController(db *sql.DB, redisClient redis.UniversalClient) *CUDController {
	ticketService := service.NewTicketService()
	ticketService.InitRepository(db, redisClient)
	return &CUDController{ticketService: ticketService}
}

type FetchController struct {
	ticketService *service.TicketService
	seedHandler   TicketSeeder
}

func (fh *FetchController) GetTicketTimeline(c *fiber.Ctx) error {
	var lastRandIdArray []string
	lastRandId := c.Query("lastRandId")
	if lastRandId != "" {
		lastRandIdArray = strings.Split(lastRandId, ",")
	}

	ticket, validLastRandId, position, isSeedingRequired, errFetch := fh.ticketService.GetTicketTimeline(lastRandIdArray)
	if errFetch != nil {
		if errors.Is(errFetch, redifu.ResetPagination) {
			lastRandIdArray = []string{}
		} else {
			return logger.Error(c, fiber.StatusInternalServerError, errFetch, "T500", "GetTicketTimeline.Fetch")
		}
	}

	if isSeedingRequired {
		errSeedTicketTimeline := fh.seedHandler.SeedTimeline(int64(len(ticket)), validLastRandId)
		if errSeedTicketTimeline != nil {
			return logger.Error(c, fiber.StatusInternalServerError, errSeedTicketTimeline, "T500", "GetTicketTimeline.Seed")
		}

		ticket, validLastRandId, position, isSeedingRequired, errFetch = fh.ticketService.GetTicketTimeline(lastRandIdArray)
		if errFetch != nil {
			return logger.Error(c, fiber.StatusInternalServerError, errFetch, "T500", "GetTicketTimelineAfterSeed.Fetch")
		}
	}

	c.Set("Content-Type", "application/json")
	return c.JSON(map[string]interface{}{
		"position": position,
		"tickets":  ticket,
	})
}

func (fh *FetchController) GetTicketTimelineByReporter(c *fiber.Ctx) error {
	var lastRandIdArray []string
	lastRandId := c.Query("lastRandId")
	if lastRandId != "" {
		lastRandIdArray = strings.Split(lastRandId, ",")
	}
	reporterUUID := c.Params("reporterUUID")

	ticket, validLastRandId, position, isSeedingRequired, errFetch := fh.ticketService.GetTicketTimelineByReporter(lastRandIdArray, reporterUUID)
	if errFetch != nil {
		return logger.Error(c, fiber.StatusInternalServerError, errFetch, "T500", "GetTicketTimeline.Fetch")
	}

	if isSeedingRequired {
		errSeedTicketTimeline := fh.seedHandler.SeedTimelineByReporter(int64(len(ticket)), validLastRandId, reporterUUID)
		if errSeedTicketTimeline != nil {
			return logger.Error(c, fiber.StatusInternalServerError, errSeedTicketTimeline, "T500", "GetTicketTimeline.Seed")
		}

		ticket, validLastRandId, position, isSeedingRequired, errFetch = fh.ticketService.GetTicketTimelineByReporter(lastRandIdArray, reporterUUID)
		if errFetch != nil {
			return logger.Error(c, fiber.StatusInternalServerError, errFetch, "T500", "GetTicketTimeline.Fetch")
		}
	}

	c.Set("Content-Type", "application/json")
	return c.JSON(map[string]interface{}{
		"position": position,
		"tickets":  ticket,
	})
}

func NewFetchController(redisClient redis.UniversalClient) *FetchController {
	ticketService := service.NewTicketService()
	ticketService.InitFetcher(redisClient)

	var seedHandler TicketSeeder
	if os.Getenv("OP_MODE") == "MONO" {
		seedHandler = NewSelfSeedHandler()
	} else if os.Getenv("OP_MODE") == "GETTER" {
		// seedHandler = GRPCHandler
	}

	return &FetchController{
		ticketService: ticketService,
		seedHandler:   seedHandler,
	}
}

type SelfSeedHandler struct {
}

func (sh *SelfSeedHandler) SeedTimeline(int64, string) error {
	return nil
}

func (sh *SelfSeedHandler) SeedTimelineByReporter(int64, string, string) error {
	return nil
}

func (sh *SelfSeedHandler) SeedSorted() error {
	return nil
}

func (sh *SelfSeedHandler) SeedSortedByReporter(string) error {
	return nil
}

func (sh *SelfSeedHandler) SeedByRandId(string) error {
	return nil
}

func NewSelfSeedHandler() *SelfSeedHandler {
	return &SelfSeedHandler{}
}

// Implement type GRPCSeedHandler here
