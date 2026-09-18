package domain

type Status string

const (
	StatusOK       Status = "ok"
	StatusDegraded Status = "degraded"
)

type Health struct {
	Status Status `json:"status"`
}
