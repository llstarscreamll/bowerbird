package infra

import (
	"context"
	"time"

	"llstarscreamll/bowerbird/internal/auth/domain"
	commonDomain "llstarscreamll/bowerbird/internal/common/domain"

	"github.com/stretchr/testify/mock"
)

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) GetLoginUrl() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockAuthService) GetUserInfo(ctx context.Context, authCode string) (domain.User, error) {
	args := m.Called(ctx, authCode)
	return args.Get(0).(domain.User), args.Error(1)
}

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Upsert(ctx context.Context, user domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

type MockULID struct {
	mock.Mock
}

func (m *MockULID) New() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockULID) NewFromDate(date time.Time) (string, error) {
	args := m.Called(date)
	return args.String(0), args.Error(1)
}

type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) Save(userID, sessionID string) error {
	args := m.Called(userID, sessionID)
	return args.Error(0)
}

var config = commonDomain.AppConfig{
	ServerPort:  ":8080",
	FrontendUrl: "http://localhost:4200",
}

var testUser = domain.User{
	ID:            "01JGCZXZEC00000000000000U1",
	Email:         "john@doe.com",
	EmailVerified: true,
	Name:          "John Doe",
	GivenName:     "John",
	FamilyName:    "Doe",
	PictureUrl:    "https://some-google.com/picture.jpg",
}
