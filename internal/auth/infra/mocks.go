package infra

import (
	"context"
	"testing"
	"time"

	"llstarscreamll/bowerbird/internal/auth/domain"
	commonDomain "llstarscreamll/bowerbird/internal/common/domain"

	"github.com/stretchr/testify/mock"
)

type MockAuthServerGateway struct {
	mock.Mock
}

func (m *MockAuthServerGateway) GetLoginUrl(provider, redirectUrl string, scopes []string, state string) (string, error) {
	args := m.Called(provider, redirectUrl, scopes, state)
	return args.String(0), args.Error(1)
}

func (m *MockAuthServerGateway) GetTokens(ctx context.Context, provider string, authCode string, state string) (domain.Tokens, error) {
	args := m.Called(ctx, provider, authCode, state)
	return args.Get(0).(domain.Tokens), args.Error(1)
}

func (m *MockAuthServerGateway) GetUserProfile(ctx context.Context, provider string, authCode string) (domain.User, error) {
	args := m.Called(ctx, provider, authCode)
	return args.Get(0).(domain.User), args.Error(1)
}

type MockMailGateway struct {
	mock.Mock
}

func (m *MockMailGateway) SearchFromDateAndSenders(ctx context.Context, provider string, tokens domain.Tokens, startDate time.Time, senders []string) ([]domain.MailMessage, error) {
	args := m.Called(ctx, provider, tokens, startDate, senders)
	return args.Get(0).([]domain.MailMessage), args.Error(1)
}

type MockMailMessageRepository struct {
	mock.Mock
}

