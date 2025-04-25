package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"lambda/types"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type GoogleServices struct {
	Client        *http.Client
	SheetsService *sheets.Service
	GmailService  *gmail.Service
}

type SheetProcessor struct {
	Service *sheets.Service
}

type EmailSender struct {
	Service  *gmail.Service
	UserInfo *types.UserInfo
}

func NewGoogleServices(token *oauth2.Token) (*GoogleServices, error) {
	// Create Google OAuth client
	client := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(token))

	// Initialize Sheets service
	sheetsService, err := sheets.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create sheets service: %v", err)
	}

	// Initialize Gmail service
	gmailService, err := gmail.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create gmail service: %v", err)
	}

	return &GoogleServices{
		Client:        client,
		SheetsService: sheetsService,
		GmailService:  gmailService,
	}, nil
}

func NewSheetProcessor(service *sheets.Service) *SheetProcessor {
	return &SheetProcessor{
		Service: service,
	}
}

func (sp *SheetProcessor) ProcessSheetData(spreadsheetID, readRange string) ([]*types.Document, error) {
	// Fetch data from spreadsheet
	resp, err := sp.Service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve data from sheet: %v", err)
	}

	if len(resp.Values) == 0 {
		return nil, fmt.Errorf("no data found")
	}

	// Process the sheet data into documents
	var docs []*types.Document
	for _, row := range resp.Values {
		if len(row) < 5 {
			fmt.Printf("Skipping row with insufficient columns: %v\n", row)
			continue
		}

		documentName := fmt.Sprintf("%v", row[0])
		fmt.Printf("Attempting to parse dates for document: %s\n", documentName)

		issueDate, err1 := time.Parse("2006-01-02", fmt.Sprintf("%v", row[1]))
		if err1 != nil {
			fmt.Printf("Error parsing issue date '%v': %v\n", row[1], err1)
		}

		expiryDate, err2 := time.Parse("2006-01-02", fmt.Sprintf("%v", row[2]))
		if err2 != nil {
			fmt.Printf("Error parsing expiry date '%v': %v\n", row[2], err2)
		}

		durationDaysStr := fmt.Sprintf("%v", row[3])
		durationDays, err3 := strconv.Atoi(durationDaysStr)
		if err3 != nil {
			fmt.Printf("Error parsing duration '%v': %v\n", row[3], err3)
			continue
		}
		durationValue := time.Duration(durationDays) * 24 * time.Hour

		status := fmt.Sprintf("%v", row[4])

		fmt.Printf("document name: %v\n", documentName)
		fmt.Printf("status: %v\n", status)
		fmt.Printf("issue date: %v\n", issueDate)
		fmt.Printf("expiry date: %v\n", expiryDate)

		if err1 != nil || err2 != nil {
			continue
		}

		// Create a doc and append to the slice
		doc := types.NewDoc(documentName, issueDate, expiryDate, durationValue, status)
		docs = append(docs, doc)
	}

	return docs, nil
}

func NewEmailSender(service *gmail.Service, userInfo *types.UserInfo) *EmailSender {
	return &EmailSender{
		Service:  service,
		UserInfo: userInfo,
	}
}

func (es *EmailSender) SendDocumentSummary(docs []*types.Document) error {
	// Build email content
	var bodyContent strings.Builder
	bodyContent.WriteString("Document Summary\n")
	bodyContent.WriteString("===============\n\n")

	for _, doc := range docs {
		bodyContent.WriteString(fmt.Sprintf("Document: %s\nIssue Date: %s\nExpiry Date: %s\nStatus: %s\n\n",
			doc.DocumentName,
			doc.IssueDate.Format("2006-01-02"),
			doc.ExpiryDate.Format("2006-01-02"),
			doc.Status))
	}

	// Create the email
	var emailBuilder strings.Builder
	emailBuilder.WriteString("From: me\r\n")
	emailBuilder.WriteString(fmt.Sprintf("To: %s\r\n", es.UserInfo.Email))
	emailBuilder.WriteString("Subject: Document Summary\r\n")
	emailBuilder.WriteString("MIME-Version: 1.0\r\n")
	emailBuilder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	emailBuilder.WriteString(bodyContent.String())

	// Encode the email
	emailRaw := base64.RawURLEncoding.EncodeToString([]byte(emailBuilder.String()))

	// Send email
	_, err := es.Service.Users.Messages.Send("me", &gmail.Message{
		Raw: emailRaw,
	}).Do()

	return err
}
