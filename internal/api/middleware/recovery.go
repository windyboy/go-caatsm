package middleware

import (
	"net/http"

	"caatsm/pkg/utils"

	"github.com/labstack/echo/v4"
)

// Recovery returns a middleware that recovers from panics
func Recovery() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					log := utils.GetSugaredLogger()
					log.Errorf("Panic recovered: %v", r)

					err := echo.NewHTTPError(http.StatusInternalServerError, "Internal Server Error")
					c.Error(err)
				}
			}()

			return next(c)
		}
	}
}

