package rest

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type HealthController struct {
	client *mongo.Client
}

// NewHealthController exposes liveness and readiness probes. Liveness only
// proves the process responds; readiness also checks the database, so
// orchestrators stop routing traffic when Mongo is gone.
func NewHealthController(e *echo.Echo, client *mongo.Client) *HealthController {
	controller := &HealthController{client: client}

	e.GET("/healthz", controller.Liveness)
	e.GET("/readyz", controller.Readiness)

	return controller
}

func (hc *HealthController) Liveness(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (hc *HealthController) Readiness(c echo.Context) error {
	if err := hc.client.Ping(c.Request().Context(), nil); err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"status": "unavailable",
			"reason": "database unreachable",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
