package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/21strive/item"
	"github.com/gofiber/fiber/v2"
	"log/slog"
	"os"
	"strings"
)

var Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

type ServiceError struct {
	Code string `json:"code"`
	ID   string `json:"id"`
}

func Error(c *fiber.Ctx, status int, error error, appCode string, source ...string) error {
	errorId := item.RandId()

	response := ServiceError{
		Code: appCode,
		ID:   errorId,
	}

	type LogEntry struct {
		json.RawMessage
	}

	var inputBody json.RawMessage
	if c.Request().Body() != nil && len(c.Request().Body()) > 0 {
		var compactJSON bytes.Buffer
		json.Compact(&compactJSON, c.Request().Body())
		inputBody = compactJSON.Bytes()
	}

	logEntry := LogEntry{inputBody}

	var sourceStr string
	if len(source) > 1 {
		sourceStr = strings.Join(source, ".")
	} else if len(source) == 1 {
		sourceStr = source[0]
	}

	var returnedError string
	if error != nil {
		returnedError = error.Error()
	} else {
		returnedError = errors.New("error").Error()
	}
	Logger.Error("endpoint-error",
		"component", "paystore", "source", sourceStr, "appCode", appCode,
		"error", returnedError, "ID", errorId, "input", logEntry)

	c.Set("Content-Type", "application/json")
	return c.Status(status).JSON(response)
}

func Info() {}
