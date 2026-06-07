package xml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/bowerbird/internal/invoices/application/ports"
	"github.com/bowerbird/internal/invoices/domain"
)

type DianUBL21Parser struct{}

func NewDianUBL21Parser() *DianUBL21Parser {
	return &DianUBL21Parser{}
}

func (p *DianUBL21Parser) ParseInvoiceXML(data []byte) (*domain.InvoiceDocument, error) {
	invoice, err := decodeInvoiceDocument(data)
	if err != nil {
		return nil, err
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
			LineID:          strings.TrimSpace(line.ID.Value),
			ItemDescription: firstNonEmpty(firstNonEmpty(line.Item.Descriptions...), strings.TrimSpace(line.Item.Name)),
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
				Percent:   parseFloat(firstNonEmpty(subtotal.Percent, subtotal.TaxCategory.Percent)),
			})
		}
	}

	paymentMeansCode := ""
	for _, paymentMeans := range invoice.PaymentMeans {
		paymentMeansCode = firstNonEmpty(paymentMeansCode, paymentMeans.PaymentMeansCode)
		if paymentMeansCode != "" {
			break
		}
	}

	doc := &domain.InvoiceDocument{
		ProfileID:        strings.TrimSpace(invoice.ProfileID),
		InvoiceID:        strings.TrimSpace(invoice.ID),
		IssueDate:        strings.TrimSpace(invoice.IssueDate),
		IssueTime:        strings.TrimSpace(invoice.IssueTime),
		CurrencyCode:     strings.TrimSpace(invoice.DocumentCurrencyCode),
		CUFE:             cufe,
		PaymentMeansCode: strings.TrimSpace(paymentMeansCode),
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

func decodeInvoiceDocument(data []byte) (ublInvoice, error) {
	var invoice ublInvoice
	if err := decodeXML(data, &invoice); err == nil {
		if strings.EqualFold(invoice.XMLName.Local, "Invoice") {
			return invoice, nil
		}
	}

	embeddedInvoiceXML, err := extractEmbeddedInvoice(data)
	if err != nil {
		return ublInvoice{}, fmt.Errorf("%w: %v", domain.ErrMalformedXML, err)
	}

	if err := decodeXML([]byte(embeddedInvoiceXML), &invoice); err != nil {
		return ublInvoice{}, fmt.Errorf("%w: %v", domain.ErrMalformedXML, err)
	}

	return invoice, nil
}

func decodeXML(data []byte, v any) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	return decoder.Decode(v)
}

func extractEmbeddedInvoice(data []byte) (string, error) {
	var attached attachedDocument
	if err := decodeXML(data, &attached); err != nil {
		return "", err
	}

	for _, candidate := range attached.AllDescriptions() {
		invoiceXML := strings.TrimSpace(candidate)
		if strings.Contains(invoiceXML, "<Invoice") {
			return invoiceXML, nil
		}
	}

	return "", fmt.Errorf("embedded invoice xml not found")
}

