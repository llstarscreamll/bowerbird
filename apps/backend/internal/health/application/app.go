package application

import (
	"github.com/bowerbird/internal/health/application/queries"
)

type Application struct {
	Queries Queries
}

type Queries struct {
	CheckHealth *queries.CheckHealthQuery
}
