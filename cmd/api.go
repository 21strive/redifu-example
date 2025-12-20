package cmd

import (
	"github.com/gofiber/fiber/v2"
	"os"
	"redifu-example/api"
	"redifu-example/internal/cache"
	"redifu-example/internal/dbconn"
)

func InitSetterOnly() {
	app := fiber.App{}
	db := dbconn.CreatePostgresConnection()
	redis := cache.ConnectRedis(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PASSWORD"), false)

	api.SetterEndpoints(&app, db, redis)
	app.Listen(":" + os.Getenv("RUNNING_PORT"))
}

func InitGetterOnly() {
	app := fiber.App{}
	redisClient := cache.ConnectRedis(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PASSWORD"), false)

	api.GetterEndpoints(&app, redisClient)
	app.Listen(":" + os.Getenv("RUNNING_PORT"))
}

func Init() {
	app := fiber.App{}
	db := dbconn.CreatePostgresConnection()
	redis := cache.ConnectRedis(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PASSWORD"), false)

	api.SetterEndpoints(&app, db, redis)
	api.GetterEndpoints(&app, redis)
	app.Listen(":" + os.Getenv("RUNNING_PORT"))
}

func StartAPI() {
	if os.Getenv("OP_MODE") == "SETTER" {
		InitSetterOnly()
	} else if os.Getenv("OP_MODE") == "GETTER" {
		InitGetterOnly()
	} else if os.Getenv("OP_MODE") == "MONO" {
		Init()
	}
}
