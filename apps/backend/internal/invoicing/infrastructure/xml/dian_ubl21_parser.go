package xml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/money-path/bowerbird/apps/backend/internal/invoicing/domain"
)

type DIANUBL21Parser struct{}

func NewDIANUBL21Parser() *DIANUBL21Parser {
	return &DIANUBL21Parser{}
}

func (p *DIANUBL21Parser) ParseInvoiceXML(data []byte) (*domain.InvoiceDocument, error) {
	var invoice ublInvoice
	decoder := xml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&invoice); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrMalformedXML, err)
	}

	cufe := strings.TrimSpace(invoice.UUID.Value)
	if cufe == "" {
		return nil, domain.ErrMissingCUFE
	}

	issuer := mapParty(invoice.AccountingSupplierParty)
	if strings.TrimSpace(issuer.Name) == "" || strings.TrimSpace(issuer.CompanyID) == "" {
		return nil, domain.ErrMissingIssuer
	}

	receiver := mapParty(invoice.AccountingCustomerParty)
	if strings.TrimSpace(receiver.Name) == "" || strings.TrimSpace(receiver.CompanyID) == "" {
		return nil, domain.ErrMissingReceiver
	}

	lines := make([]domain.InvoiceLine, 0, len(invoice.InvoiceLines))
	for _, line := range invoice.InvoiceLines {
		mapped := domain.InvoiceLine{
			LineID:          strings.TrimSpace(line.ID),
			ItemDescription: firstNonEmpty(strings.TrimSpace(line.Item.Description), strings.TrimSpace(line.Item.Name)),
			Quantity:        parseFloat(line.InvoicedQuantity.Value),
			UnitCode:        strings.TrimSpace(line.InvoicedQuantity.UnitCode),
			UnitPrice:       parseFloat(line.Price.PriceAmount.Value),
			LineExtension:   parseFloat(line.LineExtensionAmount.Value),
		}
		for _, tax := range line.TaxTotals {
			mapped.TaxAmount += parseFloat(tax.TaxAmount.Value)
		}
		lines = append(lines, mapped)
	}
	if len(lines) == 0 {
		return nil, domain.ErrMissingLineItems
	}

	taxTotals := make([]domain.TaxTotal, 0, len(invoice.TaxTotals))
	for _, total := range invoice.TaxTotals {
		for _, subtotal := range total.TaxSubtotals {
			taxTotals = append(taxTotals, domain.TaxTotal{
				TaxAmount: parseFloat(total.TaxAmount.Value),
				Taxable:   parseFloat(subtotal.TaxableAmount.Value),
				TaxCode:   strings.TrimSpace(subtotal.TaxCategory.TaxScheme.ID),
				Percent:   parseFloat(subtotal.Percent),
			})
		}
	}

	doc := &domain.InvoiceDocument{
		ProfileID:        strings.TrimSpace(invoice.ProfileID),
		InvoiceID:        strings.TrimSpace(invoice.ID),
		IssueDate:        strings.TrimSpace(invoice.IssueDate),
		IssueTime:        strings.TrimSpace(invoice.IssueTime),
		CurrencyCode:     strings.TrimSpace(invoice.DocumentCurrencyCode),
		CUFE:             cufe,
		PaymentMeansCode: strings.TrimSpace(invoice.PaymentMeans.PaymentMeansCode),
		Issuer:           issuer,
		Receiver:         receiver,
		TaxTotals:        taxTotals,
		LineExtension:    parseFloat(invoice.LegalMonetaryTotal.LineExtensionAmount.Value),
		TaxExclusive:     parseFloat(invoice.LegalMonetaryTotal.TaxExclusiveAmount.Value),
		TaxInclusive:     parseFloat(invoice.LegalMonetaryTotal.TaxInclusiveAmount.Value),
		PayableAmount:    parseFloat(invoice.LegalMonetaryTotal.PayableAmount.Value),
		Lines:            lines,
		RawData:          data,
	}
	if err := doc.Validate(); err != nil {
		return nil, err
	}

	return doc, nil
}

