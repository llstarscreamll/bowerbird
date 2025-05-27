package infra

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"llstarscreamll/bowerbird/internal/auth/domain"
	"llstarscreamll/bowerbird/internal/auth/testdata"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var bgContextType = mock.AnythingOfType(fmt.Sprintf("%T", context.Background()))

func TestGoogleLogin(t *testing.T) {
	mux := http.NewServeMux()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/auth/google/login", nil)

	ulidMock := new(MockULID)
	ulidMock.On("NewFromDate", mock.Anything).Return("01JGCA8BBB00000000000000U1", nil)

	sessionRepoMock := new(MockSessionRepository)
	sessionRepoMock.On("Save", bgContextType, "googleOAuth-01JGCA8BBB00000000000000U1", "ABC-123", mock.Anything).Return(nil)

	authServerGatewayMock := new(MockAuthServerGateway)
	authServerGatewayMock.On("GetLoginUrl", "google", config.ApiUrl+"/api/v1/auth/google/callback", []string{}, "googleOAuth-01JGCA8BBB00000000000000U1").Return("https://some-google.com/auth/login", nil)

	cryptMock := neverCalledMockCrypt(t)
	userRepoMock := neverCalledMockUserRepository(t)
	mailGatewayMock := neverCalledMockMailGateway(t)
	categoryRepoMock := neverCalledMockCategoryRepository(t)
	walletRepoMock := neverCalledMockMockWalletRepository(t)
	mailMessageRepo := neverCalledMockMailMessageRepository(t)
	transactionRepoMock := neverCalledMockTransactionRepository(t)
	mailSecretRepoMock := neverCalledMockMailCredentialRepository(t)
	filePasswordRepoMock := neverCalledMockFilePasswordRepository(t)

	RegisterRoutes(mux, config, ulidMock, authServerGatewayMock, userRepoMock, sessionRepoMock, cryptMock, mailSecretRepoMock, mailGatewayMock, mailMessageRepo, walletRepoMock, transactionRepoMock, categoryRepoMock, filePasswordRepoMock)
	mux.ServeHTTP(w, r)

	response := w.Result()
	defer response.Body.Close()

	assert.Equal(t, http.StatusFound, response.StatusCode)
	assert.Equal(t, []string{"https://some-google.com/auth/login"}, response.Header["Location"])

	ulidMock.AssertExpectations(t)
	cryptMock.AssertExpectations(t)
	userRepoMock.AssertExpectations(t)
	walletRepoMock.AssertExpectations(t)
	mailGatewayMock.AssertExpectations(t)
	mailMessageRepo.AssertExpectations(t)
	sessionRepoMock.AssertExpectations(t)
	mailSecretRepoMock.AssertExpectations(t)
	transactionRepoMock.AssertExpectations(t)
	authServerGatewayMock.AssertExpectations(t)
	categoryRepoMock.AssertExpectations(t)
	filePasswordRepoMock.AssertExpectations(t)
}

