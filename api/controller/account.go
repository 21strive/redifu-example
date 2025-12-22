package controller

import (
	"database/sql"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"redifu-example/internal/logger"
	"redifu-example/pkg/service"
)

type CreateAccountRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type AccountController struct {
	accountService *service.AccountService
}

func (ac *AccountController) CreateAccount(c *fiber.Ctx) error {
	var reqBody CreateAccountRequest
	if err := c.BodyParser(&reqBody); err != nil {
		return logger.Error(c, fiber.StatusBadRequest, err, "A100", "CreateAccount.BodyParser")
	}

	errCreate := ac.accountService.Create(reqBody.Name, reqBody.Email)
	if errCreate != nil {
		return logger.Error(c, fiber.StatusInternalServerError, errCreate, "A500", "CreateAccount.Create")
	}

	return c.SendStatus(fiber.StatusCreated)
}

func NewAccountCUDController(db *sql.DB, redisClient redis.UniversalClient) *AccountController {
	accountService := service.NewAccountService()
	accountService.InitRepository(db, redisClient)

	return &AccountController{accountService: accountService}
}
