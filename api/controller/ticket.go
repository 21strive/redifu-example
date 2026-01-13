package controller

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/21strive/redifu"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"redifu-example/internal/logger"
	"redifu-example/pkg/service"
	"strconv"
	"strings"
	"time"
)

type CreateTicketRequest struct {
	Description  string `json:"description"`
	ReporterUUID string `json:"reporter_uuid"`
	SecurityRisk int64  `json:"security_risk"`
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

type TicketCUDController struct {
	ticketService *service.TicketService
}

func (cud *TicketCUDController) CreateTicket(c *fiber.Ctx) error {
	var reqBody CreateTicketRequest
	mainCtx := c.Context()

	if err := c.BodyParser(&reqBody); err != nil {
		return logger.Error(c, fiber.StatusBadRequest, err, "T100", "CreateTicket.BodyParser")
	}

	errCreate := cud.ticketService.Create(mainCtx, reqBody.Description, reqBody.ReporterUUID, reqBody.SecurityRisk)
	if errCreate != nil {
		return logger.Error(c, fiber.StatusInternalServerError, errCreate, "T500", "CreateTicket.Create")
	}

	return c.SendStatus(fiber.StatusCreated)
}

func (cud *TicketCUDController) PatchTicket(c *fiber.Ctx) error {
	var reqBody UpdateTicketDescriptionRequest
	mainCtx := c.Context()

	if err := c.BodyParser(&reqBody); err != nil {
		return logger.Error(c, fiber.StatusBadRequest, err, "T100", "UpdateTicketDescription.BodyParser")
	}

	errUpdate := cud.ticketService.UpdateDescription(mainCtx, reqBody.TicketUUID, reqBody.Description)
	if errUpdate != nil {
		return logger.Error(c, fiber.StatusInternalServerError, errUpdate, "T500", "UpdateTicketDescription.Update")
	}
	return c.SendStatus(fiber.StatusOK)
}

func (cud *TicketCUDController) ResolveTicket(c *fiber.Ctx) error {
	var reqBody UpdateTicketDescriptionRequest
	mainCtx := c.Context()

	if err := c.BodyParser(&reqBody); err != nil {
		return logger.Error(c, fiber.StatusBadRequest, err, "T100", "UpdateTicketDescription.BodyParser")
	}

	errResolve := cud.ticketService.ResolveTicket(mainCtx, reqBody.TicketUUID)
	if errResolve != nil {
		return logger.Error(c, fiber.StatusInternalServerError, errResolve, "T500", "UpdateTicketDescription.Resolve")
	}

	return c.SendStatus(fiber.StatusOK)
}

func (cud *TicketCUDController) DeleteTicket(c *fiber.Ctx) error {
	mainCtx := c.Context()
	ticketUUID := c.Params("ticketUUID")
	if ticketUUID == "" {
		return logger.Error(c, fiber.StatusBadRequest, fmt.Errorf("ticketUUID is empty"), "T100", "DeleteTicket.Params")
	}

	errDelete := cud.ticketService.Delete(mainCtx, ticketUUID)
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
	mainCtx := c.Context()
	ticketRandId := c.Params("randid")
	if ticketRandId == "" {
		return logger.Error(c, fiber.StatusBadRequest, fmt.Errorf("ticketRandId is empty"), "T100", "GetTicket.Params")
	}

	ticket, account, isBlank, errFetch := fh.ticketService.GetTicket(mainCtx, ticketRandId)
	if errFetch != nil {
		return logger.Error(c, fiber.StatusInternalServerError, errFetch, "T500", "GetTicket.Fetch")
	}
	if ticket == nil {
		if isBlank {
			return logger.Error(c, fiber.StatusNotFound, fmt.Errorf("ticket not found"), "T404", "GetTicket.NotFound")
		} else {
			errSeedTicket := fh.seedHandler.SeedTicket(mainCtx, ticketRandId)
			if errSeedTicket != nil {
				return logger.Error(c, fiber.StatusInternalServerError, errSeedTicket, "T500", "GetTicket.Seed")
			}

			ticket, account, isBlank, errFetch = fh.ticketService.GetTicket(mainCtx, ticketRandId)
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
	mainCtx := c.Context()
	sortBy := c.Query("sort")
	page := c.Query("page")
	lowerbound := c.Query("lowerbound")
	upperbound := c.Query("upperbound")

	if sortBy == "security" {
		var lastRandIdArray []string
		lastRandId := c.Query("lastRandId")
		if lastRandId != "" {
			lastRandIdArray = strings.Split(lastRandId, ",")
		}

		tickets, validLastRandId, position, isSeedingRequired, errFetch := fh.ticketService.GetTicketsBySecurityRisk(mainCtx, lastRandIdArray)
		if errFetch != nil {
			if errors.Is(errFetch, redifu.ResetPagination) {
				lastRandIdArray = []string{}
			} else {
				return logger.Error(c, fiber.StatusInternalServerError, errFetch, "T500", "GetTicketsBySecurityRisk.Fetch")
			}
		}
		if isSeedingRequired {
			errSeedTicketTimeline := fh.seedHandler.SeedTicketBySecurityRisk(mainCtx, int64(len(tickets)), validLastRandId)
			if errSeedTicketTimeline != nil {
				return logger.Error(c, fiber.StatusInternalServerError, errSeedTicketTimeline, "T500", "GetTicketsBySecurityRisk.Seed")
			}

			tickets, validLastRandId, position, isSeedingRequired, errFetch = fh.ticketService.GetTicketsBySecurityRisk(mainCtx, lastRandIdArray)
			if errFetch != nil {
				return logger.Error(c, fiber.StatusInternalServerError, errFetch, "T500", "GetTicketsBySecurityRiskAfterSeed.Fetch")
			}
		}

		c.Set("Content-Type", "application/json")
		return c.JSON(map[string]interface{}{
			"position": position,
			"tickets":  tickets,
		})
	} else if lowerbound != "" && upperbound != "" {
		lowerboundAsTime, errParse := time.Parse(time.RFC3339, lowerbound)
		upperboundAsTime, errParse := time.Parse(time.RFC3339, upperbound)
		if errParse != nil {
			return logger.Error(c, fiber.StatusBadRequest, errors.New("incorrect lowerbound value-type"), "T100", "GetTicketsByDate.Parse")
		}

		tickets, seedRequired, errFetch := fh.ticketService.GetTicketsByDate(mainCtx, lowerboundAsTime, upperboundAsTime)
		if errFetch != nil {
			return logger.Error(c, fiber.StatusInternalServerError, errFetch, "T500", "GetTicketsByDate.Fetch")
		}
		if seedRequired {
			errSeedTicketByDate := fh.seedHandler.SeedTicketsByDate(mainCtx, lowerboundAsTime, upperboundAsTime)
			if errSeedTicketByDate != nil {
				return logger.Error(c, fiber.StatusInternalServerError, errSeedTicketByDate, "T500", "GetTicketsByDate.Seed")
			}

			tickets, seedRequired, errFetch = fh.ticketService.GetTicketsByDate(mainCtx, lowerboundAsTime, upperboundAsTime)
			if errFetch != nil {
				return logger.Error(c, fiber.StatusInternalServerError, errFetch, "T500", "GetTicketsByDateAfterSeed.Fetch")
			}
		}

		c.Set("Content-Type", "application/json")
		return c.JSON(tickets)

	} else {
		if page != "" {
			pageAsInt, errParse := strconv.ParseInt(page, 10, 64)
			if errParse != nil {
				return logger.Error(c, fiber.StatusBadRequest, errors.New("incorrect page number value-type"), "T100", "GetTicketsByPage.Parse")
			}
			tickets, seedRequired, errFetch := fh.ticketService.GetTicketsByPage(mainCtx, pageAsInt)
			if errFetch != nil {
				return logger.Error(c, fiber.StatusInternalServerError, errFetch, "T500", "GetTicketsByPage.Fetch")
			}
			if seedRequired {
				errSeedTicketByPage := fh.seedHandler.SeedTicketsByPage(mainCtx, pageAsInt)
				if errSeedTicketByPage != nil {
					return logger.Error(c, fiber.StatusInternalServerError, errSeedTicketByPage, "T500", "GetTicketsByPage.Seed")
				}

				tickets, seedRequired, errFetch = fh.ticketService.GetTicketsByPage(mainCtx, pageAsInt)
				if errFetch != nil {
					return logger.Error(c, fiber.StatusInternalServerError, errFetch, "T500", "GetTicketsByPage.Fetch")
				}
			}

			c.Set("Content-Type", "application/json")
			return c.JSON(tickets)
		} else {
			var lastRandIdArray []string
			lastRandId := c.Query("lastRandId")
			if lastRandId != "" {
				lastRandIdArray = strings.Split(lastRandId, ",")
			}

			ticket, validLastRandId, position, isSeedingRequired, errFetch := fh.ticketService.GetTickets(mainCtx, lastRandIdArray)
			if errFetch != nil {
				if errors.Is(errFetch, redifu.ResetPagination) {
					lastRandIdArray = []string{}
				} else {
					return logger.Error(c, fiber.StatusInternalServerError, errFetch, "T500", "GetTickets.Fetch")
				}
			}

			if isSeedingRequired {
				errSeedTicketTimeline := fh.seedHandler.SeedTickets(mainCtx, int64(len(ticket)), validLastRandId)
				if errSeedTicketTimeline != nil {
					return logger.Error(c, fiber.StatusInternalServerError, errSeedTicketTimeline, "T500", "GetTickets.Seed")
				}

				ticket, validLastRandId, position, isSeedingRequired, errFetch = fh.ticketService.GetTickets(mainCtx, lastRandIdArray)
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
	}
}

func (fh *TicketFetchController) GetTicketsByReporter(c *fiber.Ctx) error {
	mainCtx := c.Context()
	accountUUID := c.Params("accountUUID")
	ticket, requireSeeding, errFetch := fh.ticketService.GetTicketsByReporter(mainCtx, accountUUID)
	if errFetch != nil {
		return logger.Error(c, fiber.StatusInternalServerError, errFetch, "T500", "GetTicketSorted.Fetch")
	}
	if requireSeeding {
		errSeedTicketSorted := fh.seedHandler.SeedByAccount(mainCtx, accountUUID)
		if errSeedTicketSorted != nil {
			return logger.Error(c, fiber.StatusInternalServerError, errSeedTicketSorted, "T500", "GetTicketSorted.Seed")
		}

		ticket, requireSeeding, errFetch = fh.ticketService.GetTicketsByReporter(mainCtx, accountUUID)
		if errFetch != nil {
			return logger.Error(c, fiber.StatusInternalServerError, errFetch, "T500", "GetTicketSortedAfterSeed.Fetch")
		}
	}

	return c.JSON(ticket)
}

func NewTicketFetchController(redisClient redis.UniversalClient, seeder TicketSeeder) *TicketFetchController {
	ticketService := service.NewTicketService()
	ticketService.InitFetcher(redisClient)

	return &TicketFetchController{
		ticketService: ticketService,
		seedHandler:   seeder,
	}
}

type TicketSeeder interface {
	SeedTickets(context.Context, int64, string) error
	SeedTicketBySecurityRisk(context.Context, int64, string) error
	SeedByAccount(context.Context, string) error
	SeedTicket(context.Context, string) error
	SeedTicketsByPage(ctx context.Context, page int64) error
	SeedTicketsByDate(ctx context.Context, lowerbound time.Time, upperbound time.Time) error
}

type TicketSeedHandler struct {
	ticketService *service.TicketService
}

func (sh *TicketSeedHandler) SeedTickets(ctx context.Context, subtraction int64, lastRandId string) error {
	return sh.ticketService.SeedTickets(ctx, subtraction, lastRandId)
}

func (sh *TicketSeedHandler) SeedTicketBySecurityRisk(ctx context.Context, subtraction int64, lastRandId string) error {
	return sh.ticketService.SeedTicketsBySecurityRisk(ctx, subtraction, lastRandId)
}

func (sh *TicketSeedHandler) SeedByAccount(ctx context.Context, accountUUID string) error {
	return sh.ticketService.SeedTicketsByAccount(ctx, accountUUID)
}

func (sh *TicketSeedHandler) SeedTicket(ctx context.Context, randId string) error {
	return sh.ticketService.SeedTicket(ctx, randId)
}

func (sh *TicketSeedHandler) SeedTicketsByPage(ctx context.Context, page int64) error {
	return sh.ticketService.SeedTicketsByPage(ctx, page)
}

func (sh *TicketSeedHandler) SeedTicketsByDate(ctx context.Context, lowerbound time.Time, upperbound time.Time) error {
	return sh.ticketService.SeedTicketsByDate(ctx, lowerbound, upperbound)
}

func NewSelfSeedHandler(db *sql.DB, redisClient redis.UniversalClient) *TicketSeedHandler {
	ticketService := service.NewTicketService()
	accountService := service.NewAccountService()
	accountService.InitRepository(db, redisClient)

	ticketService.InitRepository(db, redisClient, accountService)
	return &TicketSeedHandler{
		ticketService: ticketService,
	}
}

// Implement type GRPCSeedHandler here
