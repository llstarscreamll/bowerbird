package domain

const (
	ProductInvoicing = "invoicing"
	ProductMail      = "mail"

	FeatureInvoicingWorkspace        = "invoicing.workspace"
	FeatureInvoicingCaptureFromEmail = "invoicing.capture_from_email"
	FeatureMailInbox                 = "mail.inbox"
	FeatureMailOrganize              = "mail.organize"
	FeatureMailSend                  = "mail.send"

	StatusActive    = "active"
	StatusTrial     = "trial"
	StatusSuspended = "suspended"
	StatusExpired   = "expired"

	SourceManual = "manual"
	SourceTrial  = "trial"

	PlatformRoleOperator = "operator"
)

type Feature struct {
	Key      string
	Product  string
	Required bool
}

type Product struct {
	Key      string
	Features []Feature
}

func Catalog() []Product {
	return []Product{
		{
			Key: ProductInvoicing,
			Features: []Feature{
				{Key: FeatureInvoicingWorkspace, Product: ProductInvoicing, Required: true},
				{Key: FeatureInvoicingCaptureFromEmail, Product: ProductInvoicing, Required: true},
			},
		},
		{
			Key: ProductMail,
			Features: []Feature{
				{Key: FeatureMailInbox, Product: ProductMail, Required: true},
				{Key: FeatureMailOrganize, Product: ProductMail, Required: true},
				{Key: FeatureMailSend, Product: ProductMail, Required: false},
			},
		},
	}
}

func DefaultPackFeatureKeys() []string {
	return []string{
		FeatureInvoicingWorkspace,
		FeatureInvoicingCaptureFromEmail,
		FeatureMailInbox,
		FeatureMailOrganize,
	}
}

func RequiredFeatureKeys(productKey string) []string {
	keys := make([]string, 0)
	for _, product := range Catalog() {
		if product.Key != productKey {
			continue
		}
		for _, feature := range product.Features {
			if feature.Required {
				keys = append(keys, feature.Key)
			}
		}
	}
	return keys
}

func AllFeatureKeys(productKey string) []string {
	keys := make([]string, 0)
	for _, product := range Catalog() {
		if product.Key != productKey {
			continue
		}
		for _, feature := range product.Features {
			keys = append(keys, feature.Key)
		}
	}
	return keys
}

func FeatureExists(featureKey string) bool {
	for _, product := range Catalog() {
		for _, feature := range product.Features {
			if feature.Key == featureKey {
				return true
			}
		}
	}
	return false
}

func ProductExists(productKey string) bool {
	for _, product := range Catalog() {
		if product.Key == productKey {
			return true
		}
	}
	return false
}
