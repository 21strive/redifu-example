package api

import (
	"database/sql"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"redifu-example/api/controller"
)

func SetterEndpoints(app *fiber.App, db *sql.DB, redisClient redis.UniversalClient) {
	cudController := controller.NewTicketCUDController(db, redisClient)
	app.Post("/ticket", cudController.CreateTicket)
	app.Patch("/ticket", cudController.PatchTicket)
	app.Post("/ticket/resolve", cudController.ResolveTicket)
	app.Delete("/ticket/:ticketUUID", cudController.DeleteTicket)

	accountController := controller.NewAccountCUDController(db, redisClient)
	app.Post("/account", accountController.CreateAccount)
}

func GetterEndpoints(app *fiber.App, redisClient redis.UniversalClient) {
	fetchController := controller.NewTicketFetchController(redisClient)
	app.Get("/ticket/timeline", fetchController.GetTickets)
	app.Get("/ticket/sorted/:reporterUUID", fetchController.GetTicketsByReporter)
	// Get individual ticket
	app.Get("/ticket/:ticketRandId", fetchController.GetTicket)
}
