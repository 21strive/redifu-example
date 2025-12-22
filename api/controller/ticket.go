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
	"redifu-example/pkg/service"
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
	SeedSortedByReporter(string) error
	SeedByRandId(string) error
}

type TicketCUDController struct {
	ticketService *service.TicketService
}

func (cud *TicketCUDController) CreateTicket(c *fiber.Ctx) error {
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

func (cud *TicketCUDController) PatchTicket(c *fiber.Ctx) error {
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

func (cud *TicketCUDController) ResolveTicket(c *fiber.Ctx) error {
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

func (cud *TicketCUDController) DeleteTicket(c *fiber.Ctx) error {
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

func NewTicketCUDController(db *sql.DB, redisClient redis.UniversalClient) *TicketCUDController {
	ticketService := service.NewTicketService()
	accountService := service.NewAccountService()

	ticketService.InitRepository(db, redisClient, accountService)
	accountService.InitRepository(db, redisClient)
	return &TicketCUDController{ticketService: ticketService}
}

type TicketFetchController struct {
	ticketService *service.TicketService
	seedHandler   TicketSeeder
}

func (fh *TicketFetchController) GetTicket(c *fiber.Ctx) error {
	ticketRandId := c.Params("randid")
	if ticketRandId == "" {
		return logger.Error(c, fiber.StatusBadRequest, fmt.Errorf("ticketRandId is empty"), "T100", "GetTicket.Params")
	}

	ticket, account, isBlank, errFetch := fh.ticketService.GetTicket(ticketRandId)
	if errFetch != nil {
		return logger.Error(c, fiber.StatusInternalServerError, errFetch, "T500", "GetTicket.Fetch")
	}
	if ticket == nil {
		if isBlank {
			return logger.Error(c, fiber.StatusNotFound, fmt.Errorf("ticket not found"), "T404", "GetTicket.NotFound")
		} else {
			errSeedTicket := fh.seedHandler.SeedByRandId(ticketRandId)
			if errSeedTicket != nil {
				return logger.Error(c, fiber.StatusInternalServerError, errSeedTicket, "T500", "GetTicket.Seed")
			}

			ticket, account, isBlank, errFetch = fh.ticketService.GetTicket(ticketRandId)
			if errFetch != nil {
				return logger.Error(c, fiber.StatusInternalServerError, errFetch, "T550", "GetTicket.Fetch")
			}
			if ticket == nil || isBlank {
				return logger.Error(c, fiber.StatusNotFound, fmt.Errorf("ticket not found"), "T404", "GetTicket.NotFound")
			}
		}
	}

	return c.JSON(map[string]interface{}{
		"ticket":  ticket,
		"account": account,
	})
}

func (fh *TicketFetchController) GetTickets(c *fiber.Ctx) error {
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

func (fh *TicketFetchController) GetTicketsByReporter(c *fiber.Ctx) error {
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

func NewTicketFetchController(redisClient redis.UniversalClient) *TicketFetchController {
	ticketService := service.NewTicketService()
	ticketService.InitFetcher(redisClient)

	var seedHandler TicketSeeder
	if os.Getenv("OP_MODE") == "MONO" {
		seedHandler = NewSelfSeedHandler()
	} else if os.Getenv("OP_MODE") == "GETTER" {
		// seedHandler = GRPCHandler
	}

	return &TicketFetchController{
		ticketService: ticketService,
		seedHandler:   seedHandler,
	}
}

type SelfTicketSeedHandler struct {
}

func (sh *SelfTicketSeedHandler) SeedTimeline(int64, string) error {
	return nil
}

func (sh *SelfTicketSeedHandler) SeedSortedByReporter(string) error {
	return nil
}

func (sh *SelfTicketSeedHandler) SeedByRandId(string) error {
	return nil
}

func NewSelfSeedHandler() *SelfTicketSeedHandler {
	return &SelfTicketSeedHandler{}
}

// Implement type GRPCSeedHandler here
