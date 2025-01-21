package infra

import (
	"context"
	"testing"
	"time"

	"llstarscreamll/bowerbird/internal/auth/domain"
	commonDomain "llstarscreamll/bowerbird/internal/common/domain"

	"github.com/stretchr/testify/mock"
)

type MockAuthServer struct {
	mock.Mock
}

func (m *MockAuthServer) GetLoginUrl(scopes []string) string {
	args := m.Called(scopes)
	return args.String(0)
}

func (m *MockAuthServer) GetTokens(ctx context.Context, authCode string) (string, string, time.Time, error) {
	args := m.Called(ctx, authCode)
	return args.String(0), args.String(1), args.Get(2).(time.Time), args.Error(3)
}

func (m *MockAuthServer) GetUserInfo(ctx context.Context, authCode string) (domain.User, error) {
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

func (m *MockUserRepository) GetByID(ctx context.Context, userID string) (domain.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(domain.User), args.Error(1)
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

func (m *MockSessionRepository) Save(ctx context.Context, userID, sessionID string, expirationDate time.Time) error {
	args := m.Called(ctx, userID, sessionID, expirationDate)
	return args.Error(0)
}

func (m *MockSessionRepository) GetByID(ctx context.Context, ID string) (string, error) {
	args := m.Called(ctx, ID)
	return args.String(0), args.Error(1)
}

type MockCrypt struct {
	mock.Mock
}

func (m *MockCrypt) EncryptString(str string) (string, error) {
	args := m.Called(str)
	return args.String(0), args.Error(1)
}

type MockMailCredentialRepository struct {
	mock.Mock
}

func (m *MockMailCredentialRepository) Save(ctx context.Context, userID, mailProvider, accessToken, refreshToken string, expiresAt time.Time) error {
	args := m.Called(ctx, userID, mailProvider, accessToken, refreshToken, expiresAt)
	return args.Error(0)
}

func neverCalledMockUlid(t *testing.T) *MockULID {
	m := new(MockULID)
	m.AssertNotCalled(t, "New")
	return m
}

func neverCalledMockCrypt(t *testing.T) *MockCrypt {
	m := new(MockCrypt)
	m.AssertNotCalled(t, "EncryptString")
	return m
}

func neverCalledMockUserRepository(t *testing.T) *MockUserRepository {
	m := new(MockUserRepository)
	m.AssertNotCalled(t, "Upsert")
	return m
}

func neverCalledMockSessionRepository(t *testing.T) *MockSessionRepository {
	m := new(MockSessionRepository)
	m.AssertNotCalled(t, "Save")
	return m
}

func neverCalledMockMailSecretRepository(t *testing.T) *MockMailCredentialRepository {
	m := new(MockMailCredentialRepository)
	m.AssertNotCalled(t, "Save")
	return m
}

func neverCalledMockAuthService(t *testing.T) *MockAuthServer {
	m := new(MockAuthServer)
	m.AssertNotCalled(t, "Save")
	return m
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
