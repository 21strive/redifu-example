package controller

import (
	"github.com/gofiber/fiber/v2"
	"redifu-example/internal/logger"
	"redifu-example/pkg/account"
)

type CreateAccountRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type AccountController struct {
	accountService *account.AccountService
}

func (ac *AccountController) CreateAccount(c *fiber.Ctx) error {
	mainCtx := c.Context()

	var reqBody CreateAccountRequest
	if err := c.BodyParser(&reqBody); err != nil {
		return logger.Error(c, fiber.StatusBadRequest, err, "A100", "CreateAccount.BodyParser")
	}

	errCreate := ac.accountService.Create(mainCtx, reqBody.Name, reqBody.Email)
	if errCreate != nil {
		return logger.Error(c, fiber.StatusInternalServerError, errCreate, "A500", "CreateAccount.Create")
	}

	return c.SendStatus(fiber.StatusCreated)
}

func NewAccountCUDController(accountService *account.AccountService) *AccountController {
	return &AccountController{accountService: accountService}
}
