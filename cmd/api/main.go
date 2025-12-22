package main

import (
	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"
	"os"
	"redifu-example/api"
	"redifu-example/api/controller"
	"redifu-example/internal/cache"
	"redifu-example/internal/dbconn"
)

func InitSetterOnly() {
	app := fiber.New()
	db := dbconn.CreatePostgresConnection(os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_SSLMODE"))
	redisClient := cache.ConnectRedis(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_USER"),
		os.Getenv("REDIS_PASS"), false)

	api.SetterEndpoints(app, db, redisClient)
	app.Listen(":" + os.Getenv("RUNNING_PORT"))
}

func InitGetterOnly() {
	app := fiber.New()
	redisClient := cache.ConnectRedis(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_USER"),
		os.Getenv("REDIS_PASS"), false)

	api.GetterEndpoints(app, redisClient, nil)
	app.Listen(":" + os.Getenv("RUNNING_PORT"))
}

func Init() {
	app := fiber.New()
	db := dbconn.CreatePostgresConnection(os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_SSLMODE"))
	redisClient := cache.ConnectRedis(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_USER"),
		os.Getenv("REDIS_PASS"), false)

	api.SetterEndpoints(app, db, redisClient)

	var seedHandler controller.TicketSeeder
	if os.Getenv("OP_MODE") == "" {
		seedHandler = controller.NewSelfSeedHandler(db, redisClient)
	} else if os.Getenv("OP_MODE") == "GETTER" {
		// seedHandler = GRPCHandler
	}

	api.GetterEndpoints(app, redisClient, seedHandler)
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
