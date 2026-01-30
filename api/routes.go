package api

import (
	"github.com/gofiber/fiber/v2"
	"redifu-example/api/controller"
	"redifu-example/pkg/account"
	"redifu-example/pkg/ticket"
)

func SetterEndpoints(app *fiber.App, ticketService *ticket.TicketService, accountService *account.AccountService) {
	cudController := controller.NewTicketCUDController(ticketService)

	// Ticket management group
	ticketGroup := app.Group("/ticket")
	ticketGroup.Post("/", cudController.CreateTicket)
	ticketGroup.Patch("/", cudController.PatchTicket)
	ticketGroup.Post("/resolve", cudController.ResolveTicket)
	ticketGroup.Delete("/:ticketUUID", cudController.DeleteTicket)

	// Account management group
	accountGroup := app.Group("/account")
	accountController := controller.NewAccountCUDController(accountService)
	accountGroup.Post("/", accountController.CreateAccount)
}

func GetterEndpoints(app *fiber.App, ticketService *ticket.TicketService, ticketSeeder controller.TicketSeeder) {
	fetchController := controller.NewTicketFetchController(ticketService, ticketSeeder)

	// Ticket retrieval group
	ticketGroup := app.Group("/ticket")
	ticketGroup.Get("/", fetchController.GetTickets)
	ticketGroup.Get("/account/:reporterUUID", fetchController.GetTicketsByReporter)
	ticketGroup.Get("/:ticketRandId", fetchController.GetTicket)
}
