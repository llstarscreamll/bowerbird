package infra

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"llstarscreamll/bowerbird/internal/auth/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGoogleLogin(t *testing.T) {
	mux := http.NewServeMux()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/v1/auth/google/login", nil)

	ulidMock := new(MockULID)
	ulidMock.AssertNotCalled(t, "New")

	userRepoMock := new(MockUserRepository)
	userRepoMock.AssertNotCalled(t, "Create")

	authServerMock := new(MockAuthService)
	authServerMock.On("GetLoginUrl").Return("https://some-google.com/auth/login")

	sessionRepoMock := new(MockSessionRepository)
	sessionRepoMock.AssertNotCalled(t, "Save")

	RegisterRoutes(mux, config, ulidMock, authServerMock, userRepoMock, sessionRepoMock)
	mux.ServeHTTP(w, r)

	response := w.Result()
	defer response.Body.Close()

	assert.Equal(t, http.StatusFound, response.StatusCode)
	ulidMock.AssertExpectations(t)
	userRepoMock.AssertExpectations(t)
	authServerMock.AssertExpectations(t)
}

func TestGoogleLoginCallback(t *testing.T) {
	testCases := []struct {
		testCase              string
		verb                  string
		endpoint              string
		ulidMock              func() *MockULID
		authServerMock        func() *MockAuthService
		userRepoMock          func() *MockUserRepository
		sessionRepositoryMock func() *MockSessionRepository
		expectStatusCode      int
		expectedHeaders       map[string]string
	}{
		{
			"should return 302 when callback succeeds",
			"GET", "/v1/auth/google/callback?code=123",
			func() *MockULID {
				m := new(MockULID)
				m.On("New").Return("01JGCA8BBB00000000000000U1").Once()
				m.On("NewFromDate", mock.AnythingOfType(fmt.Sprintf("%T", time.Now()))).Return("01JGCA8BBB00000000000000S1", nil).Once()
				return m
			},
			func() *MockAuthService {
				m := new(MockAuthService)
				m.On("GetUserInfo", mock.AnythingOfType(fmt.Sprintf("%T", context.Background())), "123").Return(testUser, nil)
				return m
			},
			func() *MockUserRepository {
				m := new(MockUserRepository)
				m.On("Upsert", mock.AnythingOfType(fmt.Sprintf("%T", context.Background())), mock.MatchedBy(func(u domain.User) bool {
					return assert.Equal(t, "01JGCA8BBB00000000000000U1", u.ID) &&
						assert.Equal(t, testUser.Email, u.Email) &&
						assert.Equal(t, testUser.EmailVerified, u.EmailVerified) &&
						assert.Equal(t, testUser.Name, u.Name) &&
						assert.Equal(t, testUser.GivenName, u.GivenName) &&
						assert.Equal(t, testUser.FamilyName, u.FamilyName) &&
						assert.Equal(t, testUser.PictureUrl, u.PictureUrl)
				})).Return(nil)
				return m
			},
			func() *MockSessionRepository {
				m := new(MockSessionRepository)
				m.On("Save", mock.AnythingOfType(fmt.Sprintf("%T", context.Background())), "01JGCA8BBB00000000000000S1", "01JGCA8BBB00000000000000U1", mock.Anything).Return(nil)
				return m
			},
			http.StatusFound,
			map[string]string{
				"Location":   config.FrontendUrl + "/dashboard",
				"Set-Cookie": "session_token=01JGCA8BBB00000000000000S1; Path=/; HttpOnly; Secure",
			},
		},
		{
			"should return 400 when auth code is empty",
			"GET",
			"/v1/auth/google/callback?code=",
			func() *MockULID {
				m := new(MockULID)
				m.AssertNotCalled(t, "New")
				return m
			},
			func() *MockAuthService {
				m := new(MockAuthService)
				m.AssertNotCalled(t, "GetUserInfo")
				return m
			},
			func() *MockUserRepository {
				m := new(MockUserRepository)
				m.AssertNotCalled(t, "Upsert")
				return m
			},
			func() *MockSessionRepository {
				m := new(MockSessionRepository)
				m.AssertNotCalled(t, "Save")
				return m
			},
			http.StatusBadRequest,
			map[string]string{},
		},
		{
			"should return 500 when user info can't be retrieved",
			"GET",
			"/v1/auth/google/callback?code=123",
			func() *MockULID {
				m := new(MockULID)
				m.AssertNotCalled(t, "New")
				return m
			},
			func() *MockAuthService {
				m := new(MockAuthService)
				m.On("GetUserInfo", mock.Anything, "123").Return(domain.User{}, assert.AnError)
				return m
			},
			func() *MockUserRepository {
				m := new(MockUserRepository)
				m.AssertNotCalled(t, "Upsert")
				return m
			},
			func() *MockSessionRepository {
				m := new(MockSessionRepository)
				m.AssertNotCalled(t, "Save")
				return m
			},
			http.StatusInternalServerError,
			map[string]string{},
		},
		{
			"should return 500 when user info can't be saved",
			"GET",
			"/v1/auth/google/callback?code=123",
			func() *MockULID {
				m := new(MockULID)
				m.On("New").Return("01JGCA8BBB00000000000000U1").Once()
				return m
			},
			func() *MockAuthService {
				m := new(MockAuthService)
				m.On("GetUserInfo", mock.AnythingOfType(fmt.Sprintf("%T", context.Background())), "123").Return(testUser, nil)
				return m
			},
			func() *MockUserRepository {
				m := new(MockUserRepository)
				m.On("Upsert", mock.Anything, mock.Anything).Return(assert.AnError)
				return m
			},
			func() *MockSessionRepository {
				m := new(MockSessionRepository)
				m.AssertNotCalled(t, "Save")
				return m
			},
			http.StatusInternalServerError,
			map[string]string{},
		},
		{
			"should return 500 when session ID can't be generated",
			"GET",
			"/v1/auth/google/callback?code=123",
			func() *MockULID {
				m := new(MockULID)
				m.On("New").Return("01JGCA8BBB00000000000000U1").Once()
				m.On("NewFromDate", mock.AnythingOfType(fmt.Sprintf("%T", time.Now()))).Return("", assert.AnError).Once()
				return m
			},
			func() *MockAuthService {
				m := new(MockAuthService)
				m.On("GetUserInfo", mock.AnythingOfType(fmt.Sprintf("%T", context.Background())), "123").Return(testUser, nil)
				return m
			},
			func() *MockUserRepository {
				m := new(MockUserRepository)
				m.On("Upsert", mock.Anything, mock.Anything).Return(nil)
				return m
			},
			func() *MockSessionRepository {
				m := new(MockSessionRepository)
				m.AssertNotCalled(t, "Save")
				return m
			},
			http.StatusInternalServerError,
			map[string]string{},
		},
		{
			"should return 500 when session can't be saved",
			"GET",
			"/v1/auth/google/callback?code=123",
			func() *MockULID {
				m := new(MockULID)
				m.On("New").Return("01JGCA8BBB00000000000000U1").Once()
				m.On("NewFromDate", mock.AnythingOfType(fmt.Sprintf("%T", time.Now()))).Return("01JGCA8BBB00000000000000S1", nil).Once()
				return m
			},
			func() *MockAuthService {
				m := new(MockAuthService)
				m.On("GetUserInfo", mock.AnythingOfType(fmt.Sprintf("%T", context.Background())), "123").Return(testUser, nil)
				return m
			},
			func() *MockUserRepository {
				m := new(MockUserRepository)
				m.On("Upsert", mock.Anything, mock.Anything).Return(nil)
				return m
			},
			func() *MockSessionRepository {
				m := new(MockSessionRepository)
				m.On("Save", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)
				return m
			},
			http.StatusInternalServerError,
			map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testCase, func(t *testing.T) {
			mux := http.NewServeMux()
			w := httptest.NewRecorder()
			r := httptest.NewRequest(tc.verb, tc.endpoint, nil)

			ulidMock := tc.ulidMock()
			userRepoMock := tc.userRepoMock()
			authServerMock := tc.authServerMock()
			sessionRepoMock := tc.sessionRepositoryMock()

			RegisterRoutes(mux, config, ulidMock, authServerMock, userRepoMock, sessionRepoMock)
			mux.ServeHTTP(w, r)

			response := w.Result()
			defer response.Body.Close()

			assert.Equal(t, tc.expectStatusCode, response.StatusCode, "unexpected status code %d, expected %d", response.StatusCode, tc.expectStatusCode)

			for headerName, headerValue := range tc.expectedHeaders {
				assert.Contains(t, response.Header, headerName)
				assert.Equal(t, headerValue, response.Header.Get(headerName))
			}

			ulidMock.AssertExpectations(t)
			userRepoMock.AssertExpectations(t)
			authServerMock.AssertExpectations(t)
			sessionRepoMock.AssertExpectations(t)
		})
	}
}
