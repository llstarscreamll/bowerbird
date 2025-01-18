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

var bgContextType = mock.AnythingOfType(fmt.Sprintf("%T", context.Background()))

func TestGoogleLogin(t *testing.T) {
	mux := http.NewServeMux()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/v1/auth/google/login", nil)

	ulidMock := neverCalledMockUlid(t)
	cryptMock := neverCalledMockCrypt(t)
	userRepoMock := neverCalledMockUserRepository(t)
	sessionRepoMock := neverCalledMockSessionRepository(t)
	mailSecretRepoMock := neverCalledMockMailSecretRepository(t)

	authServerMock := new(MockAuthServer)
	authServerMock.On("GetLoginUrl", []string{}).Return("https://some-google.com/auth/login")

	RegisterRoutes(mux, config, ulidMock, authServerMock, userRepoMock, sessionRepoMock, cryptMock, mailSecretRepoMock)
	mux.ServeHTTP(w, r)

	response := w.Result()
	defer response.Body.Close()

	assert.Equal(t, http.StatusFound, response.StatusCode)
	assert.Equal(t, []string{"https://some-google.com/auth/login"}, response.Header["Location"])

	ulidMock.AssertExpectations(t)
	cryptMock.AssertExpectations(t)
	userRepoMock.AssertExpectations(t)
	authServerMock.AssertExpectations(t)
	sessionRepoMock.AssertExpectations(t)
	mailSecretRepoMock.AssertExpectations(t)
}

