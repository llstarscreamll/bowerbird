package application

import (
	"github.com/atta/internal/health/application/queries"
)

type Application struct {
	Queries Queries
}

type Queries struct {
	CheckHealth *queries.CheckHealthQuery
}
