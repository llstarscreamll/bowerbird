package infra

import (
	"llstarscreamll/bowerbird/internal/auth/domain"
	"log"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type NuBankEmailParserStrategy struct{}

var payment = " con cuenta nu"
var transferToExternalBank = "tu dinero ya va en camino"
var transferToNuBank = "el dinero que enviaste ya está del otro lado"

func (s NuBankEmailParserStrategy) Parse(emailMessage domain.MailMessage) []domain.Transaction {
	messageSubject := strings.ToLower(emailMessage.Subject)
	plainTextMessage := s.cleanUpHTML(emailMessage.Body)
	plainTextMessage = s.extractPlainText(plainTextMessage)

	subjects := []string{payment, transferToNuBank, transferToExternalBank}

	if !slices.ContainsFunc(subjects, func(s string) bool {
		return strings.Contains(messageSubject, s)
	}) {
		return []domain.Transaction{}
	}

	transactionType := "expense"
	transferAmount := float32(0)
	transferTaxAmount := float32(0)
	description := ""

	transferAmount = s.getTransferAmount(plainTextMessage)
	transferTaxAmount = s.getTransferTaxAmount(plainTextMessage)

	if strings.Contains(messageSubject, transferToNuBank) {
		description = s.getNuTransferDescription(plainTextMessage)
	}

	if strings.Contains(messageSubject, transferToExternalBank) {
		description = s.getExternalBankTransferDescription(plainTextMessage)
	}

	if strings.Contains(messageSubject, payment) {
		description = s.getPaymentDescription(plainTextMessage)
		transferAmount = s.getPaymentAmount(plainTextMessage)
		transferTaxAmount = s.getPaymentTaxAmount(plainTextMessage)
	}

	return slices.DeleteFunc([]domain.Transaction{
		{
			Origin:            "email",
			Reference:         emailMessage.ExternalID,
			Amount:            transferAmount,
			Type:              transactionType,
			SystemDescription: description,
			ProcessedAt:       emailMessage.ReceivedAt,
		},
		{
			Origin:            "email",
			Reference:         emailMessage.ExternalID + "_tax",
			Amount:            transferTaxAmount,
			Type:              transactionType,
			SystemDescription: "4x1.000",
			ProcessedAt:       emailMessage.ReceivedAt,
		},
	}, func(t domain.Transaction) bool {
		return t.Amount == 0
	})
}

func (s NuBankEmailParserStrategy) cleanUpHTML(html string) string {
	reOfficeDocSettings := regexp.MustCompile(`<o:OfficeDocumentSettings[\s\S]*?</o:OfficeDocumentSettings>`)
	html = reOfficeDocSettings.ReplaceAllString(html, "")

	reCssStyleTag := regexp.MustCompile(`<style[\s\S]*?</style>`)
	html = reCssStyleTag.ReplaceAllString(html, "")

	html = strings.ReplaceAll(html, "<br>", "\n")
	html = strings.ReplaceAll(html, "<br >", "\n")
	html = strings.ReplaceAll(html, "<br \\>", "\n")
	html = strings.ReplaceAll(html, "</tr>", "</tr>\n")
	html = strings.ReplaceAll(html, "</h1>", "</h1>\n")
	html = strings.ReplaceAll(html, "</h2>", "</h2>\n")
	html = strings.ReplaceAll(html, "</h3>", "</h3>\n")
	html = strings.ReplaceAll(html, "&nbsp;", " ")

	return html
}

func (s NuBankEmailParserStrategy) extractPlainText(html string) string {
	re := regexp.MustCompile(`<.*?>`)
	html = re.ReplaceAllString(html, "")

	re = regexp.MustCompile("\n{2,}")
	html = re.ReplaceAllString(html, "\n\n")

	return html
}

func (s NuBankEmailParserStrategy) getTransferAmount(plainTextMessage string) float32 {
	amount := float32(0)
	reAmount := regexp.MustCompile(`Monto\n\$([\d\.,]+)\n`)
	matches := reAmount.FindStringSubmatch(plainTextMessage)

	if len(matches) > 1 {
		amountStr := matches[1]
		amountStr = strings.ReplaceAll(amountStr, ".", "")
		amountStr = strings.ReplaceAll(amountStr, ",", ".")

		parsedAmount, err := strconv.ParseFloat(amountStr, 32)
		if err != nil {
			log.Println("Error parsing amount ("+amountStr+") from email body:", err)
		}

		if err == nil {
			amount = float32(parsedAmount)
		}
	}

	return -amount
}

func (s NuBankEmailParserStrategy) getPaymentAmount(plainTextMessage string) float32 {
	amount := float32(0)
	reAmount := regexp.MustCompile(`La cantidad de:\n\$([\d\.,]+)\n`)
	matches := reAmount.FindStringSubmatch(plainTextMessage)

	if len(matches) > 1 {
		amountStr := matches[1]
		amountStr = strings.ReplaceAll(amountStr, ".", "")
		amountStr = strings.ReplaceAll(amountStr, ",", ".")

		parsedAmount, err := strconv.ParseFloat(amountStr, 32)
		if err != nil {
			log.Println("Error parsing amount ("+amountStr+") from email body:", err)
		}

		if err == nil {
			amount = float32(parsedAmount)
		}
	}

	return -amount
}

func (s NuBankEmailParserStrategy) getTransferTaxAmount(plainTextMessage string) float32 {
	amount := float32(0)
	reAmount := regexp.MustCompile(`Impuesto del 4x1\.000\n\$([\d\.,]+)\n`)
	matches := reAmount.FindStringSubmatch(plainTextMessage)

	if len(matches) > 1 {
		amountStr := matches[1]
		amountStr = strings.ReplaceAll(amountStr, ".", "")
		amountStr = strings.ReplaceAll(amountStr, ",", ".")

		parsedAmount, err := strconv.ParseFloat(amountStr, 32)
		if err != nil {
			log.Println("Error parsing amount ("+amountStr+") from email body:", err)
		}

		if err == nil {
			amount = float32(parsedAmount)
		}
	}

	return -amount
}

func (s NuBankEmailParserStrategy) getPaymentTaxAmount(plainTextMessage string) float32 {
	amount := float32(0)
	reAmount := regexp.MustCompile(`Más el impuesto del 4xmil de:\n\$([\d\.,]+)\n`)
	matches := reAmount.FindStringSubmatch(plainTextMessage)

	if len(matches) > 1 {
		amountStr := matches[1]
		amountStr = strings.ReplaceAll(amountStr, ".", "")
		amountStr = strings.ReplaceAll(amountStr, ",", ".")

		parsedAmount, err := strconv.ParseFloat(amountStr, 32)
		if err != nil {
			log.Println("Error parsing amount ("+amountStr+") from email body:", err)
		}

		if err == nil {
			amount = float32(parsedAmount)
		}
	}

	return -amount
}

func (s NuBankEmailParserStrategy) getNuTransferDescription(plainTextMessage string) string {
	desc := ""
	reReceiver := regexp.MustCompile(`Recibe\n(.+) Nu`)
	matches := reReceiver.FindStringSubmatch(plainTextMessage)

	if len(matches) > 1 {
		match := matches[1]
		desc = "Envío a " + match
	}

	return desc
}

func (s NuBankEmailParserStrategy) getExternalBankTransferDescription(plainTextMessage string) string {
	desc := ""
	reReceiver := regexp.MustCompile(`Recibe\n(.+)\n`)
	reBank := regexp.MustCompile(`Banco\n(.+)\n`)
	receiverMatches := reReceiver.FindStringSubmatch(plainTextMessage)
	bankMatches := reBank.FindStringSubmatch(plainTextMessage)

	if len(receiverMatches) > 1 {
		desc = receiverMatches[1]
	}

	if len(bankMatches) > 1 {
		match := bankMatches[1]
		desc += " (" + match + ")"
	}

	return desc
}

func (s NuBankEmailParserStrategy) getPaymentDescription(plainTextMessage string) string {
	desc := ""
	reReceiver := regexp.MustCompile(`Pagaste en:\n(.+)\n`)
	receiverMatches := reReceiver.FindStringSubmatch(plainTextMessage)

	if len(receiverMatches) > 1 {
		desc = receiverMatches[1]
	}

	return desc
}
