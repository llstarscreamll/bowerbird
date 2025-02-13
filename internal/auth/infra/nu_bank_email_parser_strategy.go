package infra

import (
	"fmt"
	"llstarscreamll/bowerbird/internal/auth/domain"
	"log"
	"regexp"
	"strconv"
	"strings"
)

type NuBankEmailParserStrategy struct{}

func (s NuBankEmailParserStrategy) Parse(emailMessage domain.MailMessage) []domain.Transaction {
	plainTextMessage := s.cleanUpHTML(emailMessage.Body)
	plainTextMessage = s.extractPlainText(plainTextMessage)

	fmt.Println(plainTextMessage)

	if !strings.Contains(plainTextMessage, "Aquí puedes ver los detalles del envío") {
		return []domain.Transaction{}
	}

	transactionType := "expense"
	transferAmount := s.getTransferAmount(plainTextMessage)
	transferTaxAmount := s.getTransferTaxAmount(plainTextMessage)

	if transactionType == "expense" {
		transferAmount = -transferAmount
	}

	return []domain.Transaction{
		{
			Origin:            "email",
			Amount:            transferAmount,
			Type:              transactionType,
			SystemDescription: s.getTransferDescription(plainTextMessage),
			ProcessedAt:       emailMessage.ReceivedAt,
		},
		{
			Origin:            "email",
			Amount:            transferTaxAmount,
			Type:              transactionType,
			SystemDescription: "Impuesto 4x1.000",
			ProcessedAt:       emailMessage.ReceivedAt,
		},
	}
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

	return amount
}

func (s NuBankEmailParserStrategy) getTransferDescription(plainTextMessage string) string {
	desc := ""
	reAmount := regexp.MustCompile(`Recibe\n([\w\s\.,]+) Nu`)
	matches := reAmount.FindStringSubmatch(plainTextMessage)

	if len(matches) > 1 {
		match := matches[1]
		desc = "Transferencia a " + match
	}

	return desc
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
