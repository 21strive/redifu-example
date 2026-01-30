package main

import (
	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"
	"os"
	"redifu-example/api"
	"redifu-example/api/controller"
	"redifu-example/internal/fetcher"
	"redifu-example/internal/pools"
	"redifu-example/internal/repository"
	"redifu-example/pkg/account"
	"redifu-example/pkg/ticket"
	"redifu-example/pkg/utils"
)

func InitSetterOnly() {
	app := fiber.New()
	db := utils.CreatePostgresConnection(os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_SSLMODE"))
	redisClient := utils.ConnectRedis(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_USER"),
		os.Getenv("REDIS_PASS"), false)

	fetcherPool := pools.NewFetcherPool(redisClient)
	seederPool := pools.NewSeederPool()

	seederPool.InitTicketSeeder(redisClient, db, fetcherPool.BaseTicket, fetcherPool.Timeline)
	seederPool.InitTicketBySecurityRiskSeeder(redisClient, db, fetcherPool.BaseTicket, fetcherPool.TimelineSortBySecurityRisk)
	seederPool.InitTicketByCategorySeeder(redisClient, db, fetcherPool.BaseTicket, fetcherPool.TimelineByCategory)
	seederPool.InitTicketByAccountSeeder(redisClient, db, fetcherPool.BaseTicket, fetcherPool.SortedByAccount)
	seederPool.InitTicketPageSeeder(redisClient, db, fetcherPool.BaseTicket, fetcherPool.Page)
	seederPool.InitTicketTimeSeriesSeeder(redisClient, db, fetcherPool.BaseTicket, fetcherPool.TimeSeries)

	ticketRepo := repository.NewTicketRepository(db, fetcherPool, seederPool)
	accountRepo := repository.NewAccountRepository(db, redisClient, fetcherPool)
	categoryRepo := repository.NewCategoryRepository(db)

	ticketService := ticket.NewTicketService()
	accountService := account.NewAccountService()

	ticketService.InitRepository(ticketRepo, categoryRepo, accountService)
	accountService.InitRepository(accountRepo)

	api.SetterEndpoints(app, ticketService, accountService)
	app.Listen(":" + os.Getenv("RUNNING_PORT"))
}

func InitGetterOnly() {
	app := fiber.New()
	db := utils.CreatePostgresConnection(os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_SSLMODE"))
	redisClient := utils.ConnectRedis(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_USER"),
		os.Getenv("REDIS_PASS"), false)

	fetcherPool := pools.NewFetcherPool(redisClient)
	categoryRepo := repository.NewCategoryRepository(db)

	accountFetcher := fetcher.NewAccountFetcher(redisClient, fetcherPool)
	ticketFetcher := fetcher.NewTicketFetcher(fetcherPool)

	ticketService := ticket.NewTicketService()
	accountService := account.NewAccountService()

	ticketService.InitRepository(nil, categoryRepo, nil)
	ticketService.InitFetcher(ticketFetcher)
	accountService.InitFetcher(accountFetcher)

	api.GetterEndpoints(app, ticketService, nil)
	app.Listen(":" + os.Getenv("RUNNING_PORT"))
}

func Init() {
	app := fiber.New()
	db := utils.CreatePostgresConnection(os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_SSLMODE"))
	redisClient := utils.ConnectRedis(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_USER"),
		os.Getenv("REDIS_PASS"), false)

	fetcherPool := pools.NewFetcherPool(redisClient)
	seederPool := pools.NewSeederPool()

	seederPool.InitTicketSeeder(redisClient, db, fetcherPool.BaseTicket, fetcherPool.Timeline)
	seederPool.InitTicketBySecurityRiskSeeder(redisClient, db, fetcherPool.BaseTicket, fetcherPool.TimelineSortBySecurityRisk)
	seederPool.InitTicketByCategorySeeder(redisClient, db, fetcherPool.BaseTicket, fetcherPool.TimelineByCategory)
	seederPool.InitTicketByAccountSeeder(redisClient, db, fetcherPool.BaseTicket, fetcherPool.SortedByAccount)
	seederPool.InitTicketPageSeeder(redisClient, db, fetcherPool.BaseTicket, fetcherPool.Page)
	seederPool.InitTicketTimeSeriesSeeder(redisClient, db, fetcherPool.BaseTicket, fetcherPool.TimeSeries)

	ticketRepo := repository.NewTicketRepository(db, fetcherPool, seederPool)
	accountRepo := repository.NewAccountRepository(db, redisClient, fetcherPool)
	categoryRepo := repository.NewCategoryRepository(db)
	accountFetcher := fetcher.NewAccountFetcher(redisClient, fetcherPool)
	ticketFetcher := fetcher.NewTicketFetcher(fetcherPool)

	ticketService := ticket.NewTicketService()
	accountService := account.NewAccountService()

	ticketService.InitRepository(ticketRepo, categoryRepo, accountService)
	ticketService.InitFetcher(ticketFetcher)
	accountService.InitRepository(accountRepo)
	accountService.InitFetcher(accountFetcher)

	api.SetterEndpoints(app, ticketService, accountService)

	var seedHandler controller.TicketSeeder
	if os.Getenv("OP_MODE") == "" {
		seedHandler = controller.NewSelfSeedHandler(ticketService)
	} else if os.Getenv("OP_MODE") == "GETTER" {
		// seedHandler = GRPCHandler
	}

	api.GetterEndpoints(app, ticketService, seedHandler)
	app.Listen(":" + os.Getenv("RUNNING_PORT"))
}

func StartAPI() {
	if os.Getenv("OP_MODE") == "SETTER" {
		InitSetterOnly()
	} else if os.Getenv("OP_MODE") == "GETTER" {
		InitGetterOnly()
	} else if os.Getenv("OP_MODE") == "" {
		Init()
	}
}

func main() {
	StartAPI()
}
