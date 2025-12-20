package controller

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/21strive/redifu"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"os"
	"redifu-example/internal/logger"
	"redifu-example/internal/service"
	"strings"
)

type CreateTicketRequest struct {
	Description  string `json:"description"`
	ReporterUUID string `json:"reporter_uuid"`
}

type UpdateTicketDescriptionRequest struct {
	TicketUUID  string `json:"ticket_uuid"`
	Description string `json:"description"`
}

type UpdateAccountRequest struct {
	AccountUUID string `json:"account_uuid"`
	Name        string `json:"name"`
	Email       string `json:"email"`
}

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
	var reqBody CreateTicketRequest
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
	var reqBody UpdateTicketDescriptionRequest
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
	var reqBody UpdateTicketDescriptionRequest
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

	ticket, validLastRandId, position, isSeedingRequired, errFetch := fh.ticketService.GetTickets(lastRandIdArray)
	if errFetch != nil {
		if errors.Is(errFetch, redifu.ResetPagination) {
			lastRandIdArray = []string{}
		} else {
			return logger.Error(c, fiber.StatusInternalServerError, errFetch, "T500", "GetTickets.Fetch")
		}
	}

	if isSeedingRequired {
		errSeedTicketTimeline := fh.seedHandler.SeedTimeline(int64(len(ticket)), validLastRandId)
		if errSeedTicketTimeline != nil {
			return logger.Error(c, fiber.StatusInternalServerError, errSeedTicketTimeline, "T500", "GetTickets.Seed")
		}

		ticket, validLastRandId, position, isSeedingRequired, errFetch = fh.ticketService.GetTickets(lastRandIdArray)
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

func (fh *FetchController) GetTicketsByReporter(c *fiber.Ctx) error {
	accountUUID := c.Params("accountUUID")
	ticket, requireSeeding, errFetch := fh.ticketService.GetTicketsByReporter(accountUUID)
	if errFetch != nil {
		return logger.Error(c, fiber.StatusInternalServerError, errFetch, "T500", "GetTicketSorted.Fetch")
	}
	if requireSeeding {
		errSeedTicketSorted := fh.seedHandler.SeedSortedByReporter(accountUUID)
		if errSeedTicketSorted != nil {
			return logger.Error(c, fiber.StatusInternalServerError, errSeedTicketSorted, "T500", "GetTicketSorted.Seed")
		}

		ticket, requireSeeding, errFetch = fh.ticketService.GetTicketsByReporter(accountUUID)
		if errFetch != nil {
			return logger.Error(c, fiber.StatusInternalServerError, errFetch, "T500", "GetTicketSortedAfterSeed.Fetch")
		}
	}

	return c.JSON(ticket)
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
