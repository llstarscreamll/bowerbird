package xml

import (
	"errors"
	"testing"

	"github.com/money-path/bowerbird/apps/backend/internal/invoicing/domain"
)

func TestDIANUBL21ParserParseInvoiceXMLSuccess(t *testing.T) {
	parser := NewDIANUBL21Parser()

	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
         xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2"
         xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2">
  <cbc:ProfileID>DIAN 2.1</cbc:ProfileID>
  <cbc:ID>FE-12345</cbc:ID>
  <cbc:IssueDate>2026-05-25</cbc:IssueDate>
  <cbc:IssueTime>10:00:00-05:00</cbc:IssueTime>
  <cbc:DocumentCurrencyCode>COP</cbc:DocumentCurrencyCode>
  <cbc:UUID schemeName="CUFE-SHA384">cufe-abc-123</cbc:UUID>
  <cac:AccountingSupplierParty>
    <cac:Party>
      <cac:PartyName><cbc:Name>Proveedor SAS</cbc:Name></cac:PartyName>
      <cac:PartyTaxScheme>
        <cbc:RegistrationName>Proveedor SAS</cbc:RegistrationName>
        <cbc:CompanyID schemeID="31">900123456</cbc:CompanyID>
        <cbc:TaxLevelCode>O-13</cbc:TaxLevelCode>
      </cac:PartyTaxScheme>
    </cac:Party>
  </cac:AccountingSupplierParty>
  <cac:AccountingCustomerParty>
    <cac:Party>
      <cac:PartyName><cbc:Name>Cliente SAS</cbc:Name></cac:PartyName>
      <cac:PartyTaxScheme>
        <cbc:RegistrationName>Cliente SAS</cbc:RegistrationName>
        <cbc:CompanyID schemeID="31">901999888</cbc:CompanyID>
        <cbc:TaxLevelCode>R-99-PN</cbc:TaxLevelCode>
      </cac:PartyTaxScheme>
    </cac:Party>
  </cac:AccountingCustomerParty>
  <cac:PaymentMeans>
    <cbc:PaymentMeansCode>1</cbc:PaymentMeansCode>
  </cac:PaymentMeans>
  <cac:TaxTotal>
    <cbc:TaxAmount currencyID="COP">1900.00</cbc:TaxAmount>
    <cac:TaxSubtotal>
      <cbc:TaxableAmount currencyID="COP">10000.00</cbc:TaxableAmount>
      <cbc:Percent>19.00</cbc:Percent>
      <cac:TaxCategory>
        <cac:TaxScheme><cbc:ID>01</cbc:ID></cac:TaxScheme>
      </cac:TaxCategory>
    </cac:TaxSubtotal>
  </cac:TaxTotal>
  <cac:LegalMonetaryTotal>
    <cbc:LineExtensionAmount currencyID="COP">10000.00</cbc:LineExtensionAmount>
    <cbc:TaxExclusiveAmount currencyID="COP">10000.00</cbc:TaxExclusiveAmount>
    <cbc:TaxInclusiveAmount currencyID="COP">11900.00</cbc:TaxInclusiveAmount>
    <cbc:PayableAmount currencyID="COP">11900.00</cbc:PayableAmount>
  </cac:LegalMonetaryTotal>
  <cac:InvoiceLine>
    <cbc:ID>1</cbc:ID>
    <cbc:InvoicedQuantity unitCode="EA">2</cbc:InvoicedQuantity>
    <cbc:LineExtensionAmount currencyID="COP">10000.00</cbc:LineExtensionAmount>
    <cac:TaxTotal>
      <cbc:TaxAmount currencyID="COP">1900.00</cbc:TaxAmount>
    </cac:TaxTotal>
    <cac:Item>
      <cbc:Description>Servicio de prueba</cbc:Description>
      <cbc:Name>Servicio</cbc:Name>
    </cac:Item>
    <cac:Price>
      <cbc:PriceAmount currencyID="COP">5000.00</cbc:PriceAmount>
    </cac:Price>
  </cac:InvoiceLine>
</Invoice>`)

	doc, err := parser.ParseInvoiceXML(xmlData)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if doc.CUFE != "cufe-abc-123" {
		t.Fatalf("expected CUFE parsed, got %q", doc.CUFE)
	}
	if doc.Issuer.CompanyID != "900123456" {
		t.Fatalf("expected issuer id parsed, got %q", doc.Issuer.CompanyID)
	}
	if doc.Receiver.CompanyID != "901999888" {
		t.Fatalf("expected receiver id parsed, got %q", doc.Receiver.CompanyID)
	}
	if doc.PaymentMeansCode != "1" {
		t.Fatalf("expected payment means code parsed, got %q", doc.PaymentMeansCode)
	}
	if len(doc.TaxTotals) != 1 || doc.TaxTotals[0].TaxCode != "01" {
		t.Fatalf("expected tax totals parsed, got %#v", doc.TaxTotals)
	}
	if len(doc.Lines) != 1 {
		t.Fatalf("expected invoice line parsed, got %d", len(doc.Lines))
	}
}

func TestDIANUBL21ParserParseInvoiceXMLErrorsOnMissingCUFE(t *testing.T) {
	parser := NewDIANUBL21Parser()

	xmlData := []byte(`<Invoice><ID>1</ID></Invoice>`)
	_, err := parser.ParseInvoiceXML(xmlData)
	if !errors.Is(err, domain.ErrMissingCUFE) {
		t.Fatalf("expected missing CUFE error, got %v", err)
	}
}

func TestDIANUBL21ParserParseInvoiceXMLErrorsOnMalformedXML(t *testing.T) {
	parser := NewDIANUBL21Parser()

	_, err := parser.ParseInvoiceXML([]byte(`<Invoice><ID>1</ID>`))
	if !errors.Is(err, domain.ErrMalformedXML) {
		t.Fatalf("expected malformed xml error, got %v", err)
	}
}

func TestDIANUBL21ParserParseInvoiceXMLErrorsOnMissingParties(t *testing.T) {
	parser := NewDIANUBL21Parser()

	xmlData := []byte(`
<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2">
  <cbc:UUID>cufe</cbc:UUID>
  <cac:InvoiceLine xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2">
    <cbc:ID>1</cbc:ID>
  </cac:InvoiceLine>
</Invoice>`)

	_, err := parser.ParseInvoiceXML(xmlData)
	if !errors.Is(err, domain.ErrMissingIssuer) {
		t.Fatalf("expected missing issuer error, got %v", err)
	}
}
