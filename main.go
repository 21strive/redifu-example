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
	"strings"
	"time"
)

var Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func CreatePostgresConnection() *sql.DB {
	// Environment variables approach (recommended)
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	sslmode := os.Getenv("DB_SSLMODE")

	// Build connection string
	connectionString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, dbname, sslmode)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to open database connection: %w", err))
	}

	if err := db.Ping(); err != nil {
		db.Close()
		log.Fatal(fmt.Errorf("failed to ping database: %w", err))
	}

	db.SetMaxOpenConns(25)                 // Maximum number of open connections
	db.SetMaxIdleConns(5)                  // Maximum number of idle connections
	db.SetConnMaxLifetime(5 * time.Minute) // Maximum connection lifetime

	log.Println("Successfully connected to PostgreSQL database")
	return db
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
	db, errConnectDB := CreatePostgresConnection()
	if errConnectDB != nil {
		log.Fatal(errConnectDB)
	}
	redis := ConnectRedis(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PASSWORD"), false)

	ticketRepository := lib.NewTicketRepository(db, redis)
	ticketFetcher := lib.NewTicketFetcher(redis)

	app := fiber.New()

	app.Post("/ticket", func(c *fiber.Ctx) error {
		var reqBody request.CreateTicketRequest
		if err := c.BodyParser(&reqBody); err != nil {
			return ConstructErrorResponse(c, "ticket", fiber.StatusBadRequest, err, "T100", "CreateTicket.BodyParser")
		}

		errCreate := CreateTicket(reqBody, ticketRepository)
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

		errUpdate := UpdateDescription(reqBody, ticketRepository)
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

		errResolve := ResolveTicket(reqBody.TicketUUID, ticketRepository)
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

		errDelete := DeleteTicket(request.UpdateTicketDescriptionRequest{TicketUUID: ticketUUID}, ticketRepository)
		if errDelete != nil {
			return ConstructErrorResponse(c, "ticket", errDelete.Status, errDelete.Error, errDelete.Code, "DeleteTicket.Delete")
		}
		return c.SendStatus(fiber.StatusOK)
	})

	// Get individual ticket
	app.Get("/ticket/:ticketRandId", func(c *fiber.Ctx) error {
		ticketRandId := c.Params("ticketRandId")
		if ticketRandId == "" {
			return ConstructErrorResponse(c, "ticket", fiber.StatusBadRequest, fmt.Errorf("ticketRandId is empty"), "T100", "GetTicket.Params")
		}

		ticket, isBlank, errFetch := Fetch(ticketRandId, ticketFetcher)
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

				ticket, isBlank, errFetch = Fetch(ticketRandId, ticketFetcher)
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

	app.Get("/ticket/timeline", func(c *fiber.Ctx) error {
		var lastRandIdArray []string
		lastRandId := c.Query("lastRandId")
		if lastRandId != "" {
			lastRandIdArray = strings.Split(lastRandId, ",")
		}

		ticket, validLastRandId, position, isSeedingRequired, errFetch := FetchTimeline(lastRandIdArray, ticketFetcher)
		if errFetch != nil {
			return ConstructErrorResponse(c, "ticket", errFetch.Status, errFetch.Error, errFetch.Code, "GetTicketTimeline.Fetch")
		}

		if isSeedingRequired {
			errSeedTicketTimeline := ticketRepository.SeedTimeline(int64(len(ticket)), validLastRandId)
			if errSeedTicketTimeline != nil {
				return ConstructErrorResponse(c, "ticket", fiber.StatusInternalServerError, errSeedTicketTimeline, "T500", "GetTicketTimeline.Seed")
			}

			ticket, validLastRandId, position, isSeedingRequired, errFetch = FetchTimeline(lastRandIdArray, ticketFetcher)
			if errFetch != nil {
				return ConstructErrorResponse(c, "ticket", errFetch.Status, errFetch.Error, errFetch.Code, "GetTicketTimeline.Fetch")
			}
		}

		c.Set("Content-Type", "application/json")
		return c.JSON(map[string]interface{}{
			"position": position,
			"tickets":  ticket,
		})
	})

	app.Get("/ticket/timeline/:reporterUUID", func(c *fiber.Ctx) error {
		var lastRandIdArray []string
		lastRandId := c.Query("lastRandId")
		if lastRandId != "" {
			lastRandIdArray = strings.Split(lastRandId, ",")
		}
		reporterUUID := c.Params("reporterUUID")

		ticket, validLastRandId, position, isSeedingRequired, errFetch := FetchTimelineByReporter(lastRandIdArray, reporterUUID, ticketFetcher)
		if errFetch != nil {
			return ConstructErrorResponse(c, "ticket", errFetch.Status, errFetch.Error, errFetch.Code, "GetTicketTimeline.Fetch")
		}

		if isSeedingRequired {
			errSeedTicketTimeline := ticketRepository.SeedTimelineByReporter(int64(len(ticket)), validLastRandId, reporterUUID)
			if errSeedTicketTimeline != nil {
				return ConstructErrorResponse(c, "ticket", fiber.StatusInternalServerError, errSeedTicketTimeline, "T500", "GetTicketTimeline.Seed")
			}

			ticket, validLastRandId, position, isSeedingRequired, errFetch = FetchTimelineByReporter(lastRandIdArray, reporterUUID, ticketFetcher)
			if errFetch != nil {
				return ConstructErrorResponse(c, "ticket", errFetch.Status, errFetch.Error, errFetch.Code, "GetTicketTimeline.Fetch")
			}
		}

		c.Set("Content-Type", "application/json")
		return c.JSON(map[string]interface{}{
			"position": position,
			"tickets":  ticket,
		})
	})

	app.Get("/ticket/sorted", func(c *fiber.Ctx) error {
		ticket, requiredSeeding, errFetch := FetchSorted(ticketFetcher)
		if errFetch != nil {
			return ConstructErrorResponse(c, "ticket", errFetch.Status, errFetch.Error, errFetch.Code, "GetTicketSorted.Fetch")
		}
		if requiredSeeding {
			errSeedTicketSorted := ticketRepository.SeedSorted()
			if errSeedTicketSorted != nil {
				return ConstructErrorResponse(c, "ticket", fiber.StatusInternalServerError, errSeedTicketSorted, "T500", "GetTicketSorted.Seed")
			}

			ticket, requiredSeeding, errFetch = FetchSorted(ticketFetcher)
			if errFetch != nil {
				return ConstructErrorResponse(c, "ticket", errFetch.Status, errFetch.Error, errFetch.Code, "GetTicketSorted.Fetch")
			}
		}

		return c.JSON(ticket)
	})

	app.Get("/ticket/sorted/:reporterUUID", func(c *fiber.Ctx) error {
		reporterUUID := c.Params("reporterUUID")
		ticket, requireSeeding, errFetch := FetchSortedByReporter(reporterUUID, ticketFetcher)
		if errFetch != nil {
			return ConstructErrorResponse(c, "ticket", errFetch.Status, errFetch.Error, errFetch.Code, "GetTicketSorted.Fetch")
		}
		if requireSeeding {
			errSeedTicketSorted := ticketRepository.SeedSortedByReporter(reporterUUID)
			if errSeedTicketSorted != nil {
				return ConstructErrorResponse(c, "ticket", fiber.StatusInternalServerError, errSeedTicketSorted, "T500", "GetTicketSorted.Seed")
			}

			ticket, requireSeeding, errFetch = FetchSortedByReporter(reporterUUID, ticketFetcher)
			if errFetch != nil {
				return ConstructErrorResponse(c, "ticket", errFetch.Status, errFetch.Error, errFetch.Code, "GetTicketSorted.Fetch")
			}
		}

		return c.JSON(ticket)
	})

	app.Listen(":3000")
}