func TestGoogleLoginCallback(t *testing.T) {
	testCases := []struct {
		testCase              string
		verb                  string
		endpoint              string
		ulidMock              func(t *testing.T) *MockULID
		authServerGatewayMock func(t *testing.T) *MockAuthServerGateway
		userRepoMock          func(t *testing.T) *MockUserRepository
		sessionRepositoryMock func(t *testing.T) *MockSessionRepository
		expectStatusCode      int
		expectedHeaders       map[string]string
	}{
		{
			"should return 302 when callback succeeds",
			"GET", "/api/v1/auth/google/callback?code=123",
			func(t *testing.T) *MockULID {
				m := new(MockULID)
				m.On("New").Return("01JGCA8BBB00000000000000U1").Once()
				m.On("NewFromDate", mock.AnythingOfType(fmt.Sprintf("%T", time.Now()))).Return("01JGCA8BBB00000000000000S1", nil).Once()
				return m
			},
			func(t *testing.T) *MockAuthServerGateway {
				m := new(MockAuthServerGateway)
				m.On("GetTokens", mock.Anything, "google", "123").Return(domain.Tokens{AccessToken: "access-token", RefreshToken: "refresh-token", ExpiresAt: time.Date(2025, time.January, 18, 13, 30, 00, 0, time.Local)}, nil).Once()
				m.On("GetUserProfile", bgContextType, "google", "access-token").Return(testUser, nil)
				return m
			},
			func(t *testing.T) *MockUserRepository {
				m := new(MockUserRepository)
				m.On("Upsert", bgContextType, mock.MatchedBy(func(u domain.User) bool {
					return assert.Equal(t, "01JGCA8BBB00000000000000U1", u.ID) &&
						assert.Equal(t, testdata.TestUser.Email, u.Email) &&
						assert.Equal(t, testdata.TestUser.Name, u.Name) &&
						assert.Equal(t, testdata.TestUser.GivenName, u.GivenName) &&
						assert.Equal(t, testdata.TestUser.FamilyName, u.FamilyName) &&
						assert.Equal(t, testdata.TestUser.PictureUrl, u.PictureUrl)
				})).Return("01JGCA8BBB00000000000000U1", nil)
				return m
			},
			func(t *testing.T) *MockSessionRepository {
				m := new(MockSessionRepository)
				m.On("Save", bgContextType, "01JGCA8BBB00000000000000S1", "01JGCA8BBB00000000000000U1", mock.Anything).Return(nil)
				return m
			},
			http.StatusFound,
			map[string]string{
				"Location":   config.WebUrl + "/dashboard",
				"Set-Cookie": "session_token=01JGCA8BBB00000000000000S1; Path=/; HttpOnly; Secure",
			},
		},
		{
			"should return 400 when auth code is empty",
			"GET",
			"/api/v1/auth/google/callback?code=",
			neverCalledMockUlid,
			neverCalledMockAuthServerGateway,
			neverCalledMockUserRepository,
			neverCalledMockSessionRepository,
			http.StatusBadRequest,
			map[string]string{},
		},
		{
			"should return 500 when user info can't be retrieved",
			"GET",
			"/api/v1/auth/google/callback?code=123",
			neverCalledMockUlid,
			func(t *testing.T) *MockAuthServerGateway {
				m := new(MockAuthServerGateway)
				m.On("GetTokens", mock.Anything, "google", "123").Return(domain.Tokens{AccessToken: "access-token", RefreshToken: "refresh-token", ExpiresAt: time.Date(2025, time.January, 18, 13, 30, 00, 0, time.Local)}, nil).Once()
				m.On("GetUserProfile", mock.Anything, "google", "access-token").Return(domain.User{}, assert.AnError)
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
			"/api/v1/auth/google/callback?code=123",
			func(t *testing.T) *MockULID {
				m := new(MockULID)
				m.On("New").Return("01JGCA8BBB00000000000000U1").Once()
				return m
			},
			func(t *testing.T) *MockAuthServerGateway {
				m := new(MockAuthServerGateway)
				m.On("GetTokens", mock.Anything, "google", "123").Return(domain.Tokens{AccessToken: "access-token", RefreshToken: "refresh-token", ExpiresAt: time.Date(2025, time.January, 18, 13, 30, 00, 0, time.Local)}, nil).Once()
				m.On("GetUserProfile", bgContextType, "google", "access-token").Return(testUser, nil)
				return m
			},
			func(t *testing.T) *MockUserRepository {
				m := new(MockUserRepository)
				m.On("Upsert", mock.Anything, mock.Anything).Return("", assert.AnError)
				return m
			},
			neverCalledMockSessionRepository,
			http.StatusInternalServerError,
			map[string]string{},
		},
		{
			"should return 500 when session ID can't be generated",
			"GET",
			"/api/v1/auth/google/callback?code=123",
			func(t *testing.T) *MockULID {
				m := new(MockULID)
				m.On("New").Return("01JGCA8BBB00000000000000U1").Once()
				m.On("NewFromDate", mock.AnythingOfType(fmt.Sprintf("%T", time.Now()))).Return("", assert.AnError).Once()
				return m
			},
			func(t *testing.T) *MockAuthServerGateway {
				m := new(MockAuthServerGateway)
				m.On("GetTokens", mock.Anything, "google", "123").Return(domain.Tokens{AccessToken: "access-token", RefreshToken: "refresh-token", ExpiresAt: time.Date(2025, time.January, 18, 13, 30, 00, 0, time.Local)}, nil).Once()
				m.On("GetUserProfile", bgContextType, "google", "access-token").Return(testUser, nil)
				return m
			},
			func(t *testing.T) *MockUserRepository {
				m := new(MockUserRepository)
				m.On("Upsert", mock.Anything, mock.Anything).Return("01JGCA8BBB00000000000000U1", nil)
				return m
			},
			neverCalledMockSessionRepository,
			http.StatusInternalServerError,
			map[string]string{},
		},
		{
			"should return 500 when session can't be saved",
			"GET",
			"/api/v1/auth/google/callback?code=123",
			func(t *testing.T) *MockULID {
				m := new(MockULID)
				m.On("New").Return("01JGCA8BBB00000000000000U1").Once()
				m.On("NewFromDate", mock.AnythingOfType(fmt.Sprintf("%T", time.Now()))).Return("01JGCA8BBB00000000000000S1", nil).Once()
				return m
			},
			func(t *testing.T) *MockAuthServerGateway {
				m := new(MockAuthServerGateway)
				m.On("GetTokens", mock.Anything, "google", "123").Return(domain.Tokens{AccessToken: "access-token", RefreshToken: "refresh-token", ExpiresAt: time.Date(2025, time.January, 18, 13, 30, 00, 0, time.Local)}, nil).Once()
				m.On("GetUserProfile", bgContextType, "google", "access-token").Return(testUser, nil)
				return m
			},
			func(t *testing.T) *MockUserRepository {
				m := new(MockUserRepository)
				m.On("Upsert", mock.Anything, mock.Anything).Return("01JGCA8BBB00000000000000U1", nil)
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
			sessionRepoMock := tc.sessionRepositoryMock(t)
			mailGatewayMock := neverCalledMockMailGateway(t)
			authServerGatewayMock := tc.authServerGatewayMock(t)
			walletRepoMock := neverCalledMockMockWalletRepository(t)
			mailMessageRepo := neverCalledMockMailMessageRepository(t)
			transactionRepoMock := neverCalledMockTransactionRepository(t)
			mailSecretRepoMock := neverCalledMockMailCredentialRepository(t)
			categoryRepoMock := neverCalledMockCategoryRepository(t)
			filePasswordRepoMock := neverCalledMockFilePasswordRepository(t)

			RegisterRoutes(mux, config, ulidMock, authServerGatewayMock, userRepoMock, sessionRepoMock, cryptMock, mailSecretRepoMock, mailGatewayMock, mailMessageRepo, walletRepoMock, transactionRepoMock, categoryRepoMock, filePasswordRepoMock)
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
			cryptMock.AssertExpectations(t)
			sessionRepoMock.AssertExpectations(t)
			mailGatewayMock.AssertExpectations(t)
			authServerGatewayMock.AssertExpectations(t)
			mailMessageRepo.AssertExpectations(t)
			mailSecretRepoMock.AssertExpectations(t)
			walletRepoMock.AssertExpectations(t)
			transactionRepoMock.AssertExpectations(t)
			categoryRepoMock.AssertExpectations(t)
			filePasswordRepoMock.AssertExpectations(t)
		})
	}
}

func TestGMailLogin(t *testing.T) {
	mux := http.NewServeMux()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/auth/google-mail/login", nil)
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
	sessionRepoMock.On("GetByID", bgContextType, "ABC-S3SS10N").Return(testdata.TestUser.ID, nil)
	userRepoMock := new(MockUserRepository)
	userRepoMock.On("GetByID", bgContextType, testdata.TestUser.ID).Return(testUser, nil)
	mailGatewayMock := neverCalledMockMailGateway(t)
	mailMessageRepo := neverCalledMockMailMessageRepository(t)
	mailSecretRepoMock := neverCalledMockMailCredentialRepository(t)
	walletRepoMock := neverCalledMockMockWalletRepository(t)
	transactionRepoMock := neverCalledMockTransactionRepository(t)
	categoryRepoMock := neverCalledMockCategoryRepository(t)
	filePasswordRepoMock := neverCalledMockFilePasswordRepository(t)

	authServerGatewayMock := new(MockAuthServerGateway)
	authServerGatewayMock.On("GetLoginUrl", "google", config.ApiUrl+"/api/v1/auth/google-mail/callback", []string{"https://www.googleapis.com/auth/gmail.readonly"}).Return("https://some-google.com/auth/login", nil)

	RegisterRoutes(mux, config, ulidMock, authServerGatewayMock, userRepoMock, sessionRepoMock, cryptMock, mailSecretRepoMock, mailGatewayMock, mailMessageRepo, walletRepoMock, transactionRepoMock, categoryRepoMock, filePasswordRepoMock)
	mux.ServeHTTP(w, r)

	response := w.Result()
	defer response.Body.Close()

	assert.Equal(t, http.StatusFound, response.StatusCode)
	assert.Equal(t, []string{"https://some-google.com/auth/login"}, response.Header["Location"])

	ulidMock.AssertExpectations(t)
	cryptMock.AssertExpectations(t)
	userRepoMock.AssertExpectations(t)
	walletRepoMock.AssertExpectations(t)
	mailGatewayMock.AssertExpectations(t)
	mailMessageRepo.AssertExpectations(t)
	sessionRepoMock.AssertExpectations(t)
	mailSecretRepoMock.AssertExpectations(t)
	transactionRepoMock.AssertExpectations(t)
	authServerGatewayMock.AssertExpectations(t)
	categoryRepoMock.AssertExpectations(t)
	filePasswordRepoMock.AssertExpectations(t)
}

func TestGMailLoginCallback(t *testing.T) {
	testCases := []struct {
		name                   string
		verb                   string
		endpoint               string
		requestHeaders         map[string]string
		sessionRepoMock        func(t *testing.T) *MockSessionRepository
		userRepoMock           func(t *testing.T) *MockUserRepository
		authServerGatewayMock  func(t *testing.T) *MockAuthServerGateway
		cryptMock              func(t *testing.T) *MockCrypt
		ulidMock               func(t *testing.T) *MockULID
		mailCredentialRepoMock func(t *testing.T) *MockMailCredentialRepository
		expectedStatusCode     int
		expectedHeaders        map[string]string
	}{
		{
			"should save access and refresh tokens as encrypted values in storage",
			"GET", "/api/v1/auth/google-mail/callback?code=some-auth-code",
			map[string]string{"Cookie": "session_token=01JGCA8BBB00000000000000S1; Path=/; HttpOnly; Secure"},
			func(t *testing.T) *MockSessionRepository {
				m := new(MockSessionRepository)
				m.On("GetByID", mock.Anything, "01JGCA8BBB00000000000000S1").Return(testdata.TestUser.ID, nil).Once()
				return m
			},
			func(t *testing.T) *MockUserRepository {
				m := new(MockUserRepository)
				m.On("GetByID", mock.Anything, testdata.TestUser.ID).Return(testUser, nil).Once()
				return m
			},
			func(t *testing.T) *MockAuthServerGateway {
				m := new(MockAuthServerGateway)
				m.On("GetTokens", mock.Anything, "google", "some-auth-code").Return(domain.Tokens{AccessToken: "access-token", RefreshToken: "refresh-token", ExpiresAt: time.Date(2025, time.January, 18, 13, 30, 00, 0, time.Local)}, nil).Once()
				m.On("GetUserProfile", mock.Anything, "google", "access-token").Return(testUser, nil).Once()
				return m
			},
			func(t *testing.T) *MockCrypt {
				m := new(MockCrypt)
				m.On("EncryptString", "access-token").Return("access-encrypted", nil).Once()
				m.On("EncryptString", "refresh-token").Return("refresh-encrypted", nil).Once()
				return m
			},
			func(t *testing.T) *MockULID {
				m := new(MockULID)
				m.On("New").Return("01JJ4DAEJQ0000000000000000").Once()
				return m
			},
			func(t *testing.T) *MockMailCredentialRepository {
				m := new(MockMailCredentialRepository)
				m.On("Save", mock.Anything, "01JJ4DAEJQ0000000000000000", testdata.TestUser.ID, "google", testdata.TestUser.Email, "access-encrypted", "refresh-encrypted", time.Date(2025, time.January, 18, 13, 30, 00, 0, time.Local)).Return(nil)
				return m
			},
			http.StatusFound,
			map[string]string{
				"Location": config.WebUrl + "/dashboard",
			},
		},
		{
			"should return 401 whe session ID does not exists",
			"GET", "/api/v1/auth/google-mail/callback?code=some-auth-code",
			map[string]string{"Cookie": "session_token=01JGCA8BBB00000000000000S1; Path=/; HttpOnly; Secure"},
			func(t *testing.T) *MockSessionRepository {
				m := new(MockSessionRepository)
				m.On("GetByID", mock.Anything, "01JGCA8BBB00000000000000S1").Return("", nil).Once()
				return m
			},
			neverCalledMockUserRepository,
			neverCalledMockAuthServerGateway,
			neverCalledMockCrypt,
			neverCalledMockUlid,
			neverCalledMockMailCredentialRepository,
			http.StatusUnauthorized,
			map[string]string{},
		},
		{
			"should return 500 whe user ID is not found",
			"GET", "/api/v1/auth/google-mail/callback?code=some-auth-code",
			map[string]string{"Cookie": "session_token=01JGCA8BBB00000000000000S1; Path=/; HttpOnly; Secure"},
			func(t *testing.T) *MockSessionRepository {
				m := new(MockSessionRepository)
				m.On("GetByID", mock.Anything, "01JGCA8BBB00000000000000S1").Return(testdata.TestUser.ID, nil).Once()
				return m
			},
			func(t *testing.T) *MockUserRepository {
				m := new(MockUserRepository)
				m.On("GetByID", mock.Anything, testdata.TestUser.ID).Return(domain.User{}, nil)
				return m
			},
			neverCalledMockAuthServerGateway,
			neverCalledMockCrypt,
			neverCalledMockUlid,
			neverCalledMockMailCredentialRepository,
			http.StatusInternalServerError,
			map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cryptMock := tc.cryptMock(t)
			ulidMock := tc.ulidMock(t)
			userRepoMock := tc.userRepoMock(t)
			sessionRepoMock := tc.sessionRepoMock(t)
			mailGatewayMock := neverCalledMockMailGateway(t)
			authServerGatewayMock := tc.authServerGatewayMock(t)
			mailCredentialRepoMock := tc.mailCredentialRepoMock(t)
			mailMessageRepo := neverCalledMockMailMessageRepository(t)
			walletRepoMock := neverCalledMockMockWalletRepository(t)
			transactionRepoMock := neverCalledMockTransactionRepository(t)
			categoryRepoMock := neverCalledMockCategoryRepository(t)
			filePasswordRepoMock := neverCalledMockFilePasswordRepository(t)
			mux := http.NewServeMux()
			w := httptest.NewRecorder()
			r := httptest.NewRequest(tc.verb, tc.endpoint, nil)

			for k, v := range tc.requestHeaders {
				r.Header.Add(k, v)
			}

			RegisterRoutes(mux, config, ulidMock, authServerGatewayMock, userRepoMock, sessionRepoMock, cryptMock, mailCredentialRepoMock, mailGatewayMock, mailMessageRepo, walletRepoMock, transactionRepoMock, categoryRepoMock, filePasswordRepoMock)
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
			walletRepoMock.AssertExpectations(t)
			mailGatewayMock.AssertExpectations(t)
			mailMessageRepo.AssertExpectations(t)
			sessionRepoMock.AssertExpectations(t)
			transactionRepoMock.AssertExpectations(t)
			authServerGatewayMock.AssertExpectations(t)
			mailCredentialRepoMock.AssertExpectations(t)
			categoryRepoMock.AssertExpectations(t)
		})
	}
}