func (m *MockMailMessageRepository) UpsertMany(ctx context.Context, messages []domain.MailMessage) error {
	args := m.Called(ctx, messages)
	return args.Error(0)
}

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Upsert(ctx context.Context, user domain.User) (string, error) {
	args := m.Called(ctx, user)
	return args.String(0), args.Error(1)
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

func (m *MockSessionRepository) Delete(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockSessionRepository) GetByID(ctx context.Context, ID string) (domain.Session, error) {
	args := m.Called(ctx, ID)
	return args.Get(0).(domain.Session), args.Error(1)
}

type MockCrypt struct {
	mock.Mock
}

func (m *MockCrypt) EncryptString(str string) (string, error) {
	args := m.Called(str)
	return args.String(0), args.Error(1)
}

func (m *MockCrypt) DecryptString(str string) (string, error) {
	args := m.Called(str)
	return args.String(0), args.Error(1)
}

type MockMailCredentialRepository struct {
	mock.Mock
}

func (m *MockMailCredentialRepository) Save(ctx context.Context, ID, userID, walletID, mailProvider, mailAddress, accessToken, refreshToken string, expiresAt time.Time) error {
	args := m.Called(ctx, ID, userID, walletID, mailProvider, mailAddress, accessToken, refreshToken, expiresAt)
	return args.Error(0)
}

func (m *MockMailCredentialRepository) FindByWalletID(ctx context.Context, walletID string) ([]domain.MailCredential, error) {
	args := m.Called(ctx, walletID)

	return args.Get(0).([]domain.MailCredential), args.Error(1)
}

type MockWalletRepository struct {
	mock.Mock
}

func (m *MockWalletRepository) Create(ctx context.Context, wallet domain.UserWallet) error {
	args := m.Called(ctx, wallet)

	return args.Error(0)
}

func (m *MockWalletRepository) FindByUserID(ctx context.Context, userID string) ([]domain.UserWallet, error) {
	args := m.Called(ctx, userID)

	return args.Get(0).([]domain.UserWallet), args.Error(1)
}

type MockTransactionRepository struct {
	mock.Mock
}

func (m *MockTransactionRepository) GetByWalletIDAndID(ctx context.Context, walletID string, transactionID string) (domain.Transaction, error) {
	panic("unimplemented")
}

func (m *MockTransactionRepository) Update(ctx context.Context, transaction domain.Transaction) error {
	panic("unimplemented")
}

func (m *MockTransactionRepository) UpsertMany(ctx context.Context, transactions []domain.Transaction) error {
	args := m.Called(ctx, transactions)
	return args.Error(0)
}

func (m *MockTransactionRepository) FindByWalletID(ctx context.Context, walletID string) ([]domain.Transaction, error) {
	args := m.Called(ctx, walletID)
	return args.Get(0).([]domain.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) GetMetrics(ctx context.Context, walletID string, from, to time.Time) (domain.Metrics, error) {
	args := m.Called(ctx, walletID, from, to)
	return args.Get(0).(domain.Metrics), args.Error(1)
}

type MockCategoryRepository struct {
	mock.Mock
}

func (m *MockCategoryRepository) Create(ctx context.Context, category domain.Category) error {
	args := m.Called(ctx, category)
	return args.Error(0)
}

func (m *MockCategoryRepository) FindByWalletID(ctx context.Context, walletID string) ([]domain.Category, error) {
	args := m.Called(ctx, walletID)
	return args.Get(0).([]domain.Category), args.Error(1)
}

type MockFilePasswordRepository struct {
	mock.Mock
}

func (m *MockFilePasswordRepository) GetByUserID(ctx context.Context, userID string) ([]string, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockFilePasswordRepository) Upsert(ctx context.Context, userID string, passwords []string) error {
	args := m.Called(ctx, userID, passwords)
	return args.Error(0)
}

func neverCalledMockFilePasswordRepository(t *testing.T) *MockFilePasswordRepository {
	m := new(MockFilePasswordRepository)
	m.AssertNotCalled(t, "GetByUserID")
	return m
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

func neverCalledMockMailCredentialRepository(t *testing.T) *MockMailCredentialRepository {
	m := new(MockMailCredentialRepository)
	m.AssertNotCalled(t, "Save")
	return m
}

func neverCalledMockAuthServerGateway(t *testing.T) *MockAuthServerGateway {
	m := new(MockAuthServerGateway)
	m.AssertNotCalled(t, "Save")
	return m
}

func neverCalledMockMailGateway(t *testing.T) *MockMailGateway {
	m := new(MockMailGateway)
	m.AssertNotCalled(t, "SearchByDateRangeAndSenders")
	return m
}

func neverCalledMockMailMessageRepository(t *testing.T) *MockMailMessageRepository {
	m := new(MockMailMessageRepository)
	m.AssertNotCalled(t, "UpsertMany")
	return m
}

func neverCalledMockTransactionRepository(t *testing.T) *MockTransactionRepository {
	m := new(MockTransactionRepository)
	m.AssertNotCalled(t, "New")
	return m
}

func neverCalledMockCategoryRepository(t *testing.T) *MockCategoryRepository {
	m := new(MockCategoryRepository)
	m.AssertNotCalled(t, "New")
	return m
}

func neverCalledMockMockWalletRepository(t *testing.T) *MockWalletRepository {
	m := new(MockWalletRepository)
	m.AssertNotCalled(t, "New")
	return m
}

var config = commonDomain.AppConfig{
	ApiUrl:     "http://localhost:8080",
	ServerPort: ":8080",
	WebUrl:     "http://localhost:4200",
}

var testUser = domain.User{
	ID:         "01JGCZXZEC00000000000000U1",
	Email:      "john@doe.com",
	Name:       "John Doe",
	GivenName:  "John",
	FamilyName: "Doe",
	PictureUrl: "https://some-google.com/picture.jpg",
}

var nuBankTransactionMailExample = `
<!doctype html><html xmlns="http://www.w3.org/1999/xhtml" xmlns:v="urn:schemas-microsoft-com:vml" xmlns:o="urn:schemas-microsoft-com:office:office"><head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8"><title></title><!--[if !mso]><!-- --><meta http-equiv="X-UA-Compatible" content="IE=edge"><!--<![endif]--><meta name="viewport" content="width=device-width,initial-scale=1"><style type="text/css">#outlook a { padding:0; }
          body { margin:0;padding:0;-webkit-text-size-adjust:100%;-ms-text-size-adjust:100%; }
          table, td { border-collapse:collapse;mso-table-lspace:0pt;mso-table-rspace:0pt; }
          img { border:0;height:auto;line-height:100%; outline:none;text-decoration:none;-ms-interpolation-mode:bicubic; }
          p { display:block;margin:13px 0; }</style><!--[if mso]>
        <xml>
        <o:OfficeDocumentSettings>
          <o:AllowPNG/>
          <o:PixelsPerInch>96</o:PixelsPerInch>
        </o:OfficeDocumentSettings>
        </xml>
        <![endif]--><!--[if lte mso 11]>
        <style type="text/css">
          .mj-outlook-group-fix { width:100% !important; }
        </style>
        <![endif]--><!--[if !mso]><!--><link href="https://fonts.googleapis.com/css?family=Ubuntu:300,400,500,700" rel="stylesheet" type="text/css"><style type="text/css">@import url(https://fonts.googleapis.com/css?family=Ubuntu:300,400,500,700);</style><!--<![endif]--><style type="text/css">@media only screen and (min-width:480px) {
        .mj-column-per-100 { width:100% !important; max-width: 100%; }
      }</style><style type="text/css">@media only screen and (max-width:480px) {
      table.mj-full-width-mobile { width: 100% !important; }
      td.mj-full-width-mobile { width: auto !important; }
    }</style><meta name="format-detection" content="telephone=no"></head><body style="background-color:#f7f7f7;"><div style="display:none;font-size:1px;color:#ffffff;line-height:1px;max-height:0px;max-width:0px;opacity:0;overflow:hidden;">Confirmamos tu envío de dinero</div><div style="background-color:#f7f7f7;"><!--[if mso | IE]><table align="center" border="0" cellpadding="0" cellspacing="0" class="" style="width:600px;" width="600" ><tr><td style="line-height:0px;font-size:0px;mso-line-height-rule:exactly;"><![endif]--><div style="margin:0px auto;max-width:600px;"><table align="center" border="0" cellpadding="0" cellspacing="0" role="presentation" style="width:100%;"><tbody><tr><td style="direction:ltr;font-size:0px;padding:10px;text-align:center;"><!--[if mso | IE]><table role="presentation" border="0" cellpadding="0" cellspacing="0"><tr></tr></table><![endif]--></td></tr></tbody></table></div><!--[if mso | IE]></td></tr></table><table align="center" border="0" cellpadding="0" cellspacing="0" class="dropShadow-1-outlook mainContainer-outlook" style="width:600px;" width="600" ><tr><td style="line-height:0px;font-size:0px;mso-line-height-rule:exactly;"><![endif]--><div class="dropShadow-1 mainContainer" style="border: 1px solid #D9D9D9; box-shadow: 0 0 30px 0 rgba(0, 0, 0, 0.10); background: white; background-color: white; margin: 0px auto; max-width: 600px;"><table align="center" border="0" cellpadding="0" cellspacing="0" role="presentation" style="background:white;background-color:white;width:100%;"><tbody><tr><td style="direction:ltr;font-size:0px;padding:0;padding-bottom:30px;text-align:center;"><!--[if mso | IE]><table role="presentation" border="0" cellpadding="0" cellspacing="0"><tr><td class="" width="600px" ><table align="center" border="0" cellpadding="0" cellspacing="0" role="presentation" style="width:600px;" width="600" ><tr><td style="line-height:0;font-size:0;mso-line-height-rule:exactly;"><v:image style="border:0;mso-position-horizontal:center;position:absolute;top:0;width:600px;z-index:-3;" xmlns:v="urn:schemas-microsoft-com:vml" /><![endif]--><div style="margin:0 auto;max-width:600px;"><table border="0" cellpadding="0" cellspacing="0" role="presentation" style="width:100%;"><tr style="vertical-align:top;"><td style="width:0.01%;padding-bottom:NaN%;mso-padding-bottom-alt:0;"></td><td style="background:#A376FF;background-position:center center;background-repeat:no-repeat;padding:0px;vertical-align:top;"><!--[if mso | IE]><table border="0" cellpadding="0" cellspacing="0" style="width:600px;" width="600" ><tr><td style=""><![endif]--><div class="mj-hero-content" style="margin:0px auto;"><table border="0" cellpadding="0" cellspacing="0" role="presentation" style="width:100%;margin:0px;"><tr><td><table border="0" cellpadding="0" cellspacing="0" role="presentation" style="width:100%;margin:0px;"><tr><td align="left" style="font-size:0px;padding:40px 24px 30px 24px;word-break:break-word;"><table cellpadding="0" cellspacing="0" width="100%" border="0" style="color:#555555;font-family:Ubuntu, Helvetica, Arial, sans-serif;font-size:16px;line-height:150%;table-layout:auto;width:100%;border:none;"><tr><td><img src="https://cdn.nubank.com.br/colombia/cuenta/transfer-hero-image.png" width="312px"></td></tr></table></td></tr><tr><td align="left" style="font-size:0px;padding:48px 20px 144px 20px;word-break:break-word;"><div style="font-family:Helvetica Neue, sans-serif;font-size:40px;font-weight:bold;line-height:100%;text-align:left;color:#ffffff;">Confirmamos tu envío de dinero</div></td></tr></table></td></tr></table></div><!--[if mso | IE]></td></tr></table><![endif]--></td><td style="width:0.01%;padding-bottom:NaN%;mso-padding-bottom-alt:0;"></td></tr></table></div><!--[if mso | IE]></td></tr></table></td></tr><tr><td class="" width="600px" ><table align="center" border="0" cellpadding="0" cellspacing="0" class="" style="width:600px;" width="600" ><tr><td style="line-height:0px;font-size:0px;mso-line-height-rule:exactly;"><![endif]--><div style="margin:0px auto;max-width:600px;"><table align="center" border="0" cellpadding="0" cellspacing="0" role="presentation" style="width:100%;"><tbody><tr><td style="direction:ltr;font-size:0px;padding:0 20px;text-align:center;"><!--[if mso | IE]><table role="presentation" border="0" cellpadding="0" cellspacing="0"><tr><td class="" style="vertical-align:top;width:560px;" ><![endif]--><div class="mj-column-per-100 mj-outlook-group-fix" style="font-size:0px;text-align:left;direction:ltr;display:inline-block;vertical-align:top;width:100%;"><table border="0" cellpadding="0" cellspacing="0" role="presentation" style="vertical-align:top;" width="100%"><tr><td align="left" style="font-size:0px;padding:24px 0 0 0;word-break:break-word;"><div style="font-family:Helvetica;font-size:18px;line-height:150%;text-align:left;color:#111111;"><h2>Hola, Johan:</h2></div></td></tr></table></div><!--[if mso | IE]></td></tr></table><![endif]--></td></tr></tbody></table></div><!--[if mso | IE]></td></tr></table></td></tr><tr><td class="" width="600px" ><table align="center" border="0" cellpadding="0" cellspacing="0" class="" style="width:600px;" width="600" ><tr><td style="line-height:0px;font-size:0px;mso-line-height-rule:exactly;"><![endif]--><div style="margin:0px auto;max-width:600px;"><table align="center" border="0" cellpadding="0" cellspacing="0" role="presentation" style="width:100%;"><tbody><tr><td style="direction:ltr;font-size:0px;padding:0 20px;text-align:center;"><!--[if mso | IE]><table role="presentation" border="0" cellpadding="0" cellspacing="0"><tr><td class="" style="vertical-align:top;width:560px;" ><![endif]--><div class="mj-column-per-100 mj-outlook-group-fix" style="font-size:0px;text-align:left;direction:ltr;display:inline-block;vertical-align:top;width:100%;"><table border="0" cellpadding="0" cellspacing="0" role="presentation" style="vertical-align:top;" width="100%"><tr><td align="left" style="font-size:0px;padding:0;word-break:break-word;"><table cellpadding="0" cellspacing="0" width="100%" border="0" style="color:#555555;font-family:Ubuntu, Helvetica, Arial, sans-serif;font-size:16px;line-height:150%;table-layout:auto;width:100%;border:none;"><tr><td><img src="https://cdn.nubank.com.br/colombia/cuenta/transfer-action.png" width="48px"></td></tr></table></td></tr><tr><td align="left" style="font-size:0px;padding:24px 0 0 0;word-break:break-word;"><div style="font-family:Helvetica;font-size:16px;font-weight:400;line-height:150%;text-align:left;color:#111111;">En Nu queremos que tengas en todo momento el control de tu dinero y que sepas cuando hubo un movimiento en tu cuenta.</div></td></tr><tr><td align="left" style="font-size:0px;padding:24px 0px 24px 0px;word-break:break-word;"><div style="font-family:Helvetica;font-size:16px;font-weight:500;line-height:150%;text-align:left;color:#8d3dc8;">Aquí puedes ver los detalles del envío</div></td></tr></table></div><!--[if mso | IE]></td></tr></table><![endif]--></td></tr></tbody></table></div><!--[if mso | IE]></td></tr></table></td></tr><tr><td class="" width="600px" ><table align="center" border="0" cellpadding="0" cellspacing="0" class="" style="width:600px;" width="600" ><tr><td style="line-height:0px;font-size:0px;mso-line-height-rule:exactly;"><![endif]--><div style="margin:0px auto;max-width:600px;"><table align="center" border="0" cellpadding="0" cellspacing="0" role="presentation" style="width:100%;"><tbody><tr><td style="direction:ltr;font-size:0px;padding:0 20px;padding-top:0px;text-align:center;"><!--[if mso | IE]><table role="presentation" border="0" cellpadding="0" cellspacing="0"><tr><td class="" style="vertical-align:top;width:560px;" ><![endif]--><div class="mj-column-per-100 mj-outlook-group-fix" style="font-size:0px;text-align:left;direction:ltr;display:inline-block;vertical-align:top;width:100%;"><table border="0" cellpadding="0" cellspacing="0" role="presentation" width="100%"><tbody><tr><td style="background-color:#f5f5f5;vertical-align:top;padding:20px;"><table border="0" cellpadding="0" cellspacing="0" role="presentation" width="100%"><tr><td align="left" style="font-size:0px;padding:2px 24px 2px 24px;word-break:break-word;"><div style="font-family:Helvetica;font-size:16px;font-weight:500;line-height:150%;text-align:left;color:#111111;">Recibe</div></td></tr><tr><td align="left" style="font-size:0px;padding:2px 24px 24px 24px;word-break:break-word;"><div style="font-family:Helvetica;font-size:14px;font-weight:400;line-height:150%;text-align:left;color:#707070;">Diana E. Nu placa: DEA983</div></td></tr><tr><td align="left" style="font-size:0px;padding:2px 24px 2px 24px;word-break:break-word;"><div style="font-family:Helvetica;font-size:16px;font-weight:500;line-height:150%;text-align:left;color:#111111;">Monto</div></td></tr><tr><td align="left" style="font-size:0px;padding:2px 24px 24px 24px;word-break:break-word;"><div style="font-family:Helvetica;font-size:14px;font-weight:400;line-height:150%;text-align:left;color:#707070;">$300.000,00</div></td></tr><tr><td align="left" style="font-size:0px;padding:2px 24px 2px 24px;word-break:break-word;"><div style="font-family:Helvetica;font-size:16px;font-weight:500;line-height:150%;text-align:left;color:#111111;">Impuesto del 4x1.000</div></td></tr><tr><td align="left" style="font-size:0px;padding:2px 24px 24px 24px;word-break:break-word;"><div style="font-family:Helvetica;font-size:14px;font-weight:400;line-height:150%;text-align:left;color:#707070;">$1.200,00</div></td></tr></table></td></tr></tbody></table></div><!--[if mso | IE]></td></tr></table><![endif]--></td></tr></tbody></table></div><!--[if mso | IE]></td></tr></table></td></tr><tr><td class="" width="600px" ><table align="center" border="0" cellpadding="0" cellspacing="0" class="" style="width:600px;" width="600" ><tr><td style="line-height:0px;font-size:0px;mso-line-height-rule:exactly;"><![endif]--><div style="margin:0px auto;max-width:600px;"><table align="center" border="0" cellpadding="0" cellspacing="0" role="presentation" style="width:100%;"><tbody><tr><td style="direction:ltr;font-size:0px;padding:0 20px;text-align:center;"><!--[if mso | IE]><table role="presentation" border="0" cellpadding="0" cellspacing="0"><tr><td class="" style="vertical-align:top;width:560px;" ><![endif]--><div class="mj-column-per-100 mj-outlook-group-fix" style="font-size:0px;text-align:left;direction:ltr;display:inline-block;vertical-align:top;width:100%;"><table border="0" cellpadding="0" cellspacing="0" role="presentation" style="vertical-align:top;" width="100%"><tr><td align="left" style="font-size:0px;padding:24px 0 0 0;word-break:break-word;"><div style="font-family:Helvetica;font-size:16px;font-weight:400;line-height:150%;text-align:left;color:#111111;">Si tienes preguntas, siempre nos puedes escribir a ayuda@nu.com.co</div></td></tr></table></div><!--[if mso | IE]></td></tr></table><![endif]--></td></tr></tbody></table></div><!--[if mso | IE]></td></tr></table></td></tr><tr><td class="" width="600px" ><table align="center" border="0" cellpadding="0" cellspacing="0" class="" style="width:600px;" width="600" ><tr><td style="line-height:0px;font-size:0px;mso-line-height-rule:exactly;"><![endif]--><div style="margin:0px auto;max-width:600px;"><table align="center" border="0" cellpadding="0" cellspacing="0" role="presentation" style="width:100%;"><tbody><tr><td style="direction:ltr;font-size:0px;padding:0 20px;text-align:center;"><!--[if mso | IE]><table role="presentation" border="0" cellpadding="0" cellspacing="0"><tr><td class="" style="vertical-align:top;width:560px;" ><![endif]--><div class="mj-column-per-100 mj-outlook-group-fix" style="font-size:0px;text-align:left;direction:ltr;display:inline-block;vertical-align:top;width:100%;"><table border="0" cellpadding="0" cellspacing="0" role="presentation" style="vertical-align:top;" width="100%"><tr><td align="left" style="font-size:0px;padding:48px 0 12px 0;word-break:break-word;"><div style="font-family:Helvetica;font-size:18px;line-height:150%;text-align:left;color:#111111;">Un abrazo,<br><strong>Equipo Nu</strong></div></td></tr></table></div><!--[if mso | IE]></td></tr></table><![endif]--></td></tr></tbody></table></div><!--[if mso | IE]></td></tr></table></td></tr></table><![endif]--></td></tr></tbody></table></div><!--[if mso | IE]></td></tr></table><table align="center" border="0" cellpadding="0" cellspacing="0" class="" style="width:600px;" width="600" ><tr><td style="line-height:0px;font-size:0px;mso-line-height-rule:exactly;"><![endif]--><div style="background:#191919;background-color:#191919;margin:0px auto;max-width:600px;"><table align="center" border="0" cellpadding="0" cellspacing="0" role="presentation" style="background:#191919;background-color:#191919;width:100%;"><tbody><tr><td style="direction:ltr;font-size:0px;padding:0 20px;text-align:center;"><!--[if mso | IE]><table role="presentation" border="0" cellpadding="0" cellspacing="0"><tr><td class="" style="vertical-align:top;width:560px;" ><![endif]--><div class="mj-column-per-100 mj-outlook-group-fix" style="font-size:0px;text-align:left;direction:ltr;display:inline-block;vertical-align:top;width:100%;"><table border="0" cellpadding="0" cellspacing="0" role="presentation" style="vertical-align:top;" width="100%"><tr><td style="font-size:0px;word-break:break-word;"><!--[if mso | IE]><table role="presentation" border="0" cellpadding="0" cellspacing="0"><tr><td height="50" style="vertical-align:top;height:50px;"><![endif]--><div style="height:50px;">&nbsp;</div><!--[if mso | IE]></td></tr></table><![endif]--></td></tr><tr><td align="left" style="font-size:0px;padding:10px 25px;word-break:break-word;"><!--[if mso | IE]><table align="left" border="0" cellpadding="0" cellspacing="0" role="presentation" ><tr><td><![endif]--><table align="left" border="0" cellpadding="0" cellspacing="0" role="presentation" style="float:none;display:inline-table;"><tr><td style="padding:0 20px 0 0px;"><table border="0" cellpadding="0" cellspacing="0" role="presentation" style="border-radius:3px;width:40px;"><tr><td style="font-size:0;height:40px;vertical-align:middle;width:40px;"><a href="https://url3462.nu.com.co/ls/click?upn=u001.hYbpdgGXNmHJnNDZW5hKZNaA6aQ-2FaWw6cWYIg5qwQcOOv89MecQ56BJtLrRzkupEwfeI_uPPJ2PGmHsJ220WnCZ4gZt4sdebFX9D0XEz37quGCBH5PiqdHvAvFK2LMvUPjcgdDSwkGTTuzRbmQD61otD0A3a5sjU5xL6MrSGD9tRyVfxFB-2BiOu-2BbtcNKsqvUeI4ki5M3xcBY70fAvGMN-2FQmhHXLkNk8NdNMBTW6lKIxgc4X70o0KSOLVxv-2FtwBtIe9ahJj-2BZhMhKoGcMfjO5graMbFHZ-2Feovp5NozwFnTWG4KqxkXPzWV-2FkqkvzJObOOWFSncjxSqMANAu924JZOx0XkqOZlj1EuOOaddRkWz504QJV1n5X8N4j3644SB7vdelSjZckHcglBbEFxLOrz6u567rhaS3DhbQAvTgPoKtm4wpElyvF2Gj0BEm-2BtetVPzO8ARab8-2Bgzql8ytwu0Zv0XOBt2UCh-2BB3tcL8gziM5dkwoMVhSr3WvRrPsj0ZTVxXHmKVBn5boCsb8xNimolfq3qJQxeZWiYqIiywJVJrK0uaoRI-3D" target="_blank"><img height="40" src="https://cdn.nubank.com.br/colombia/icono-facebook.png" style="border-radius:3px;display:block;" width="40"></a></td></tr></table></td></tr></table><!--[if mso | IE]></td><td><![endif]--><table align="left" border="0" cellpadding="0" cellspacing="0" role="presentation" style="float:none;display:inline-table;"><tr><td style="padding:0 20px 0 0px;"><table border="0" cellpadding="0" cellspacing="0" role="presentation" style="border-radius:3px;width:40px;"><tr><td style="font-size:0;height:40px;vertical-align:middle;width:40px;"><a href="https://url3462.nu.com.co/ls/click?upn=u001.hYbpdgGXNmHJnNDZW5hKZDzOVJmn0Y0qjVPU1q2KW0dWihfS9lFMPIiINA3GrzxrEv9y_uPPJ2PGmHsJ220WnCZ4gZt4sdebFX9D0XEz37quGCBH5PiqdHvAvFK2LMvUPjcgdDSwkGTTuzRbmQD61otD0A3a5sjU5xL6MrSGD9tRyVfxFB-2BiOu-2BbtcNKsqvUeI4ki5M3xcBY70fAvGMN-2FQmhHXLkNk8NdNMBTW6lKIxgc4X70o0KSOLVxv-2FtwBtIe9ahJj-2BZhMhKoGcMfjO5graMbFHZ-2Feovp5NozwFnTWG4KqxkXPzWV-2FkqkvzJObOOWFSncjxSqMANAu924JZOx0XkqOZlj1EuOOaddRkWz504QJV1n5X8N4j3644SB7vdelSjZckHcglBbEFxLOrz6u567rhWqRKsGEAsFPun5Gcigyrxx2AejhAONClcixE3wdnGkctXMEuJJWJzDNMPXRcL4xYg5BdwfTmTdwwc-2Bw-2BEsc5ECs3TUeDt16Bss0v4HiDC6mu9xC9EuW92cId-2BeR7QFI3LkbB0-2Bn4xGrKia7IY5AcE-3D" target="_blank"><img height="40" src="https://cdn.nubank.com.br/colombia/icono-instagram.png" style="border-radius:3px;display:block;" width="40"></a></td></tr></table></td></tr></table><!--[if mso | IE]></td><td><![endif]--><table align="left" border="0" cellpadding="0" cellspacing="0" role="presentation" style="float:none;display:inline-table;"><tr><td style="padding:0 20px 0 0px;"><table border="0" cellpadding="0" cellspacing="0" role="presentation" style="border-radius:3px;width:40px;"><tr><td style="font-size:0;height:40px;vertical-align:middle;width:40px;"><a href="https://url3462.nu.com.co/ls/click?upn=u001.hYbpdgGXNmHJnNDZW5hKZP5ny2Ga1AxrbR8eKFU7hWVP-2B6i5COTGtB-2FmUzpS2UB9SBtPfYqKBy1rKeDGmcRgZNakL9RXKE1VZd4-2FQCSMc18-3DstLC_uPPJ2PGmHsJ220WnCZ4gZt4sdebFX9D0XEz37quGCBH5PiqdHvAvFK2LMvUPjcgdDSwkGTTuzRbmQD61otD0A3a5sjU5xL6MrSGD9tRyVfxFB-2BiOu-2BbtcNKsqvUeI4ki5M3xcBY70fAvGMN-2FQmhHXLkNk8NdNMBTW6lKIxgc4X70o0KSOLVxv-2FtwBtIe9ahJj-2BZhMhKoGcMfjO5graMbFHZ-2Feovp5NozwFnTWG4KqxkXPzWV-2FkqkvzJObOOWFSncjxSqMANAu924JZOx0XkqOZlj1EuOOaddRkWz504QJV1n5X8N4j3644SB7vdelSjZckHcglBbEFxLOrz6u567rsDJ6iDdbpydMfGyl09Zg87nH23V7ggVTHxPb9KoPmnyjgLsEMS24c0nFfZvf3wZWuzE5-2B7x-2FhnXfx-2B6fzXoM-2Bu9F8ijuztKh10KyRw8-2BEbxQhJBsPBZ4KNiekdOjdFKoeOqxMRc6arF9ToHb2vM1eI-3D" target="_blank"><img height="40" src="https://cdn.nubank.com.br/colombia/icono-youtube.png" style="border-radius:3px;display:block;" width="40"></a></td></tr></table></td></tr></table><!--[if mso | IE]></td><td><![endif]--><table align="left" border="0" cellpadding="0" cellspacing="0" role="presentation" style="float:none;display:inline-table;"><tr><td style="padding:0 20px 0 0px;"><table border="0" cellpadding="0" cellspacing="0" role="presentation" style="border-radius:3px;width:40px;"><tr><td style="font-size:0;height:40px;vertical-align:middle;width:40px;"><a href="https://url3462.nu.com.co/ls/click?upn=u001.hYbpdgGXNmHJnNDZW5hKZNyKbV-2B6sd-2F6oZVpC-2FPGT-2FPX57KR5-2Foeqo-2F0skbUB-2Bq3xjAbndQsID9ezt4SFBGK6A-3D-3DMd7L_uPPJ2PGmHsJ220WnCZ4gZt4sdebFX9D0XEz37quGCBH5PiqdHvAvFK2LMvUPjcgdDSwkGTTuzRbmQD61otD0A3a5sjU5xL6MrSGD9tRyVfxFB-2BiOu-2BbtcNKsqvUeI4ki5M3xcBY70fAvGMN-2FQmhHXLkNk8NdNMBTW6lKIxgc4X70o0KSOLVxv-2FtwBtIe9ahJj-2BZhMhKoGcMfjO5graMbFHZ-2Feovp5NozwFnTWG4KqxkXPzWV-2FkqkvzJObOOWFSncjxSqMANAu924JZOx0XkqOZlj1EuOOaddRkWz504QJV1n5X8N4j3644SB7vdelSjZckHcglBbEFxLOrz6u567rtwkaI9J61XvcGyH5rzEkLm2DYLUbK-2F9ULVHpN1FoejUwPgyZqRjx4x0LCH3zeHI7n1n1GJR8IcO4Hgopu9GzUdR0QmhbExEt2kxcYSE-2F05DdQj2fGRuxtYxUVByx0Kbm7yLbygKa5mZFiRRh0wc-2BXQ-3D" target="_blank"><img height="40" src="https://cdn.nubank.com.br/colombia/icono-linkedin.png" style="border-radius:3px;display:block;" width="40"></a></td></tr></table></td></tr></table><!--[if mso | IE]></td></tr></table><![endif]--></td></tr><tr><td align="left" style="font-size:0px;padding:24px 0 0 0;word-break:break-word;"><div style="font-family:Helvetica;font-size:16px;line-height:20.8px;text-align:left;color:white;">Si tienes alguna duda, dirígete al menú ‘<strong>Ayuda</strong>’ directamente desde tu app para chatear con nosotros o escríbenos al correo ayuda@nu.com.co.<br><br>Si estás en Colombia, puedes llamarnos al 01 800 917 2000 o si estás en Bogotá, al (+57) 601 357 1102.</div></td></tr><tr><td align="left" class="unsubscription-link-class" style="font-size:0px;padding:24px 0 0 0;word-break:break-word;"><div style="font-family:Helvetica;font-size:16px;line-height:20.8px;text-align:left;color:white;">Si ya no deseas recibir nuestros correos electrónicos <a href="https://url3462.nu.com.co/ls/click?upn=u001.hYbpdgGXNmHJnNDZW5hKZK22AGugOSMsB07M-2BTSIzz9ef-2Bz3BTWgsANtDOkVPzlz23sTkGLG-2FtOqWPjP7O-2FUNnxmXOyxSVwL63qlfnZ-2Fa-2BMspNpdZ2GEBGwktOVQpagh7VBq9lW1scn8gsGH9gpQ6OVTz-2BrYT8HIB5NYmK-2B8KVOX2xDosokL6iGK7Fwo36LCnRKYPyqzly3iuxp2BPF8XHBCvgWfvvT8E5wVupHyc-2BM3P-2BTM2yGHz-2FvoGB-2Bzpq4OmBe9bjRDxdKfr9vULC7daDkAaJ98B-2B8zJ73VeZgFTFmvHzcPnSTexcvnGwyx5EF9uRjNtj9fr47frGr8m-2B3h4d1iRb9jvZqEr-2FIl1-2F2muOIJmenR-2BzxMIRCA7n0Qt-2F-2Bp-2B5j4Y2S0NO3THRG-2BHSWcPBCB-2FaOHRWQ3nZBkh9FEggEQYq3zfiKG1RYzIQMok6Iwtjcl-2FI6YGo8KNVOg-2FEx6Ax9iz2G6k2rfJ2J5f2GvuTU-2FWa2ktnrQJVqde6op-2Bxm-2FpeBg27aysfaG5A9NuSiruWbRzX-2BEYRnGE8dJaMWjTa8kGpEirLU4G7Kvtt5TatdRmgPDaYWGHrl9hR4TMR7kvicquHVXxsflxHRswg2eUnURibPq9Az-2FTWmNlCnD756GqTkch-2BvMiperoiNWgxLqfmqCY4LOJgtSj6D07uIbcRJYIAoF6q2TRAivliCAzIVixCp5JCXrszIE2-2BpoIUN6FVuWFgtaT-2BykIgaxwinwInO3tZ4tAlgcVqPiLuPH12giYzDTphcvzHYx3S-2BN341fhg-3D-3Dn7LH_uPPJ2PGmHsJ220WnCZ4gZt4sdebFX9D0XEz37quGCBH5PiqdHvAvFK2LMvUPjcgdDSwkGTTuzRbmQD61otD0A3a5sjU5xL6MrSGD9tRyVfxFB-2BiOu-2BbtcNKsqvUeI4ki5M3xcBY70fAvGMN-2FQmhHXLkNk8NdNMBTW6lKIxgc4X70o0KSOLVxv-2FtwBtIe9ahJj-2BZhMhKoGcMfjO5graMbFHZ-2Feovp5NozwFnTWG4KqxkXPzWV-2FkqkvzJObOOWFSncjxSqMANAu924JZOx0XkqOZlj1EuOOaddRkWz504QJV1n5X8N4j3644SB7vdelSjZckHcglBbEFxLOrz6u567rsK6tvUGHJdx7BuNSU3PJHJiNvdzkm2LzSQ4Lt-2FImOpyhvI2dEhmeV7CRAWVo83-2Blp98tfCvSwlJOSQn6mUvR9PPRdwT2aUIBajQkXAgrMz77cDGy-2BqFDCe5Fs6pYSGdh-2BTznqeH6E01zhVFiBRYHYo-3D" style="color: #FFFFFF; font-weight: bold">cancela aquí la suscripción.</a></div></td></tr><tr><td align="left" style="font-size:0px;padding:24px 0 0 0;word-break:break-word;"><div style="font-family:Helvetica;font-size:16px;font-weight:700;line-height:20.8px;text-align:left;color:#FFFFFF;"><a href="https://url3462.nu.com.co/ls/click?upn=u001.hYbpdgGXNmHJnNDZW5hKZPhDSdXJLx53eE22ppdTM5qV8Rn74RWNOovLAwoLKF84pprTwmBBuU-2FiEA-2Fjr9242wawscTd8yqEDTBr-2Fn9QWH-2FEmjb-2FJ1UyVA3CTsH-2FnV1G55q4_uPPJ2PGmHsJ220WnCZ4gZt4sdebFX9D0XEz37quGCBH5PiqdHvAvFK2LMvUPjcgdDSwkGTTuzRbmQD61otD0A3a5sjU5xL6MrSGD9tRyVfxFB-2BiOu-2BbtcNKsqvUeI4ki5M3xcBY70fAvGMN-2FQmhHXLkNk8NdNMBTW6lKIxgc4X70o0KSOLVxv-2FtwBtIe9ahJj-2BZhMhKoGcMfjO5graMbFHZ-2Feovp5NozwFnTWG4KqxkXPzWV-2FkqkvzJObOOWFSncjxSqMANAu924JZOx0XkqOZlj1EuOOaddRkWz504QJV1n5X8N4j3644SB7vdelSjZckHcglBbEFxLOrz6u567rg8tsWbvwu0z8F6-2FcPZlr10e-2FOajtkJNlvINnXFJJFtOmaYXg8SR0YTloNQC5rg63Ap1-2FFn7No-2FDkQ61Q750aeI-2FbkfCKs0X5Vztw1wP3qh2yMFYy9RPVkZbVeyG7OVO2L1FwGHw4azNQi7E-2FF-2BLIdY-3D" class="link-nostyle" style="color: inherit;">Política de Privacidad Nu. C.F.</a></div></td></tr><tr><td align="left" style="font-size:0px;padding:24px 0 0 0;word-break:break-word;"><div style="font-family:Helvetica;font-size:14px;font-weight:700;line-height:18.2px;text-align:left;color:#FFFFFF;">Nu. C.F. hace parte de Nubank - NIT 901.658.107-2<br>© Nu. Colombia Compañía de Financiamiento S.A. 2024</div></td></tr><tr><td align="left" style="font-size:0px;padding:10px 25px;word-break:break-word;"><table border="0" cellpadding="0" cellspacing="0" role="presentation" style="border-collapse:collapse;border-spacing:0px;"><tbody><tr><td style="width:510px;"><img height="auto" src="https://cdn.nubank.com.br/colombia/sellos-sfc-fogafin.png" style="border:0;display:block;outline:none;text-decoration:none;height:auto;width:100%;font-size:13px;" width="510"></td></tr></tbody></table></td></tr></table></div><!--[if mso | IE]></td></tr></table><![endif]--></td></tr></tbody></table></div><!--[if mso | IE]></td></tr></table><table align="center" border="0" cellpadding="0" cellspacing="0" class="unsubscription-link-class-outlook" style="width:600px;" width="600" ><tr><td style="line-height:0px;font-size:0px;mso-line-height-rule:exactly;"><![endif]--><div class="unsubscription-link-class" style="margin:0px auto;max-width:600px;"><table align="center" border="0" cellpadding="0" cellspacing="0" role="presentation" style="width:100%;"><tbody><tr><td style="direction:ltr;font-size:0px;padding:0 20px;padding-top:0;text-align:center;"><!--[if mso | IE]><table role="presentation" border="0" cellpadding="0" cellspacing="0"><tr><td class="" style="vertical-align:top;width:560px;" ><![endif]--><div class="mj-column-per-100 mj-outlook-group-fix" style="font-size:0px;text-align:left;direction:ltr;display:inline-block;vertical-align:top;width:100%;"><table border="0" cellpadding="0" cellspacing="0" role="presentation" style="vertical-align:top;" width="100%"><tr><td align="left" style="font-size:0px;padding:24px 0 0 0;word-break:break-word;"><div style="font-family:Helvetica;font-size:13px;line-height:150%;text-align:left;color:#111111;">Si ya no quieres recibir este tipo de correos, <a href="https://url3462.nu.com.co/ls/click?upn=u001.hYbpdgGXNmHJnNDZW5hKZK22AGugOSMsB07M-2BTSIzz9ef-2Bz3BTWgsANtDOkVPzlz23sTkGLG-2FtOqWPjP7O-2FUNnxmXOyxSVwL63qlfnZ-2Fa-2BMspNpdZ2GEBGwktOVQpagh7VBq9lW1scn8gsGH9gpQ6OVTz-2BrYT8HIB5NYmK-2B8KVOX2xDosokL6iGK7Fwo36LCnRKYPyqzly3iuxp2BPF8XHBCvgWfvvT8E5wVupHyc-2BM3P-2BTM2yGHz-2FvoGB-2Bzpq4OmBe9bjRDxdKfr9vULC7daDkAaJ98B-2B8zJ73VeZgFTFmvHzcPnSTexcvnGwyx5EF9uRjNtj9fr47frGr8m-2B3h4d1iRb9jvZqEr-2FIl1-2F2muOIJmenR-2BzxMIRCA7n0Qt-2F-2Bp-2B5j4Y2S0NO3THRG-2BHSWcPBCB-2FaOHRWQ3nZBkh9FEggEQYq3zfiKG1RYzIQMok6Iwtjcl-2FI6YGo8KNVOg-2FEx6Ax9iz2G6k2rfJ2J5f2GvuTU-2FWa2ktnrQJVqde6op-2Bxm-2FpeBg27aysfaG5A9NuSiruWbRzX-2BEYRnGE8dJaMWjTa8kGpEirLU4G7Kvtt5TatdRmgPDaYWGHrl9hR4TMR7kvicquHVXxsflxHRswg2eUnURibPq9Az-2FTWmNlCnD756GqTkch-2BvMiperoiNWgxLqfmqCY4LOJgtSj6D07uIbcRJYIAoF6q2TRAivliCAzIVixCp5JCXrszIE2-2BpoIUN6FVuWFgtaT-2BykIgaxwinwInO3tZ4tAlgcVqPiLuPH12giYzDTphcvzHYx3S-2BN341fhg-3D-3DdFM6_uPPJ2PGmHsJ220WnCZ4gZt4sdebFX9D0XEz37quGCBH5PiqdHvAvFK2LMvUPjcgdDSwkGTTuzRbmQD61otD0A3a5sjU5xL6MrSGD9tRyVfxFB-2BiOu-2BbtcNKsqvUeI4ki5M3xcBY70fAvGMN-2FQmhHXLkNk8NdNMBTW6lKIxgc4X70o0KSOLVxv-2FtwBtIe9ahJj-2BZhMhKoGcMfjO5graMbFHZ-2Feovp5NozwFnTWG4KqxkXPzWV-2FkqkvzJObOOWFSncjxSqMANAu924JZOx0XkqOZlj1EuOOaddRkWz504QJV1n5X8N4j3644SB7vdelSjZckHcglBbEFxLOrz6u567rlGjpn3XQoCIZ1sub8qVQVudhuyqWCnyKSQ2c3haGoPJ7H0eQ9zA7XlXaAG7N5rkNm-2F4SiY7tkOhKdIciacGnIoZZBpwQehD68IG5BtYfLcNqx1Edh3rJRQeLQ0ibkhDKcwuo7MEyJH510dRPoSBBis-3D" style="color: #111; font-weight: bold">puedes cancelar la suscripción aquí</a>.</div></td></tr></table></div><!--[if mso | IE]></td></tr></table><![endif]--></td></tr></tbody></table></div><!--[if mso | IE]></td></tr></table><table align="center" border="0" cellpadding="0" cellspacing="0" class="" style="width:600px;" width="600" ><tr><td style="line-height:0px;font-size:0px;mso-line-height-rule:exactly;"><![endif]--><div style="margin:0px auto;max-width:600px;"><table align="center" border="0" cellpadding="0" cellspacing="0" role="presentation" style="width:100%;"><tbody><tr><td style="direction:ltr;font-size:0px;padding:10px;text-align:center;"><!--[if mso | IE]><table role="presentation" border="0" cellpadding="0" cellspacing="0"><tr></tr></table><![endif]--></td></tr></tbody></table></div><!--[if mso | IE]></td></tr></table><![endif]--></div><img src="https://url3462.nu.com.co/wf/open?upn=u001.kSJWWwbEDt7ymlD7UfGr1YiKbsIFe3fVaM3nS5-2By5ka-2Bfocb3Tc-2FW32RTrojgtn5h4MrsD1uP-2Fqhr2noL6Ehq6yeAw-2BjWkDv1-2F1SZFoEJXgPGuLn5iYfaD3Aod9OLgyv-2FbA-2BULKr9zjzFJXf-2BCsFNvVau6iGvUhLLToZqvrcn2KjPhMX4Xln1Vc5h1PozAXDvBDCEzbDpfCYocBQe3UX5K49IERUhSWLvFG0Mk2kRFtsMGX-2BpiSbIieBwdRhEHfGJXRdWIK-2FZ8h1-2FruUwcCnqwha3uAgZpzVDGjOpYFOeI6YAOMph9c00IbeTUn03qsZ8YYAlqVnc9s-2Bil9G3i8bBP6M3idhnNWVMTJKgwkArg8LAZiM2PBGTJq7MA7tmA2HoMDMfeUYZoay-2BkQOogGFZbKnSdO46FvutA-2F9CYdSqHEfdfaUns4NUaLnglG8Aj5ecjqRfT0pxOrZPI8n455wnmTUkCOxf5YzAmV0NcUDixY-3D" alt="" width="1" height="1" border="0" style="height:1px !important;width:1px !important;border-width:0 !important;margin-top:0 !important;margin-bottom:0 !important;margin-right:0 !important;margin-left:0 !important;padding-top:0 !important;padding-bottom:0 !important;padding-right:0 !important;padding-left:0 !important;"></body></html>`
