package api

import (
	"database/sql"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"redifu-example/api/controller"
)

func SetterEndpoints(app *fiber.App, db *sql.DB, redisClient redis.UniversalClient) {
	cudController := controller.NewTicketCUDController(db, redisClient)

	// Ticket management group
	ticketGroup := app.Group("/ticket")
	ticketGroup.Post("/", cudController.CreateTicket)
	ticketGroup.Patch("/", cudController.PatchTicket)
	ticketGroup.Post("/resolve", cudController.ResolveTicket)
	ticketGroup.Delete("/:ticketUUID", cudController.DeleteTicket)

	// Account management group
	accountGroup := app.Group("/account")
	accountController := controller.NewAccountCUDController(db, redisClient)
	accountGroup.Post("/", accountController.CreateAccount)
}

func GetterEndpoints(app *fiber.App, redisClient redis.UniversalClient, ticketSeeder controller.TicketSeeder) {
	fetchController := controller.NewTicketFetchController(redisClient, ticketSeeder)

	// Ticket retrieval group
	ticketGroup := app.Group("/ticket")
	ticketGroup.Get("/", fetchController.GetTickets)
	ticketGroup.Get("/account/:reporterUUID", fetchController.GetTicketsByReporter)
	ticketGroup.Get("/:ticketRandId", fetchController.GetTicket)
}
