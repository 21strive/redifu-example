package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/21strive/item"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"log"
	"log/slog"
	"os"
	"redifu-example/lib"
	"redifu-example/request"
	"time"
)

var Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func CreatePostgresConnection() (*sql.DB, error) {
	connStr := "dbname=paparazoo sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(25)                 // Maximum number of open connections
	db.SetMaxIdleConns(5)                  // Maximum number of idle connections
	db.SetConnMaxLifetime(5 * time.Minute) // Maximum connection lifetime

	log.Println("Successfully connected to PostgreSQL database")
	return db, nil
}

func ConnectRedis(redisHostAddr string, password string, isClustered bool) redis.UniversalClient {
	if redisHostAddr == "" {
		log.Fatal("REDIS_HOST environment variable not set")
	}

	if isClustered {
		clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    []string{redisHostAddr},
			Password: password,
		})

		_, err := clusterClient.Ping(context.Background()).Result()
		if err != nil {
			log.Fatal(err)
		}

		return clusterClient
	}

	client := redis.NewClient(&redis.Options{
		Addr:     redisHostAddr,
		Password: password,
		DB:       0,
	})

	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal(err)
	}

	return client
}

type ErrorResponse struct {
	Code string `json:"code"`
	ID   string `json:"id"`
}

func ConstructErrorResponse(c *fiber.Ctx, componentName string, status int, error error, code string, source string) error {
	errorId := item.RandId()

	response := ErrorResponse{
		Code: code,
		ID:   errorId,
	}

	var inputBody string
	if c.Request().Body() != nil && len(c.Request().Body()) > 0 {
		inputBody = string(c.Request().Body())
	}

	Logger.Error("endpoint-error", "component", componentName, "source", source, "code", code, "error", error.Error(), "ID", errorId, "input", inputBody)

	c.Set("Content-Type", "application/json")
	return c.Status(status).JSON(response)
}

func main() {

	app := fiber.New()

	app.Post("/ticket", func(c *fiber.Ctx) error {
		var reqBody request.CreateTicketRequest
		if err := c.BodyParser(&reqBody); err != nil {
			return ConstructErrorResponse(c, "ticket", fiber.StatusBadRequest, err, "T100", "CreateTicket.BodyParser")
		}

		errCreate := CreateTicket(reqBody, nil)
		if errCreate != nil {
			return ConstructErrorResponse(c, "ticket", errCreate.Status, errCreate.Error, errCreate.Code, "CreateTicket.Create")
		}

		return c.SendStatus(fiber.StatusCreated)
	})
	app.Patch("/ticket", func(c *fiber.Ctx) error {
		var reqBody request.UpdateTicketDescriptionRequest
		if err := c.BodyParser(&reqBody); err != nil {
			return ConstructErrorResponse(c, "ticket", fiber.StatusBadRequest, err, "T100", "UpdateTicketDescription.BodyParser")
		}

		errUpdate := UpdateDescription(reqBody, nil)
		if errUpdate != nil {
			return ConstructErrorResponse(c, "ticket", errUpdate.Status, errUpdate.Error, errUpdate.Code, "UpdateTicketDescription.Update")
		}
		return c.SendStatus(fiber.StatusOK)
	})
	app.Post("/ticket/resolve", func(c *fiber.Ctx) error {
		var reqBody request.UpdateTicketDescriptionRequest
		if err := c.BodyParser(&reqBody); err != nil {
			return ConstructErrorResponse(c, "ticket", fiber.StatusBadRequest, err, "T100", "UpdateTicketDescription.BodyParser")
		}

		errResolve := ResolveTicket(reqBody.TicketUUID, nil)
		if errResolve != nil {
			return ConstructErrorResponse(c, "ticket", errResolve.Status, errResolve.Error, errResolve.Code, "UpdateTicketDescription.Resolve")
		}
		return c.SendStatus(fiber.StatusOK)
	})
	app.Delete("/ticket/:ticketUUID", func(c *fiber.Ctx) error {
		ticketUUID := c.Params("ticketUUID")
		if ticketUUID == "" {
			return ConstructErrorResponse(c, "ticket", fiber.StatusBadRequest, fmt.Errorf("ticketUUID is empty"), "T100", "DeleteTicket.Params")
		}

		errDelete := DeleteTicket(request.UpdateTicketDescriptionRequest{TicketUUID: ticketUUID}, nil)
		if errDelete != nil {
			return ConstructErrorResponse(c, "ticket", errDelete.Status, errDelete.Error, errDelete.Code, "DeleteTicket.Delete")
		}
		return c.SendStatus(fiber.StatusOK)
	})
	app.Get("/ticket/:ticketUUID", func(c *fiber.Ctx) error {
		ticketUUID := c.Params("ticketUUID")
		if ticketUUID == "" {
			return ConstructErrorResponse(c, "ticket", fiber.StatusBadRequest, fmt.Errorf("ticketUUID is empty"), "T100", "GetTicket.Params")
		}
	})

	app.Listen(":3000")
}