func TestGoogleLoginCallback(t *testing.T) {
	testCases := []struct {
		testCase              string
		verb                  string
		endpoint              string
		ulidMock              func(t *testing.T) *MockULID
		authServerMock        func(t *testing.T) *MockAuthServer
		userRepoMock          func(t *testing.T) *MockUserRepository
		sessionRepositoryMock func(t *testing.T) *MockSessionRepository
		expectStatusCode      int
		expectedHeaders       map[string]string
	}{
		{
			"should return 302 when callback succeeds",
			"GET", "/v1/auth/google/callback?code=123",
			func(t *testing.T) *MockULID {
				m := new(MockULID)
				m.On("New").Return("01JGCA8BBB00000000000000U1").Once()
				m.On("NewFromDate", mock.AnythingOfType(fmt.Sprintf("%T", time.Now()))).Return("01JGCA8BBB00000000000000S1", nil).Once()
				return m
			},
			func(t *testing.T) *MockAuthServer {
				m := new(MockAuthServer)
				m.On("GetUserInfo", bgContextType, "123").Return(testUser, nil)
				return m
			},
			func(t *testing.T) *MockUserRepository {
				m := new(MockUserRepository)
				m.On("Upsert", bgContextType, mock.MatchedBy(func(u domain.User) bool {
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
			func(t *testing.T) *MockSessionRepository {
				m := new(MockSessionRepository)
				m.On("Save", bgContextType, "01JGCA8BBB00000000000000S1", "01JGCA8BBB00000000000000U1", mock.Anything).Return(nil)
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
			neverCalledMockUlid,
			neverCalledMockAuthService,
			neverCalledMockUserRepository,
			neverCalledMockSessionRepository,
			http.StatusBadRequest,
			map[string]string{},
		},
		{
			"should return 500 when user info can't be retrieved",
			"GET",
			"/v1/auth/google/callback?code=123",
			neverCalledMockUlid,
			func(t *testing.T) *MockAuthServer {
				m := new(MockAuthServer)
				m.On("GetUserInfo", mock.Anything, "123").Return(domain.User{}, assert.AnError)
				return m
			},
			neverCalledMockUserRepository,
			neverCalledMockSessionRepository,
			http.StatusInternalServerError,
			map[string]string{},
		},
		{
			"should return 500 when user info can't be saved",
			"GET",
			"/v1/auth/google/callback?code=123",
			func(t *testing.T) *MockULID {
				m := new(MockULID)
				m.On("New").Return("01JGCA8BBB00000000000000U1").Once()
				return m
			},
			func(t *testing.T) *MockAuthServer {
				m := new(MockAuthServer)
				m.On("GetUserInfo", bgContextType, "123").Return(testUser, nil)
				return m
			},
			func(t *testing.T) *MockUserRepository {
				m := new(MockUserRepository)
				m.On("Upsert", mock.Anything, mock.Anything).Return(assert.AnError)
				return m
			},
			neverCalledMockSessionRepository,
			http.StatusInternalServerError,
			map[string]string{},
		},
		{
			"should return 500 when session ID can't be generated",
			"GET",
			"/v1/auth/google/callback?code=123",
			func(t *testing.T) *MockULID {
				m := new(MockULID)
				m.On("New").Return("01JGCA8BBB00000000000000U1").Once()
				m.On("NewFromDate", mock.AnythingOfType(fmt.Sprintf("%T", time.Now()))).Return("", assert.AnError).Once()
				return m
			},
			func(t *testing.T) *MockAuthServer {
				m := new(MockAuthServer)
				m.On("GetUserInfo", bgContextType, "123").Return(testUser, nil)
				return m
			},
			func(t *testing.T) *MockUserRepository {
				m := new(MockUserRepository)
				m.On("Upsert", mock.Anything, mock.Anything).Return(nil)
				return m
			},
			neverCalledMockSessionRepository,
			http.StatusInternalServerError,
			map[string]string{},
		},
		{
			"should return 500 when session can't be saved",
			"GET",
			"/v1/auth/google/callback?code=123",
			func(t *testing.T) *MockULID {
				m := new(MockULID)
				m.On("New").Return("01JGCA8BBB00000000000000U1").Once()
				m.On("NewFromDate", mock.AnythingOfType(fmt.Sprintf("%T", time.Now()))).Return("01JGCA8BBB00000000000000S1", nil).Once()
				return m
			},
			func(t *testing.T) *MockAuthServer {
				m := new(MockAuthServer)
				m.On("GetUserInfo", bgContextType, "123").Return(testUser, nil)
				return m
			},
			func(t *testing.T) *MockUserRepository {
				m := new(MockUserRepository)
				m.On("Upsert", mock.Anything, mock.Anything).Return(nil)
				return m
			},
			func(t *testing.T) *MockSessionRepository {
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

			ulidMock := tc.ulidMock(t)
			userRepoMock := tc.userRepoMock(t)
			cryptMock := neverCalledMockCrypt(t)
			authServerMock := tc.authServerMock(t)
			sessionRepoMock := tc.sessionRepositoryMock(t)
			mailSecretRepoMock := neverCalledMockMailSecretRepository(t)

			RegisterRoutes(mux, config, ulidMock, authServerMock, userRepoMock, sessionRepoMock, cryptMock, mailSecretRepoMock)
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

func TestGoogleMailLogin(t *testing.T) {
	mux := http.NewServeMux()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/v1/auth/google-mail/login", nil)
	r.AddCookie(&http.Cookie{
		Name:     "session_token",
		Value:    "ABC-S3SS10N",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
	})

	ulidMock := neverCalledMockUlid(t)
	cryptMock := neverCalledMockCrypt(t)
	sessionRepoMock := new(MockSessionRepository)
	sessionRepoMock.On("GetByID", bgContextType, "ABC-S3SS10N").Return(testUser.ID, nil)
	userRepoMock := new(MockUserRepository)
	userRepoMock.On("GetByID", bgContextType, testUser.ID).Return(testUser, nil)
	mailSecretRepoMock := neverCalledMockMailSecretRepository(t)

	authServerMock := new(MockAuthServer)
	authServerMock.On("GetLoginUrl", []string{"https://www.googleapis.com/auth/gmail.readonly"}).Return("https://some-google.com/auth/login")

	RegisterRoutes(mux, config, ulidMock, authServerMock, userRepoMock, sessionRepoMock, cryptMock, mailSecretRepoMock)
	mux.ServeHTTP(w, r)

	response := w.Result()
	defer response.Body.Close()

	assert.Equal(t, http.StatusFound, response.StatusCode)
	assert.Equal(t, []string{"https://some-google.com/auth/login"}, response.Header["Location"])

	ulidMock.AssertExpectations(t)
	cryptMock.AssertExpectations(t)
	userRepoMock.AssertExpectations(t)
	authServerMock.AssertExpectations(t)
	sessionRepoMock.AssertExpectations(t)
	mailSecretRepoMock.AssertExpectations(t)
}

func TestGoogleMailLoginCallback(t *testing.T) {
	testCases := []struct {
		name               string
		verb               string
		endpoint           string
		requestHeaders     map[string]string
		sessionRepoMock    func(t *testing.T) *MockSessionRepository
		userRepoMock       func(t *testing.T) *MockUserRepository
		authServerMock     func(t *testing.T) *MockAuthServer
		cryptMock          func(t *testing.T) *MockCrypt
		mailSecretRepoMock func(t *testing.T) *MockMailSecretRepository
		expectedStatusCode int
		expectedHeaders    map[string]string
	}{
		{
			"should save access and refresh tokens as encrypted values in storage",
			"GET", "/v1/auth/google-mail/callback?code=some-auth-code",
			map[string]string{"Cookie": "session_token=01JGCA8BBB00000000000000S1; Path=/; HttpOnly; Secure"},
			func(t *testing.T) *MockSessionRepository {
				m := new(MockSessionRepository)
				m.On("GetByID", mock.Anything, "01JGCA8BBB00000000000000S1").Return(testUser.ID, nil).Once()
				return m
			},
			func(t *testing.T) *MockUserRepository {
				m := new(MockUserRepository)
				m.On("GetByID", mock.Anything, testUser.ID).Return(testUser, nil).Once()
				return m
			},
			func(t *testing.T) *MockAuthServer {
				m := new(MockAuthServer)
				m.On("GetTokens", mock.Anything, "some-auth-code").Return("access-token", "refresh-token", time.Date(2025, time.January, 18, 13, 30, 00, 0, time.Local), nil).Once()
				return m
			},
			func(t *testing.T) *MockCrypt {
				m := new(MockCrypt)
				m.On("EncryptString", "access-token").Return("access-encrypted").Once()
				m.On("EncryptString", "refresh-token").Return("refresh-encrypted").Once()
				return m
			},
			func(t *testing.T) *MockMailSecretRepository {
				m := new(MockMailSecretRepository)
				m.On("Save", mock.Anything, testUser.ID, "google", "access-encrypted", "refresh-encrypted", time.Date(2025, time.January, 18, 13, 30, 00, 0, time.Local)).Return(nil)
				return m
			},
			http.StatusFound,
			map[string]string{
				"Location": config.FrontendUrl + "/dashboard",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cryptMock := tc.cryptMock(t)
			ulidMock := neverCalledMockUlid(t)
			userRepoMock := tc.userRepoMock(t)
			authServerMock := tc.authServerMock(t)
			sessionRepoMock := tc.sessionRepoMock(t)
			mailSecretRepoMock := tc.mailSecretRepoMock(t)

			mux := http.NewServeMux()
			w := httptest.NewRecorder()
			r := httptest.NewRequest(tc.verb, tc.endpoint, nil)

			for k, v := range tc.requestHeaders {
				r.Header.Add(k, v)
			}

			RegisterRoutes(mux, config, ulidMock, authServerMock, userRepoMock, sessionRepoMock, cryptMock, mailSecretRepoMock)
			mux.ServeHTTP(w, r)

			response := w.Result()
			defer response.Body.Close()

			assert.Equal(t, tc.expectedStatusCode, response.StatusCode, "unexpected status code %d, expected %d", response.StatusCode, tc.expectedStatusCode)

			for headerName, headerValue := range tc.expectedHeaders {
				assert.Contains(t, response.Header, headerName)
				assert.Equal(t, headerValue, response.Header.Get(headerName))
			}

			ulidMock.AssertExpectations(t)
			cryptMock.AssertExpectations(t)
			userRepoMock.AssertExpectations(t)
			authServerMock.AssertExpectations(t)
			sessionRepoMock.AssertExpectations(t)
			mailSecretRepoMock.AssertExpectations(t)
		})
	}
}
