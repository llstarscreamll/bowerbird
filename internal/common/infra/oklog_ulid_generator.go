package postgresql

import (
	"time"

	"github.com/oklog/ulid/v2"
	"golang.org/x/exp/rand"
)

type OklogULIDGenerator struct{}

func (u OklogULIDGenerator) New() string {
	return ulid.Make().String()
}

func (u OklogULIDGenerator) NewFromDate(date time.Time) (string, error) {
	entropy := rand.New(rand.NewSource(uint64(time.Now().UnixNano())))
	ms := ulid.Timestamp(date)

	ulid, err := ulid.New(ms, entropy)
	if err != nil {
		return "", err
	}

	return ulid.String(), nil
}
