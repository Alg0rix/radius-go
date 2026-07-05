package runtime

import (
	"github.com/labstack/echo/v4"
)

type Envelope struct {
	Success bool   `json:"success"`
	Data    any    `json:"data"`
	Meta    any    `json:"meta"`
	Error   any    `json:"error"`
}

func OK(c echo.Context, data any) error {
	return c.JSON(200, Envelope{Success: true, Data: data})
}

func Created(c echo.Context, data any) error {
	return c.JSON(201, Envelope{Success: true, Data: data})
}

func Fail(c echo.Context, status int, code, message string, details any) error {
	d := details
	if status >= 500 {
		d = nil
	}
	return c.JSON(status, Envelope{
		Success: false,
		Error: map[string]any{
			"code":    code,
			"message": message,
			"details": d,
		},
	})
}