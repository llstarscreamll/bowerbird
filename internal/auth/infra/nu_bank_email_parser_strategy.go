package infra

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"llstarscreamll/bowerbird/internal/auth/domain"
	"log"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

type NuBankEmailParserStrategy struct{}

var payment = " con cuenta nu"
var transferToExternalBank = "tu dinero ya va en camino"
var transferToNuBank = "el dinero que enviaste ya está del otro lado"
var creditCardStatement = "extracto de tu tarjeta"
var savingAccountStatement = "extracto de tu cuenta"

func (s NuBankEmailParserStrategy) Parse(emailMessage domain.MailMessage) []domain.Transaction {
	messageSubject := strings.ToLower(emailMessage.Subject)
	plainTextMessage := s.cleanUpHTML(emailMessage.Body)
	plainTextMessage = s.extractPlainText(plainTextMessage)

	subjects := []string{payment, transferToNuBank, transferToExternalBank, creditCardStatement, savingAccountStatement}
	if !slices.ContainsFunc(subjects, func(s string) bool {
		return strings.Contains(strings.ToLower(messageSubject), strings.ToLower(s))
	}) {
		return []domain.Transaction{}
	}

	description := ""
	transactionType := "expense"
	transferAmount := float32(0)
	transferTaxAmount := float32(0)

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

	transactionsFromAttachments := s.getFromAttachments(emailMessage.Attachments)
	transactionsFromEmailBody := []domain.Transaction{
		{
			Origin:            "nu-bank-email",
			Reference:         emailMessage.ExternalID,
			Amount:            transferAmount,
			Type:              transactionType,
			SystemDescription: description,
			ProcessedAt:       emailMessage.ReceivedAt,
		},
		{
			Origin:            "nu-bank-email",
			Reference:         emailMessage.ExternalID + "_tax",
			Amount:            transferTaxAmount,
			Type:              transactionType,
			SystemDescription: "4x1.000",
			ProcessedAt:       emailMessage.ReceivedAt,
		},
	}

	return slices.DeleteFunc(append(transactionsFromEmailBody, transactionsFromAttachments...), func(t domain.Transaction) bool {
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

func (s NuBankEmailParserStrategy) getFromAttachments(attachments []domain.MailAttachment) []domain.Transaction {
	transactions := []domain.Transaction{}

	for _, attachment := range attachments {
		if !strings.HasPrefix(attachment.ContentType, "application/pdf") {
			continue
		}

		tsvText := s.parsePDFToTsv(attachment)
		if tsvText == "" {
			continue
		}

		transactions = append(transactions, s.parseTsvToTransactions(tsvText)...)
	}

	return transactions
}

func (s NuBankEmailParserStrategy) parsePDFToTsv(attachment domain.MailAttachment) string {
	tmpFile, err := os.CreateTemp("/tmp", "*.pdf")

	if err != nil {
		log.Printf("Error creating temp file: %v", err)
		return ""
	}

	defer os.Remove(tmpFile.Name())

	fmt.Printf("PDF: %s\n", tmpFile.Name())

	pdfBytes, err := base64.StdEncoding.DecodeString(attachment.Content)
	if err != nil {
		log.Printf("Error decoding PDF content: %v", err)
		return ""
	}

	if _, err := tmpFile.Write(pdfBytes); err != nil {
		log.Printf("Error writing PDF content: %v", err)
		return ""
	}

	tmpFile.Close()

	tsvFile, err := os.CreateTemp("", "*.tsv")
	if err != nil {
		log.Printf("Error creating text temp file: %v", err)
		return ""
	}

	// defer os.Remove(tsvFile.Name())
	tsvFile.Close()

	fmt.Printf("tsvFile: %s\n", tsvFile.Name())

	stdErr := &bytes.Buffer{}
	stdOut := &bytes.Buffer{}
	cmd := exec.Command("pdftotext", "-tsv", tmpFile.Name(), tsvFile.Name(), "-upw", "1057581292")
	cmd.Stdout = stdOut
	cmd.Stderr = stdErr

	if err := cmd.Run(); err != nil {
		log.Printf("Error parsing PDF: %v, %s", err, stdErr.String())
		return ""
	}

	textBytes, err := os.ReadFile(tsvFile.Name())
	if err != nil {
		log.Printf("Error reading text file: %v", err)
		return ""
	}

	return string(textBytes)
}

func (s NuBankEmailParserStrategy) parseTsvToTransactions(tsvText string) []domain.Transaction {
	transactions := []domain.Transaction{}

	statementYearString := regexp.MustCompile(`(?m)^.+\t288\.06.+\t(\d{4})\n`).FindStringSubmatch(tsvText)[1]
	statementYear, err := strconv.Atoi(statementYearString)
	if err != nil {
		log.Printf("Error parsing statement year: %v", err)
	}

	pages := regexp.MustCompile(`(?m)^.+\t###PAGE###$`).Split(tsvText, -1)
	for i, page := range pages[2:] {
		// remove NuBank interest info
		page = regexp.MustCompile(`(?m)^.+FLOW###\n(?:.+\n){11}.+\tdiario.\n`).ReplaceAllString(page, "")
		page = regexp.MustCompile(`(?m)^.+FLOW###\n.+\n.+\n.+rendimientos\n.+\n.+\n.+\n.+diario\n`).ReplaceAllString(page, "")
		// remove NuBank NIT info
		page = regexp.MustCompile(`(?m)^.+\t48\.000000.+FLOW###\n(?:.+\n){9}.+\t901\.658\.107\-2\n`).ReplaceAllString(page, "")
		// remove NuBank info
		page = regexp.MustCompile(`(?m)^.+\t###FLOW###\n`+
			`.+\n.+Nu\.\n`+
			`.+Colombia\n`+
			`.+Compañía\n`+
			`.+de\n`+
			`.+Financiamiento\n`+
			`.+S\.A\.\n`).ReplaceAllString(page, "")
		// remove footer
		page = regexp.MustCompile(`(?m)^.+FLOW###\n.+\n.+¿Tienes\n.+preguntas\n(?:.+\n){94}`).ReplaceAllString(page, "")
		// get movements section
		page = regexp.MustCompile(`(?m)^.+\tMovimientos$`).Split(page, -1)[1]

		// remove page counts
		page = regexp.MustCompile(fmt.Sprintf(`(?m)^.+\t809\.598000.+FLOW###\n.+\n.+\t%d\n.+\t\/\n.+\t\d\n`, i+2)).ReplaceAllString(page, "")

		lines := slices.DeleteFunc(strings.Split(page, "\n"), func(line string) bool {
			return strings.TrimSpace(line) == ""
		})

		slices.SortFunc(lines, func(a, b string) int {
			columnsA := strings.Split(a, "\t")
			columnsB := strings.Split(b, "\t")

			if len(columnsA) != 12 || len(columnsB) != 12 {
				fmt.Printf("Lengths does not match:\n%s\n%s\n", a, b)
				return 0
			}

			topA := columnsA[7]
			leftA := columnsA[6]
			topB := columnsB[7]
			leftB := columnsB[6]
			decimalsRegex := regexp.MustCompile(`\.(\d+)`)

			if len(decimalsRegex.FindStringSubmatch(topA)[1]) < 6 {
				topA = topA + "0001"
			}

			if len(decimalsRegex.FindStringSubmatch(topB)[1]) < 6 {
				topB = topB + "0001"
			}

			if len(decimalsRegex.FindStringSubmatch(leftA)[1]) < 6 {
				leftA = leftA + "0000"
			}

			if len(decimalsRegex.FindStringSubmatch(leftB)[1]) < 6 {
				leftB = leftB + "0000"
			}

			compareA := fmt.Sprintf("%06s-%010s", topA, leftA)
			compareB := fmt.Sprintf("%06s-%010s", topB, leftB)

			return strings.Compare(compareA, compareB)
		})

		page = strings.Join(lines, "\n")

		// clean account profit info
		page = regexp.MustCompile(`(?m)^.+\tRendimiento\n.+\ttotal\n\n(?:.+\n){4}`).ReplaceAllString(page, "")

		transactionMainBlocks := regexp.MustCompile(`(?m)^.+\t68\.000000\t.+\t###FLOW###$`).Split(page, -1)
		for _, tBlock := range transactionMainBlocks {
			tBlock = regexp.MustCompile(`(?m)^.+\t###FLOW###\n`).ReplaceAllString(tBlock, "")
			tBlock = regexp.MustCompile(`(?m)^.+\t###LINE###\n`).ReplaceAllString(tBlock, "")
			tBlock = regexp.MustCompile(`(?m)\n\n`).ReplaceAllString(tBlock, "\n")
			tBlock = strings.TrimSpace(tBlock)

			if tBlock == "" {
				continue
			}

			transactions = append(transactions, s.parseTransactionBlock(tBlock, statementYear)...)
		}

	}

	return transactions
}

func (s NuBankEmailParserStrategy) parseTransactionBlock(rawTBlock string, statementYear int) []domain.Transaction {
	cleanTBlock := regexp.MustCompile(`(?m)^.+\t(.+)\n`).ReplaceAllString(rawTBlock, "$1\n")
	cleanTBlock = regexp.MustCompile(`(?m)^.+\t(.+)`).ReplaceAllString(cleanTBlock, "$1")
	lines := strings.Split(cleanTBlock, "\n")

	date, err := time.Parse("02 1 2006", fmt.Sprintf("%s %d %d", lines[0], s.parseMonth(lines[1]), statementYear))
	if err != nil {
		log.Printf("Error parsing date: %v\ntBlock:\n%s", err, rawTBlock)
	}

	has4x1000 := strings.Contains(cleanTBlock, "4x1000")
	amountsMatches := regexp.MustCompile(`[\+|\-]\$[\d|\.|,]+`).FindAllString(cleanTBlock, -1)
	amountString := amountsMatches[0]
	taxAmountString := "0"
	description := strings.Join(lines[2:len(lines)-1], " ")

	if has4x1000 {
		taxAmountString = amountsMatches[1]
		description = strings.Join(lines[2:len(lines)-5], " ")
	}

	amountString = strings.ReplaceAll(amountString, "$", "")
	amountString = strings.ReplaceAll(amountString, ".", "")
	amountString = strings.ReplaceAll(amountString, ",", ".")
	taxAmountString = strings.ReplaceAll(taxAmountString, "$", "")
	taxAmountString = strings.ReplaceAll(taxAmountString, ".", "")
	taxAmountString = strings.ReplaceAll(taxAmountString, ",", ".")
	description = strings.ReplaceAll(description, "Enviaste", "Envío")
	description = strings.ReplaceAll(description, "S A", "SA")

	amount, err := strconv.ParseFloat(amountString, 32)
	if err != nil {
		log.Printf("Error parsing amount: %v", err)
	}

	taxAmount, err := strconv.ParseFloat(taxAmountString, 32)
	if err != nil {
		log.Printf("Error parsing tax amount: %v\ntBlock:\n%s", err, rawTBlock)
	}

	return slices.DeleteFunc([]domain.Transaction{
		{
			Amount:            float32(amount),
			SystemDescription: description,
			ProcessedAt:       date,
		},
		{
			Amount:            float32(taxAmount),
			SystemDescription: "4x1.000",
			ProcessedAt:       date,
		},
	}, func(t domain.Transaction) bool {
		return t.Amount == 0
	})
}

func (s NuBankEmailParserStrategy) parseMonth(monthName string) int {
	monthName = strings.ToLower(monthName)

	months := map[string]int{
		"ene": 1,
		"feb": 2,
		"mar": 3,
		"abr": 4,
		"may": 5,
		"jun": 6,
		"jul": 7,
		"ago": 8,
		"sep": 9,
		"oct": 10,
		"nov": 11,
		"dic": 12,
	}

	return months[monthName]
}