func TestSyncTransactionsFromEmails(t *testing.T) {
	twoHourLater := time.Now().Add(time.Hour * 2)
	twoHourBefore := time.Now().Add(-time.Hour * 2)
	testCases := []struct {
		name                   string
		verb                   string
		endpoint               string
		requestHeaders         map[string]string
		sessionRepoMock        func(t *testing.T) *MockSessionRepository
		userRepoMock           func(t *testing.T) *MockUserRepository
		mailCredentialRepoMock func(t *testing.T) *MockMailCredentialRepository
		cryptMock              func(t *testing.T) *MockCrypt
		mailGatewayMock        func(t *testing.T) *MockMailGateway
		mailMessageMock        func(t *testing.T) *MockMailMessageRepository
		expectedStatusCode     int
		expectedHeaders        map[string]string
	}{
		{
			"should return 200 when sync transactions from emails succeeds",
			"POST", "/v1/transactions/sync-from-mail",
			map[string]string{"Cookie": "session_token=01JGCA8BBB00000000000000S1; Path=/; HttpOnly; Secure"},
			func(t *testing.T) *MockSessionRepository {
				m := new(MockSessionRepository)
				m.On("GetByID", mock.Anything, "01JGCA8BBB00000000000000S1").Return(testdata.TestUser.ID, nil).Once()
				return m
			},
			func(t *testing.T) *MockUserRepository {
				m := new(MockUserRepository)
				m.On("GetByID", mock.Anything, testdata.TestUser.ID).Return(testUser, nil).Once()
				return m
			},
			func(t *testing.T) *MockMailCredentialRepository {
				m := new(MockMailCredentialRepository)
				m.On("FindByUserID", mock.Anything, testdata.TestUser.ID).Return([]domain.MailCredential{
					{
						ID:           "01JJ4DAEJQ0000000000000000",
						UserID:       testdata.TestUser.ID,
						MailProvider: "google",
						MailAddress:  testdata.TestUser.Email,
						AccessToken:  "access-encrypted",
						RefreshToken: "refresh-encrypted",
						ExpiresAt:    twoHourLater,
						CreatedAt:    twoHourBefore,
					},
				}, nil).
					Once()
				return m
			},
			func(t *testing.T) *MockCrypt {
				m := new(MockCrypt)
				m.On("DecryptString", "access-encrypted").Return("access-token", nil).Once()
				m.On("DecryptString", "refresh-encrypted").Return("refresh-token", nil).Once()
				return m
			},
			func(t *testing.T) *MockMailGateway {
				m := new(MockMailGateway)
				m.On(
					"SearchFromDateAndSenders",
					mock.Anything,
					"google",
					domain.Tokens{
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
						ExpiresAt:    twoHourLater,
					},
					mock.MatchedBy(func(date time.Time) bool {
						return assert.Equal(t, date.Format("02/01/2006 15:04"), time.Now().Add(-time.Hour*24).Format("02/01/2006 15:04"))
					}), []string{"nu@nu.com.co", "colpatriaInforma@scotiabankcolpatria.com", "bancodavivienda@davivienda.com"},
				).
					Return([]domain.MailMessage{
						{
							ID:         "01JJTGEG2X0000000000000000",
							ExternalID: "194b4c06e8bc41ec",
							From:       "nu@nu.com.co",
							To:         "john@doe.com",
							Subject:    "El dinero que enviaste ya está del otro lado",
							Body:       "Example message body",
						},
					}, nil).Once()
				return m
			},
			func(t *testing.T) *MockMailMessageRepository {
				m := new(MockMailMessageRepository)
				m.On("UpsertMany", mock.Anything, []domain.MailMessage{
					{
						ID:         "01JJTGEG2X0000000000000000",
						ExternalID: "194b4c06e8bc41ec",
						UserID:     testdata.TestUser.ID,
						From:       "nu@nu.com.co",
						To:         "john@doe.com",
						Subject:    "El dinero que enviaste ya está del otro lado",
						Body:       "Example message body",
					},
				}).Return(nil).Once()
				return m
			},
			http.StatusOK,
			map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cryptMock := tc.cryptMock(t)
			userRepoMock := tc.userRepoMock(t)
			ulidMock := neverCalledMockUlid(t)
			sessionRepoMock := tc.sessionRepoMock(t)
			mailGatewayMock := tc.mailGatewayMock(t)
			mailMessageRepoMock := tc.mailMessageMock(t)
			authGatewayMock := neverCalledMockAuthServerGateway(t)
			mailCredentialRepoMock := tc.mailCredentialRepoMock(t)
			walletRepoMock := neverCalledMockMockWalletRepository(t)
			transactionRepoMock := neverCalledMockTransactionRepository(t)
			categoryRepoMock := neverCalledMockCategoryRepository(t)
			filePasswordRepoMock := neverCalledMockFilePasswordRepository(t)

			mux := http.NewServeMux()
			w := httptest.NewRecorder()
			r := httptest.NewRequest(tc.verb, tc.endpoint, nil)

			for k, v := range tc.requestHeaders {
				r.Header.Add(k, v)
			}

			RegisterRoutes(mux, config, ulidMock, authGatewayMock, userRepoMock, sessionRepoMock, cryptMock, mailCredentialRepoMock, mailGatewayMock, mailMessageRepoMock, walletRepoMock, transactionRepoMock, categoryRepoMock, filePasswordRepoMock)
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
			walletRepoMock.AssertExpectations(t)
			sessionRepoMock.AssertExpectations(t)
			mailGatewayMock.AssertExpectations(t)
			authGatewayMock.AssertExpectations(t)
			mailMessageRepoMock.AssertExpectations(t)
			transactionRepoMock.AssertExpectations(t)
			mailCredentialRepoMock.AssertExpectations(t)
			categoryRepoMock.AssertExpectations(t)
			filePasswordRepoMock.AssertExpectations(t)
		})
	}
}