func mapParty(input partyContainer) domain.Party {
	partyTaxScheme := input.Party.PartyTaxScheme
	return domain.Party{
		Name:           strings.TrimSpace(input.Party.PartyName.Name),
		CompanyID:      strings.TrimSpace(partyTaxScheme.CompanyID.Value),
		SchemeID:       strings.TrimSpace(partyTaxScheme.CompanyID.SchemeID),
		TaxLevelCode:   strings.TrimSpace(partyTaxScheme.TaxLevelCode),
		RegistrationID: strings.TrimSpace(partyTaxScheme.RegistrationName),
	}
}

func parseFloat(value string) float64 {
	v := strings.TrimSpace(value)
	if v == "" {
		return 0
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0
	}
	return f
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

type ublInvoice struct {
	XMLName                 xml.Name           `xml:"Invoice"`
	ID                      string             `xml:"ID"`
	ProfileID               string             `xml:"ProfileID"`
	IssueDate               string             `xml:"IssueDate"`
	IssueTime               string             `xml:"IssueTime"`
	DocumentCurrencyCode    string             `xml:"DocumentCurrencyCode"`
	UUID                    valueWithAttrs     `xml:"UUID"`
	AccountingSupplierParty partyContainer     `xml:"AccountingSupplierParty"`
	AccountingCustomerParty partyContainer     `xml:"AccountingCustomerParty"`
	PaymentMeans            paymentMeans       `xml:"PaymentMeans"`
	TaxTotals               []taxTotal         `xml:"TaxTotal"`
	LegalMonetaryTotal      legalMonetaryTotal `xml:"LegalMonetaryTotal"`
	InvoiceLines            []invoiceLine      `xml:"InvoiceLine"`
}

type partyContainer struct {
	Party party `xml:"Party"`
}

type party struct {
	PartyName      partyName      `xml:"PartyName"`
	PartyTaxScheme partyTaxScheme `xml:"PartyTaxScheme"`
}

type partyName struct {
	Name string `xml:"Name"`
}

type partyTaxScheme struct {
	RegistrationName string         `xml:"RegistrationName"`
	CompanyID        valueWithAttrs `xml:"CompanyID"`
	TaxLevelCode     string         `xml:"TaxLevelCode"`
}

type paymentMeans struct {
	PaymentMeansCode string `xml:"PaymentMeansCode"`
}

type taxTotal struct {
	TaxAmount    valueWithAttrs `xml:"TaxAmount"`
	TaxSubtotals []taxSubtotal  `xml:"TaxSubtotal"`
}

type taxSubtotal struct {
	TaxableAmount valueWithAttrs `xml:"TaxableAmount"`
	Percent       string         `xml:"Percent"`
	TaxCategory   taxCategory    `xml:"TaxCategory"`
}

type taxCategory struct {
	TaxScheme taxScheme `xml:"TaxScheme"`
}

type taxScheme struct {
	ID string `xml:"ID"`
}

type legalMonetaryTotal struct {
	LineExtensionAmount valueWithAttrs `xml:"LineExtensionAmount"`
	TaxExclusiveAmount  valueWithAttrs `xml:"TaxExclusiveAmount"`
	TaxInclusiveAmount  valueWithAttrs `xml:"TaxInclusiveAmount"`
	PayableAmount       valueWithAttrs `xml:"PayableAmount"`
}

type invoiceLine struct {
	ID                  string         `xml:"ID"`
	InvoicedQuantity    valueWithAttrs `xml:"InvoicedQuantity"`
	LineExtensionAmount valueWithAttrs `xml:"LineExtensionAmount"`
	Item                item           `xml:"Item"`
	Price               price          `xml:"Price"`
	TaxTotals           []taxTotal     `xml:"TaxTotal"`
}

type item struct {
	Description string `xml:"Description"`
	Name        string `xml:"Name"`
}

type price struct {
	PriceAmount valueWithAttrs `xml:"PriceAmount"`
}

type valueWithAttrs struct {
	Value      string `xml:",chardata"`
	UnitCode   string `xml:"unitCode,attr"`
	CurrencyID string `xml:"currencyID,attr"`
	SchemeID   string `xml:"schemeID,attr"`
}
