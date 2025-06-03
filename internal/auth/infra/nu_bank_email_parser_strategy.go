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
var savingsAccountStatement = "extracto de tu cuenta"

func (s NuBankEmailParserStrategy) Parse(mail domain.MailMessage, passwords []string) []domain.Transaction {

	transactionsFromEmailBody := s.parseFromMailBody(mail)
	transactionsFromAttachments := s.parseFromAttachments(mail.Attachments, passwords)

	return slices.DeleteFunc(append(transactionsFromEmailBody, transactionsFromAttachments...), func(t domain.Transaction) bool {
		return t.Amount == 0
	})
}

func (s NuBankEmailParserStrategy) parseFromMailBody(mail domain.MailMessage) []domain.Transaction {
	messageSubject := strings.ToLower(mail.Subject)

	subjects := []string{payment, transferToNuBank, transferToExternalBank, creditCardStatement, savingsAccountStatement}
	if !slices.ContainsFunc(subjects, func(s string) bool {
		return strings.Contains(strings.ToLower(messageSubject), strings.ToLower(s))
	}) {
		return []domain.Transaction{}
	}

	plainTextMail := s.cleanUpMailHTML(mail.Body)
	plainTextMail = s.extractPlainTextFromMailHTML(plainTextMail)

	description := ""
	transactionType := "expense"
	transferAmount := float32(0)
	transferTaxAmount := float32(0)

	transferAmount = s.getTransferAmountFromPlainTextMail(plainTextMail)
	transferTaxAmount = s.getTransferTaxAmountFromPlainTextMail(plainTextMail)

	if strings.Contains(messageSubject, transferToNuBank) {
		description = s.getNuTransferDescriptionFromPlainTextMail(plainTextMail)
	}

	if strings.Contains(messageSubject, transferToExternalBank) {
		description = s.getExternalBankTransferDescriptionFromPlainTextMail(plainTextMail)
	}

	if strings.Contains(messageSubject, payment) {
		description = s.getPaymentDescriptionFromPlainTextMail(plainTextMail)
		transferAmount = s.getPaymentAmountFromPlainTextMail(plainTextMail)
		transferTaxAmount = s.getPaymentTaxAmountFromPlainTextMail(plainTextMail)
	}

	return []domain.Transaction{
		{
			Origin:            "nu/savings",
			Amount:            transferAmount,
			Type:              transactionType,
			SystemDescription: description,
			ProcessedAt:       mail.ReceivedAt,
		},
		{
			Origin:            "nu/savings",
			Amount:            transferTaxAmount,
			Type:              transactionType,
			SystemDescription: "4x1.000",
			ProcessedAt:       mail.ReceivedAt,
		},
	}
}