func mapParty(input partyContainer) domain.Party {
	partyTaxScheme := input.Party.PartyTaxScheme
	companyID := firstNonEmpty(partyTaxScheme.CompanyID.Value, input.Party.PartyLegalEntity.CompanyID.Value, input.Party.PartyIdentification.ID.Value)
	schemeID := firstNonEmpty(partyTaxScheme.CompanyID.SchemeID, input.Party.PartyLegalEntity.CompanyID.SchemeID, input.Party.PartyIdentification.ID.SchemeID)
	name := firstNonEmpty(input.Party.PartyName.Name, partyTaxScheme.RegistrationName, input.Party.PartyLegalEntity.RegistrationName)
	return domain.Party{
		Name:           strings.TrimSpace(name),
		CompanyID:      strings.TrimSpace(companyID),
		SchemeID:       strings.TrimSpace(schemeID),
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
	UBLVersionID            string             `xml:"UBLVersionID"`
	CustomizationID         string             `xml:"CustomizationID"`
	ProfileID               string             `xml:"ProfileID"`
	ProfileExecutionID      string             `xml:"ProfileExecutionID"`
	ID                      string             `xml:"ID"`
	UUID                    valueWithAttrs     `xml:"UUID"`
	IssueDate               string             `xml:"IssueDate"`
	IssueTime               string             `xml:"IssueTime"`
	InvoiceTypeCode         string             `xml:"InvoiceTypeCode"`
	DocumentCurrencyCode    string             `xml:"DocumentCurrencyCode"`
	LineCountNumeric        string             `xml:"LineCountNumeric"`
	AccountingSupplierParty partyContainer     `xml:"AccountingSupplierParty"`
	AccountingCustomerParty partyContainer     `xml:"AccountingCustomerParty"`
	PaymentMeans            []paymentMeans     `xml:"PaymentMeans"`
	TaxTotals               []taxTotal         `xml:"TaxTotal"`
	LegalMonetaryTotal      legalMonetaryTotal `xml:"LegalMonetaryTotal"`
	InvoiceLines            []invoiceLine      `xml:"InvoiceLine"`
}

type attachedDocument struct {
	XMLName                  xml.Name   `xml:"AttachedDocument"`
	Attachment               attachment `xml:"Attachment"`
	ParentDocumentAttachment attachment `xml:"ParentDocumentLineReference>DocumentReference>Attachment"`
}

func (d attachedDocument) AllDescriptions() []string {
	descriptions := make([]string, 0, 2)
	descriptions = append(descriptions, d.Attachment.ExternalReference.Description)
	descriptions = append(descriptions, d.ParentDocumentAttachment.ExternalReference.Description)
	return descriptions
}

type attachment struct {
	ExternalReference externalReference `xml:"ExternalReference"`
}

type externalReference struct {
	Description string `xml:"Description"`
}

type partyContainer struct {
	Party party `xml:"Party"`
}

type party struct {
	IndustryClassificationCode string              `xml:"IndustryClassificationCode"`
	PartyIdentification        partyIdentification `xml:"PartyIdentification"`
	PartyName                  partyName           `xml:"PartyName"`
	PhysicalLocation           physicalLocation    `xml:"PhysicalLocation"`
	PartyTaxScheme             partyTaxScheme      `xml:"PartyTaxScheme"`
	PartyLegalEntity           partyLegalEntity    `xml:"PartyLegalEntity"`
	Contact                    contact             `xml:"Contact"`
}

type partyIdentification struct {
	ID valueWithAttrs `xml:"ID"`
}

type partyName struct {
	Name string `xml:"Name"`
}

type partyTaxScheme struct {
	RegistrationName    string         `xml:"RegistrationName"`
	CompanyID           valueWithAttrs `xml:"CompanyID"`
	TaxLevelCode        string         `xml:"TaxLevelCode"`
	RegistrationAddress address        `xml:"RegistrationAddress"`
	TaxScheme           taxScheme      `xml:"TaxScheme"`
}

type partyLegalEntity struct {
	RegistrationName string         `xml:"RegistrationName"`
	CompanyID        valueWithAttrs `xml:"CompanyID"`
}

type contact struct {
	Name           string `xml:"Name"`
	Telephone      string `xml:"Telephone"`
	ElectronicMail string `xml:"ElectronicMail"`
}

type physicalLocation struct {
	Address address `xml:"Address"`
}

type address struct {
	ID                   string      `xml:"ID"`
	CityName             string      `xml:"CityName"`
	PostalZone           string      `xml:"PostalZone"`
	CountrySubentity     string      `xml:"CountrySubentity"`
	CountrySubentityCode string      `xml:"CountrySubentityCode"`
	AddressLine          addressLine `xml:"AddressLine"`
	Country              country     `xml:"Country"`
}

type addressLine struct {
	Line string `xml:"Line"`
}

type country struct {
	IdentificationCode string `xml:"IdentificationCode"`
	Name               string `xml:"Name"`
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
	Percent   string    `xml:"Percent"`
	TaxScheme taxScheme `xml:"TaxScheme"`
}

type taxScheme struct {
	ID string `xml:"ID"`
}

type legalMonetaryTotal struct {
	LineExtensionAmount   valueWithAttrs `xml:"LineExtensionAmount"`
	TaxExclusiveAmount    valueWithAttrs `xml:"TaxExclusiveAmount"`
	TaxInclusiveAmount    valueWithAttrs `xml:"TaxInclusiveAmount"`
	PrepaidAmount         valueWithAttrs `xml:"PrepaidAmount"`
	PayableRoundingAmount valueWithAttrs `xml:"PayableRoundingAmount"`
	PayableAmount         valueWithAttrs `xml:"PayableAmount"`
}

type invoiceLine struct {
	ID                    valueWithAttrs    `xml:"ID"`
	InvoicedQuantity      valueWithAttrs    `xml:"InvoicedQuantity"`
	LineExtensionAmount   valueWithAttrs    `xml:"LineExtensionAmount"`
	FreeOfChargeIndicator string            `xml:"FreeOfChargeIndicator"`
	AllowanceCharges      []allowanceCharge `xml:"AllowanceCharge"`
	TaxTotals             []taxTotal        `xml:"TaxTotal"`
	Item                  item              `xml:"Item"`
	Price                 price             `xml:"Price"`
}

type allowanceCharge struct {
	ID                      string         `xml:"ID"`
	ChargeIndicator         string         `xml:"ChargeIndicator"`
	AllowanceChargeReason   string         `xml:"AllowanceChargeReason"`
	MultiplierFactorNumeric string         `xml:"MultiplierFactorNumeric"`
	Amount                  valueWithAttrs `xml:"Amount"`
	BaseAmount              valueWithAttrs `xml:"BaseAmount"`
}

type item struct {
	Descriptions               []string           `xml:"Description"`
	Name                       string             `xml:"Name"`
	PackSizeNumeric            string             `xml:"PackSizeNumeric"`
	SellersItemIdentification  itemIdentification `xml:"SellersItemIdentification"`
	StandardItemIdentification itemIdentification `xml:"StandardItemIdentification"`
}

type itemIdentification struct {
	ID valueWithAttrs `xml:"ID"`
}

type price struct {
	PriceAmount  valueWithAttrs `xml:"PriceAmount"`
	BaseQuantity valueWithAttrs `xml:"BaseQuantity"`
}

type valueWithAttrs struct {
	Value      string `xml:",chardata"`
	UnitCode   string `xml:"unitCode,attr"`
	CurrencyID string `xml:"currencyID,attr"`
	SchemeID   string `xml:"schemeID,attr"`
	SchemeName string `xml:"schemeName,attr"`
}

var _ ports.InvoiceXMLExtractor = (*DianUBL21Parser)(nil)
