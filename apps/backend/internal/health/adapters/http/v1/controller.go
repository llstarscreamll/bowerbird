package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/bowerbird/internal/health/application"
	"github.com/bowerbird/internal/health/domain"
	"github.com/bowerbird/internal/platform/http/api"
)

type Controller struct {
	app application.Application
}

func NewController(app application.Application) *Controller {
	return &Controller{app: app}
}

func (c *Controller) Health(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	health := c.app.Queries.CheckHealth.Execute(ctx)
	statusCode := http.StatusOK

	if health.Status == domain.StatusDegraded {
		statusCode = http.StatusServiceUnavailable
	}

	return api.Success(w, statusCode, health)
}