func (s NuBankEmailParserStrategy) cleanUpMailHTML(html string) string {
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

func (s NuBankEmailParserStrategy) extractPlainTextFromMailHTML(html string) string {
	re := regexp.MustCompile(`<.*?>`)
	html = re.ReplaceAllString(html, "")

	re = regexp.MustCompile("\n{2,}")
	html = re.ReplaceAllString(html, "\n\n")

	return html
}

func (s NuBankEmailParserStrategy) getTransferAmountFromPlainTextMail(plainTextMessage string) float32 {
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

func (s NuBankEmailParserStrategy) getPaymentAmountFromPlainTextMail(plainTextMessage string) float32 {
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

func (s NuBankEmailParserStrategy) getTransferTaxAmountFromPlainTextMail(plainTextMessage string) float32 {
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

func (s NuBankEmailParserStrategy) getPaymentTaxAmountFromPlainTextMail(plainTextMessage string) float32 {
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

func (s NuBankEmailParserStrategy) getNuTransferDescriptionFromPlainTextMail(plainTextMessage string) string {
	desc := ""
	reReceiver := regexp.MustCompile(`Recibe\n(.+) Nu`)
	matches := reReceiver.FindStringSubmatch(plainTextMessage)

	if len(matches) > 1 {
		match := matches[1]
		desc = match
	}

	return desc
}

func (s NuBankEmailParserStrategy) getExternalBankTransferDescriptionFromPlainTextMail(plainTextMessage string) string {
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

func (s NuBankEmailParserStrategy) getPaymentDescriptionFromPlainTextMail(plainTextMessage string) string {
	desc := ""
	reReceiver := regexp.MustCompile(`Pagaste en:\n(.+)\n`)
	receiverMatches := reReceiver.FindStringSubmatch(plainTextMessage)

	if len(receiverMatches) > 1 {
		desc = receiverMatches[1]
	}

	return desc
}

func (s NuBankEmailParserStrategy) parseFromAttachments(attachments []domain.MailAttachment, passwords []string) []domain.Transaction {
	transactions := make([]domain.Transaction, 0)

	for _, attachment := range attachments {
		if !strings.HasPrefix(attachment.ContentType, "application/pdf") {
			continue
		}

		tsv := s.parsePDFToTsv(attachment, passwords)
		if tsv == "" {
			fmt.Println("Can't parse PDF to TSV")
			continue
		}

		if s.tsvIsBankStatement(tsv) {
			transactions = append(transactions, s.parseFromBankStatementTsv(tsv)...)
		}
	}

	return transactions
}

func (s NuBankEmailParserStrategy) tsvIsBankStatement(tsv string) bool {
	m1 := regexp.MustCompile(`(?m)^.+\tHola\,(?:\n.+){4}\tLlegó\n.+\ttu\n.+\textracto`).FindStringSubmatch(tsv)
	m2 := regexp.MustCompile(`(?m)^.+\tNu\n.+\tPlaca\n`).FindStringSubmatch(tsv)
	return len(m1) > 0 && len(m2) > 0
}

func (s NuBankEmailParserStrategy) parsePDFToTsv(attachment domain.MailAttachment, passwords []string) string {
	tmpFile, err := os.CreateTemp("/tmp", "*.pdf")

	if err != nil {
		log.Printf("Error creating temp file: %v", err)
		return ""
	}

	defer os.Remove(tmpFile.Name())

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

	defer os.Remove(tsvFile.Name())
	tsvFile.Close()

	for _, password := range passwords {
		stdErr := &bytes.Buffer{}
		stdOut := &bytes.Buffer{}
		cmd := exec.Command("pdftotext", "-tsv", tmpFile.Name(), tsvFile.Name(), "-upw", password)
		cmd.Stdout = stdOut
		cmd.Stderr = stdErr

		if err := cmd.Run(); err != nil {
			log.Printf("Error parsing PDF: %v, %s", err, stdErr.String())
			continue
		}
	}

	textBytes, err := os.ReadFile(tsvFile.Name())
	if err != nil {
		log.Printf("Error reading text file: %v", err)
		return ""
	}

	return string(textBytes)
}

func (s NuBankEmailParserStrategy) parseFromBankStatementTsv(tsv string) []domain.Transaction {
	transactions := make([]domain.Transaction, 0)
	totalAccountReturns := float64(0)

	statementYearString := regexp.MustCompile(`(?m)^.+\t21\.38.+\t(\d{4})\n`).FindStringSubmatch(tsv)[1]
	statementYear, err := strconv.Atoi(statementYearString)
	if err != nil {
		log.Printf("Error parsing statement year: %v", err)
	}

	statementMonthString := regexp.MustCompile(`(?m)^.+\t(\w{3})\n.+\t21\.38.+\t\d{4}\n`).FindStringSubmatch(tsv)[1]
	statementMonth := s.parseMonth(statementMonthString)

	statementLastDayString := regexp.MustCompile(`(?m)-\n^.+\t2\t.+\t(\d+)\n`).FindStringSubmatch(tsv)[1]
	statementLastDay, err := strconv.Atoi(statementLastDayString)
	if err != nil {
		log.Printf("Error parsing statement last day: %v", err)
	}

	pages := regexp.MustCompile(`(?m)^.+\t###PAGE###$`).Split(tsv, -1)
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

		totalAccountReturnsMatches := regexp.MustCompile(`(?m)^.+\tRendimiento\n.+\ttotal\n(?:.+\n){3}.+(\+\$[\d\.,]+)`).FindStringSubmatch(page)
		if len(totalAccountReturnsMatches) > 1 {
			totalAccountReturnsString := totalAccountReturnsMatches[1]
			totalAccountReturnsString = strings.ReplaceAll(totalAccountReturnsString, "$", "")
			totalAccountReturnsString = strings.ReplaceAll(totalAccountReturnsString, ".", "")
			totalAccountReturnsString = strings.ReplaceAll(totalAccountReturnsString, ",", ".")
			totalAccountReturns, err = strconv.ParseFloat(totalAccountReturnsString, 32)
			if err != nil {
				log.Printf("Error parsing total account returns: %v", err)
			}
		}

		// clean account profit info
		page = regexp.MustCompile(`(?m)^.+\tRendimiento\n.+\ttotal(?:\n.+){4}`).ReplaceAllString(page, "")

		tsvTransactions := regexp.MustCompile(`(?m)^.+\t68\.000000\t.+\t###FLOW###$`).Split(page, -1)
		for _, tsvTransaction := range tsvTransactions {
			tsvTransaction = regexp.MustCompile(`(?m)^.+\t###FLOW###\n`).ReplaceAllString(tsvTransaction, "")
			tsvTransaction = regexp.MustCompile(`(?m)^.+\t###LINE###\n`).ReplaceAllString(tsvTransaction, "")
			tsvTransaction = regexp.MustCompile(`(?m)\n\n`).ReplaceAllString(tsvTransaction, "\n")
			tsvTransaction = strings.TrimSpace(tsvTransaction)

			if tsvTransaction == "" {
				continue
			}

			parsedTransactions := s.parseTsvTransaction(tsvTransaction, statementYear)

			// this is a uniqueness mechanism for transactions that happens in
			// the same day, to same receiver/sender and same amount, when
			// the same transaction is repeated in the same day, we need to
			// increment the uniqueness count to avoid duplicate transactions,
			// example:
			// 20250528/Nu/savings/JohnDoe/-15/0 -> last digit 0 is the uniqueness count
			// 20250528/Nu/savings/JohnDoe/-15/1 -> last digit 1 is the uniqueness count
			// 20250528/Nu/savings/JohnDoe/-15/2 -> last digit 2 is the uniqueness count
			for i, t1 := range parsedTransactions {
				for _, t2 := range transactions {
					t1UniqueString := fmt.Sprintf("%s/%s/%f", t1.ProcessedAt.Format("20060102"), t1.SystemDescription, t1.Amount)
					t2UniqueString := fmt.Sprintf("%s/%s/%f", t2.ProcessedAt.Format("20060102"), t2.SystemDescription, t2.Amount)
					if t1UniqueString == t2UniqueString {
						parsedTransactions[i].UniquenessCount = t2.UniquenessCount + 1
					}
				}
			}

			transactions = append(transactions, parsedTransactions...)
		}
	}

	transactions = append(transactions, domain.Transaction{
		SystemDescription: "Rendimientos NuBank",
		Origin:            "nu/savings",
		Type:              "income",
		Amount:            float32(totalAccountReturns),
		ProcessedAt:       time.Date(statementYear, time.Month(statementMonth), statementLastDay, 23, 59, 59, 0, time.UTC),
		CreatedAt:         time.Now(),
	})

	return transactions
}

func (s NuBankEmailParserStrategy) parseTsvTransaction(tsvTransaction string, statementYear int) []domain.Transaction {
	cleanTsv := regexp.MustCompile(`(?m)^.+\t(.+)\n`).ReplaceAllString(tsvTransaction, "$1\n")
	cleanTsv = regexp.MustCompile(`(?m)^.+\t(.+)`).ReplaceAllString(cleanTsv, "$1")
	lines := strings.Split(cleanTsv, "\n")

	date, err := time.Parse("02 1 2006", fmt.Sprintf("%s %d %d", lines[0], s.parseMonth(lines[1]), statementYear))
	if err != nil {
		log.Printf("Error parsing date: %v\ntBlock:\n%s", err, tsvTransaction)
	}

	has4x1000 := strings.Contains(cleanTsv, "4x1000")
	description := strings.Join(lines[2:len(lines)-1], " ")
	taxAmountString := "0"
	transactionType := "expense"
	amountsMatches := regexp.MustCompile(`[\+|\-]\$[\d|\.|,]+`).FindAllString(cleanTsv, -1)
	amountString := amountsMatches[0]

	if has4x1000 {
		taxAmountString = amountsMatches[1]
		description = strings.Join(lines[2:len(lines)-5], " ")
	}

	if strings.Contains(amountString, "+$") {
		transactionType = "income"
	}

	amountString = strings.ReplaceAll(amountString, "$", "")
	amountString = strings.ReplaceAll(amountString, ".", "")
	amountString = strings.ReplaceAll(amountString, ",", ".")
	taxAmountString = strings.ReplaceAll(taxAmountString, "$", "")
	taxAmountString = strings.ReplaceAll(taxAmountString, ".", "")
	taxAmountString = strings.ReplaceAll(taxAmountString, ",", ".")
	description = strings.ReplaceAll(description, "Enviaste a ", "")
	description = strings.ReplaceAll(description, "Recibiste de ", "")
	description = strings.ReplaceAll(description, "S A", "SA")

	amount, err := strconv.ParseFloat(amountString, 32)
	if err != nil {
		log.Printf("Error parsing amount: %v", err)
	}

	taxAmount, err := strconv.ParseFloat(taxAmountString, 32)
	if err != nil {
		log.Printf("Error parsing tax amount: %v\ntBlock:\n%s", err, tsvTransaction)
	}

	return []domain.Transaction{
		{
			SystemDescription: description,
			Type:              transactionType,
			Amount:            float32(amount),
			Origin:            "nu/savings",
			ProcessedAt:       date,
		},
		{
			SystemDescription: "4x1.000",
			Type:              "expense",
			Amount:            float32(taxAmount),
			Origin:            "nu/savings",
			ProcessedAt:       date,
		},
	}
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
