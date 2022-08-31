package api

import (
	oapi "github.com/swaggo/echo-swagger"

	"github.com/labstack/echo/v4"
)

const apiVersion = "v3"

// @title eth-analyse-service API
// @version 1
// @description Api for eth-anaylse
// @termsOfService http://swagger.io/terms/

// @contact.name Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host web-api.local
// @BasePath /v2
func New() *echo.Echo {
	e := echo.New()

	// Add Middleware..

	// Register Routes..

	// Initialize Database..

	e.GET("/swagger/*", oapi.WrapHandler)

	return e
}
