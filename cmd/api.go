package cmd

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/21strive/redifu"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"os"
	"redifu-example/internal/service"
	"redifu-example/pkg/cache"
	"redifu-example/pkg/dbconn"
	"redifu-example/pkg/logger"
	"redifu-example/request"
	"redifu-example/routes"
	"strings"
)

func InitSetterOnly() {
	app := fiber.App{}
	db := dbconn.CreatePostgresConnection()
	redis := cache.ConnectRedis(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PASSWORD"), false)

	routes.SetterEndpoints(&app, db, redis)
	app.Listen(":" + os.Getenv("RUNNING_PORT"))
}

func InitGetterOnly() {
	app := fiber.App{}
	redisClient := cache.ConnectRedis(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PASSWORD"), false)

	routes.GetterEndpoints(&app, redisClient)
	app.Listen(":" + os.Getenv("RUNNING_PORT"))
}

func Init() {
	app := fiber.App{}
	db := dbconn.CreatePostgresConnection()
	redis := cache.ConnectRedis(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PASSWORD"), false)

	routes.SetterEndpoints(&app, db, redis)
	routes.GetterEndpoints(&app, redis)
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
