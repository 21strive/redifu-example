package api

import (
	"database/sql"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"redifu-example/api/controller"
	"redifu-example/internal/service"
	"strings"
)

func SetterEndpoints(app *fiber.App, db *sql.DB, redisClient redis.UniversalClient) {
	cudController := controller.NewCUDController(db, redisClient)
	app.Post("/ticket", cudController.CreateTicket)
	app.Patch("/ticket", cudController.PatchTicket)
	app.Post("/ticket/resolve", cudController.ResolveTicket)
	app.Delete("/ticket/:ticketUUID", cudController.DeleteTicket)
}

func GetterEndpoints(app *fiber.App, redisClient redis.UniversalClient) error {
	fetchController := controller.NewFetchController(redisClient)
	app.Get("/ticket/timeline", fetchController.GetTicketTimeline)
	app.Get("/ticket/sorted/:reporterUUID", fetchController.GetTicketsByReporter)
	// Get individual ticket
	app.Get("/ticket/:ticketRandId", func(c *fiber.Ctx) error {
		ticketRandId := c.Params("ticketRandId")
		if ticketRandId == "" {
			return ConstructErrorResponse(c, "ticket", fiber.StatusBadRequest, fmt.Errorf("ticketRandId is empty"), "T100", "GetTicket.Params")
		}

		ticket, isBlank, errFetch := service.Fetch(ticketRandId, ticketFetcher)
		if errFetch != nil {
			return ConstructErrorResponse(c, "ticket", errFetch.Status, errFetch.Error, errFetch.Code, "GetTicket.Fetch")
		}
		if ticket == nil {
			if isBlank {
				return ConstructErrorResponse(c, "ticket", fiber.StatusNotFound, fmt.Errorf("ticket not found"), "T404", "GetTicket.NotFound")
			} else {
				errSeedTicket := ticketRepository.SeedByRandId(ticketRandId)
				if errSeedTicket != nil {
					return ConstructErrorResponse(c, "ticket", fiber.StatusInternalServerError, errSeedTicket, "T500", "GetTicket.Seed")
				}

				ticket, isBlank, errFetch = service.Fetch(ticketRandId, ticketFetcher)
				if errFetch != nil {
					return ConstructErrorResponse(c, "ticket", errFetch.Status, errFetch.Error, errFetch.Code, "GetTicket.Fetch")
				}
				if ticket == nil || isBlank {
					return ConstructErrorResponse(c, "ticket", fiber.StatusNotFound, fmt.Errorf("ticket not found"), "T404", "GetTicket.NotFound")
				}
			}
		}

		return c.JSON(ticket)
	})
}
