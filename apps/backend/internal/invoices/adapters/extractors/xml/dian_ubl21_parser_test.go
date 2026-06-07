package xml

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bowerbird/internal/invoices/domain"
)

func TestDIANUBL21ParserParseInvoiceXMLSuccess(t *testing.T) {
	parser := NewDianUBL21Parser()

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

func TestDIANUBL21ParserParseInvoiceXMLFromAttachedDocumentRealExample(t *testing.T) {
	parser := NewDianUBL21Parser()
	xmlData := mustReadFixture(t, "fv90027737040532300457505.xml")

	doc, err := parser.ParseInvoiceXML(xmlData)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if doc.InvoiceID != "FETA19245" {
		t.Fatalf("expected invoice id FETA19245, got %q", doc.InvoiceID)
	}
	if doc.CUFE != "6a9b25b04dcdcc81b4e0893b4ab3f3dcd94edc919ea30b09ecb5839214b5b50dcd902508f97ee1233e5e6ec5057aa70f" {
		t.Fatalf("unexpected CUFE: %q", doc.CUFE)
	}
	if doc.PaymentMeansCode != "48" {
		t.Fatalf("expected first payment means code 48, got %q", doc.PaymentMeansCode)
	}
	if doc.Issuer.Name != "I SHOP COLOMBIA SAS" || doc.Issuer.CompanyID != "900277370" {
		t.Fatalf("unexpected issuer: %#v", doc.Issuer)
	}
	if doc.Receiver.Name != "JOHAN ALVAREZ" || doc.Receiver.CompanyID != "1057581292" {
		t.Fatalf("unexpected receiver: %#v", doc.Receiver)
	}
	if len(doc.TaxTotals) != 1 {
		t.Fatalf("expected 1 tax total, got %d", len(doc.TaxTotals))
	}
	if !almostEqual(doc.TaxTotals[0].Percent, 19) {
		t.Fatalf("expected tax percent 19, got %v", doc.TaxTotals[0].Percent)
	}
	if len(doc.Lines) != 1 {
		t.Fatalf("expected 1 invoice line, got %d", len(doc.Lines))
	}
	if !strings.Contains(doc.Lines[0].ItemDescription, "MacBook Air 13") {
		t.Fatalf("unexpected line description: %q", doc.Lines[0].ItemDescription)
	}
}

func TestDecodeInvoiceDocumentRealExampleMatchesStructure(t *testing.T) {
	xmlData := mustReadFixture(t, "fv90027737040532300457505.xml")

	invoice, err := decodeInvoiceDocument(xmlData)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if invoice.UBLVersionID != "UBL 2.1" || invoice.CustomizationID != "10" {
		t.Fatalf("unexpected invoice envelope fields: ubl=%q customization=%q", invoice.UBLVersionID, invoice.CustomizationID)
	}
	if invoice.ProfileExecutionID != "1" || invoice.InvoiceTypeCode != "01" {
		t.Fatalf("unexpected profile/invoice type: execution=%q type=%q", invoice.ProfileExecutionID, invoice.InvoiceTypeCode)
	}
	if len(invoice.PaymentMeans) != 2 || invoice.PaymentMeans[0].PaymentMeansCode != "48" || invoice.PaymentMeans[1].PaymentMeansCode != "ZZZ" {
		t.Fatalf("unexpected payment means: %#v", invoice.PaymentMeans)
	}

	supplier := invoice.AccountingSupplierParty.Party
	if supplier.PhysicalLocation.Address.CityName != "TUNJA" {
		t.Fatalf("unexpected supplier city: %q", supplier.PhysicalLocation.Address.CityName)
	}
	if supplier.PartyTaxScheme.CompanyID.Value != "900277370" {
		t.Fatalf("unexpected supplier company id: %q", supplier.PartyTaxScheme.CompanyID.Value)
	}
	if supplier.Contact.ElectronicMail != "dte_9002773704@dte.paperless.com.co" {
		t.Fatalf("unexpected supplier email: %q", supplier.Contact.ElectronicMail)
	}

	customer := invoice.AccountingCustomerParty.Party
	if customer.PartyIdentification.ID.Value != "1057581292" {
		t.Fatalf("unexpected customer id: %q", customer.PartyIdentification.ID.Value)
	}
	if customer.PartyTaxScheme.TaxLevelCode != "R-99-PN" {
		t.Fatalf("unexpected customer tax level code: %q", customer.PartyTaxScheme.TaxLevelCode)
	}

	if invoice.LegalMonetaryTotal.PrepaidAmount.Value != "0.00" || invoice.LegalMonetaryTotal.PayableRoundingAmount.Value != "0.00" {
		t.Fatalf("unexpected legal monetary optional fields: prepaid=%q rounding=%q", invoice.LegalMonetaryTotal.PrepaidAmount.Value, invoice.LegalMonetaryTotal.PayableRoundingAmount.Value)
	}

	if len(invoice.InvoiceLines) != 1 {
		t.Fatalf("expected one invoice line, got %d", len(invoice.InvoiceLines))
	}
	line := invoice.InvoiceLines[0]
	if line.ID.Value != "1" || line.ID.SchemeID != "0" {
		t.Fatalf("unexpected line id: %#v", line.ID)
	}
	if len(line.AllowanceCharges) != 1 || line.AllowanceCharges[0].Amount.Value != "168067.23" {
		t.Fatalf("unexpected allowance charge: %#v", line.AllowanceCharges)
	}
	if len(line.Item.Descriptions) != 2 || !strings.Contains(line.Item.Descriptions[1], "SERIE/IMEI") {
		t.Fatalf("unexpected item descriptions: %#v", line.Item.Descriptions)
	}
	if line.Item.SellersItemIdentification.ID.Value != "MGND3LA/A" || line.Item.StandardItemIdentification.ID.Value != "MGND3LA/A" {
		t.Fatalf("unexpected item ids: seller=%q standard=%q", line.Item.SellersItemIdentification.ID.Value, line.Item.StandardItemIdentification.ID.Value)
	}
	if !almostEqual(parseFloat(line.Price.PriceAmount.Value), 4452941.1799999997) {
		t.Fatalf("unexpected price amount: %q", line.Price.PriceAmount.Value)
	}

	if len(invoice.TaxTotals) != 1 || len(invoice.TaxTotals[0].TaxSubtotals) != 1 {
		t.Fatalf("unexpected tax totals: %#v", invoice.TaxTotals)
	}
	if invoice.TaxTotals[0].TaxSubtotals[0].TaxCategory.Percent != "19.00" {
		t.Fatalf("unexpected tax percent location value: %q", invoice.TaxTotals[0].TaxSubtotals[0].TaxCategory.Percent)
	}
}

func TestDecodeInvoiceDocumentFindsInvoiceInParentDocumentAttachment(t *testing.T) {
	invoiceXML := `<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2" xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2"><cbc:ProfileID>DIAN 2.1</cbc:ProfileID><cbc:ID>PARENT-1</cbc:ID><cbc:IssueDate>2026-05-25</cbc:IssueDate><cbc:IssueTime>10:00:00-05:00</cbc:IssueTime><cbc:DocumentCurrencyCode>COP</cbc:DocumentCurrencyCode><cbc:UUID>parent-cufe</cbc:UUID><cac:AccountingSupplierParty><cac:Party><cac:PartyName><cbc:Name>Proveedor</cbc:Name></cac:PartyName><cac:PartyTaxScheme><cbc:RegistrationName>Proveedor</cbc:RegistrationName><cbc:CompanyID>900123</cbc:CompanyID></cac:PartyTaxScheme></cac:Party></cac:AccountingSupplierParty><cac:AccountingCustomerParty><cac:Party><cac:PartyName><cbc:Name>Cliente</cbc:Name></cac:PartyName><cac:PartyTaxScheme><cbc:RegistrationName>Cliente</cbc:RegistrationName><cbc:CompanyID>901456</cbc:CompanyID></cac:PartyTaxScheme></cac:Party></cac:AccountingCustomerParty><cac:PaymentMeans><cbc:PaymentMeansCode>1</cbc:PaymentMeansCode></cac:PaymentMeans><cac:TaxTotal><cbc:TaxAmount>19</cbc:TaxAmount><cac:TaxSubtotal><cbc:TaxableAmount>100</cbc:TaxableAmount><cac:TaxCategory><cbc:Percent>19</cbc:Percent><cac:TaxScheme><cbc:ID>01</cbc:ID></cac:TaxScheme></cac:TaxCategory></cac:TaxSubtotal></cac:TaxTotal><cac:LegalMonetaryTotal><cbc:LineExtensionAmount>100</cbc:LineExtensionAmount><cbc:TaxExclusiveAmount>100</cbc:TaxExclusiveAmount><cbc:TaxInclusiveAmount>119</cbc:TaxInclusiveAmount><cbc:PayableAmount>119</cbc:PayableAmount></cac:LegalMonetaryTotal><cac:InvoiceLine><cbc:ID>1</cbc:ID><cbc:InvoicedQuantity unitCode="EA">1</cbc:InvoicedQuantity><cbc:LineExtensionAmount>100</cbc:LineExtensionAmount><cac:TaxTotal><cbc:TaxAmount>19</cbc:TaxAmount></cac:TaxTotal><cac:Item><cbc:Description>Item</cbc:Description></cac:Item><cac:Price><cbc:PriceAmount>100</cbc:PriceAmount></cac:Price></cac:InvoiceLine></Invoice>`

	attached := `<AttachedDocument xmlns="urn:oasis:names:specification:ubl:schema:xsd:AttachedDocument-2" xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2"><cac:Attachment><cac:ExternalReference><cbc:Description><![CDATA[<ApplicationResponse/>]]></cbc:Description></cac:ExternalReference></cac:Attachment><cac:ParentDocumentLineReference><cac:DocumentReference><cac:Attachment><cac:ExternalReference><cbc:Description><![CDATA[` + invoiceXML + `]]></cbc:Description></cac:ExternalReference></cac:Attachment></cac:DocumentReference></cac:ParentDocumentLineReference></AttachedDocument>`

	invoice, err := decodeInvoiceDocument([]byte(attached))
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if invoice.ID != "PARENT-1" {
		t.Fatalf("expected fallback invoice id PARENT-1, got %q", invoice.ID)
	}
}

func TestDIANUBL21ParserParseInvoiceXMLErrorsOnMissingCUFE(t *testing.T) {
	parser := NewDianUBL21Parser()

	xmlData := []byte(`<Invoice><ID>1</ID></Invoice>`)
	_, err := parser.ParseInvoiceXML(xmlData)
	if !errors.Is(err, domain.ErrMissingCUFE) {
		t.Fatalf("expected missing CUFE error, got %v", err)
	}
}

func TestDIANUBL21ParserParseInvoiceXMLSupportsDirectAndAttached(t *testing.T) {
	parser := NewDianUBL21Parser()
	invoiceXML := `<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2" xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2"><cbc:ProfileID>DIAN 2.1</cbc:ProfileID><cbc:ID>FLEX-1</cbc:ID><cbc:IssueDate>2026-05-25</cbc:IssueDate><cbc:IssueTime>10:00:00-05:00</cbc:IssueTime><cbc:DocumentCurrencyCode>COP</cbc:DocumentCurrencyCode><cbc:UUID>flex-cufe</cbc:UUID><cac:AccountingSupplierParty><cac:Party><cac:PartyName><cbc:Name>Proveedor</cbc:Name></cac:PartyName><cac:PartyTaxScheme><cbc:RegistrationName>Proveedor</cbc:RegistrationName><cbc:CompanyID>900123</cbc:CompanyID></cac:PartyTaxScheme></cac:Party></cac:AccountingSupplierParty><cac:AccountingCustomerParty><cac:Party><cac:PartyName><cbc:Name>Cliente</cbc:Name></cac:PartyName><cac:PartyTaxScheme><cbc:RegistrationName>Cliente</cbc:RegistrationName><cbc:CompanyID>901456</cbc:CompanyID></cac:PartyTaxScheme></cac:Party></cac:AccountingCustomerParty><cac:PaymentMeans><cbc:PaymentMeansCode>1</cbc:PaymentMeansCode></cac:PaymentMeans><cac:TaxTotal><cbc:TaxAmount>19</cbc:TaxAmount><cac:TaxSubtotal><cbc:TaxableAmount>100</cbc:TaxableAmount><cac:TaxCategory><cbc:Percent>19</cbc:Percent><cac:TaxScheme><cbc:ID>01</cbc:ID></cac:TaxScheme></cac:TaxCategory></cac:TaxSubtotal></cac:TaxTotal><cac:LegalMonetaryTotal><cbc:LineExtensionAmount>100</cbc:LineExtensionAmount><cbc:TaxExclusiveAmount>100</cbc:TaxExclusiveAmount><cbc:TaxInclusiveAmount>119</cbc:TaxInclusiveAmount><cbc:PayableAmount>119</cbc:PayableAmount></cac:LegalMonetaryTotal><cac:InvoiceLine><cbc:ID>1</cbc:ID><cbc:InvoicedQuantity unitCode="EA">1</cbc:InvoicedQuantity><cbc:LineExtensionAmount>100</cbc:LineExtensionAmount><cac:TaxTotal><cbc:TaxAmount>19</cbc:TaxAmount></cac:TaxTotal><cac:Item><cbc:Description>Item</cbc:Description></cac:Item><cac:Price><cbc:PriceAmount>100</cbc:PriceAmount></cac:Price></cac:InvoiceLine></Invoice>`

	directDoc, err := parser.ParseInvoiceXML([]byte(invoiceXML))
	if err != nil {
		t.Fatalf("expected direct invoice xml to parse, got %v", err)
	}
	if directDoc.InvoiceID != "FLEX-1" {
		t.Fatalf("expected direct invoice id FLEX-1, got %q", directDoc.InvoiceID)
	}

	wrappedDoc, err := parser.ParseInvoiceXML([]byte(wrapInAttachedDocument(invoiceXML)))
	if err != nil {
		t.Fatalf("expected attached invoice xml to parse, got %v", err)
	}
	if wrappedDoc.InvoiceID != "FLEX-1" {
		t.Fatalf("expected attached invoice id FLEX-1, got %q", wrappedDoc.InvoiceID)
	}
}

func TestDIANUBL21ParserParseInvoiceXMLErrorsOnMalformedXML(t *testing.T) {
	parser := NewDianUBL21Parser()

	_, err := parser.ParseInvoiceXML([]byte(`<Invoice><ID>1</ID>`))
	if !errors.Is(err, domain.ErrMalformedXML) {
		t.Fatalf("expected malformed xml error, got %v", err)
	}
}

func TestDIANUBL21ParserParseInvoiceXMLErrorsOnMissingParties(t *testing.T) {
	parser := NewDianUBL21Parser()

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

func mustReadFixture(t *testing.T, name string) []byte {
	t.Helper()

	path := filepath.Join("test_data", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %q: %v", path, err)
	}

	return data
}

func almostEqual(left, right float64) bool {
	const epsilon = 0.000001
	return math.Abs(left-right) <= epsilon
}

func wrapInAttachedDocument(invoiceXML string) string {
	return `<AttachedDocument xmlns="urn:oasis:names:specification:ubl:schema:xsd:AttachedDocument-2" xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2"><cac:Attachment><cac:ExternalReference><cbc:Description><![CDATA[` + invoiceXML + `]]></cbc:Description></cac:ExternalReference></cac:Attachment></AttachedDocument>`
}
