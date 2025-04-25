package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"lambda/api/auth"
	"lambda/database"
	"lambda/types"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// Document represents a record with issue and expiration dates
type Document struct {
	DocumentName string
	IssueDate    time.Time
	ExpiryDate   time.Time
	Duration     time.Duration // Note: Duration is unusual as time.Time, typically it would be time.Duration
	Status       string
}

func NewDoc(documentName string, issueDate time.Time, expiryDate time.Time, duration time.Duration, status string) *Document {
	return &Document{
		DocumentName: documentName,
		IssueDate:    issueDate,
		ExpiryDate:   expiryDate,
		Duration:     duration,
		Status:       status,
	}
}

// EmailConfig holds SMTP configuration
type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// ReminderService handles document reminders
type ReminderService struct {
	Docs     []Document
	EmailCfg EmailConfig
}

type compositeState struct {
	Nonce         string `json:"nonce"`
	SpreadsheetID string `json:"spreadsheet_id"`
}

type Api struct {
	Auth          *auth.AuthConfig
	databaseStore *database.DynamoDBStore
}

func NewApi(dbStore *database.DynamoDBStore) *Api {
	return &Api{
		databaseStore: dbStore,
	}
}

//MWlvQ3pkS1ZlakNMalZrM2g3Tko3RjM1VlowRFg3dWpOYjFjN2prRTlUR0U%3D

func (a *Api) LoginHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	corsHeaders := map[string]string{
		"Content-Type":                     "application/json",
		"Access-Control-Allow-Origin":      "http://localhost:3000",
		"Access-Control-Allow-Credentials": "true",
		"Access-Control-Allow-Methods":     "GET,POST,PUT,DELETE,OPTIONS",
		"Access-Control-Allow-Headers":     "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token",
	}
	id := strings.TrimSpace(request.QueryStringParameters["spreadsheet_id"])
	if id == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    corsHeaders,
			Body:       "spreadsheet_id is missing",
		}, nil
	}
	fmt.Printf("this is the id: %s\n", id)
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    corsHeaders,
			Body:       "failed to generate nonce",
		}, nil
	}

	nonce := base64.URLEncoding.EncodeToString(b)
	// This logs to CloudWatch
	fmt.Printf("this is the nonce login Handler: %s\n", nonce)
	statePayload := compositeState{
		Nonce:         nonce,
		SpreadsheetID: id,
	}
	raw, _ := json.Marshal(statePayload)
	fmt.Printf("this is the raw loginHandler: %s\n", raw)
	state := base64.URLEncoding.EncodeToString(raw)
	fmt.Printf("this is the state loginHandler: %s\n", state)
	authconfig := auth.NewAuthConfig()
	oauthConfig := authconfig.ToOAuth2Config()
	if oauthConfig == nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    corsHeaders,
			Body:       "Oauth configuration not initialized",
		}, nil
	}
	authURL := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	corsHeaders["Location"] = authURL
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusTemporaryRedirect,
		Headers:    corsHeaders,
		Body:       "",
	}, nil
}

// buildEmail constructs a base64‑URL encoded RFC‑2822 message.
func buildEmail(to, subject, body string) (string, error) {
	// Add proper headers including MIME-Version and Content-Type
	message := fmt.Sprintf(
		"From: me\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/plain; charset=UTF-8\r\n"+
			"\r\n%s",
		to, subject, body)

	// Use RawURLEncoding instead of URLEncoding to avoid padding
	encoded := base64.RawURLEncoding.EncodeToString([]byte(message))
	return encoded, nil
}

func (a *Api) CallbackHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	corsHeaders := map[string]string{
		"Content-Type":                     "application/json",
		"Access-Control-Allow-Origin":      "http://localhost:3000",
		"Access-Control-Allow-Credentials": "true",
		"Access-Control-Allow-Methods":     "GET,POST,PUT,DELETE,OPTIONS",
		"Access-Control-Allow-Headers":     "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token",
	}

	stateParam := strings.TrimSpace(request.QueryStringParameters["state"])
	fmt.Println("CallbackHandler invoked with state:", stateParam)

	// Decode base64 URL-encoded string
	raw, err := base64.URLEncoding.DecodeString(stateParam)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "failed to decode to string",
			Headers:    corsHeaders,
		}, nil
	}

	// Unmarshal JSON into compositeState struct
	var composite compositeState
	err = json.Unmarshal(raw, &composite)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "failed to parse JSON from decoded state",
			Headers:    corsHeaders,
		}, nil
	}

	code := request.QueryStringParameters["code"]

	if code == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "missing code",
			Headers:    corsHeaders,
		}, nil
	}

	oauthCfg := auth.NewAuthConfig().ToOAuth2Config()
	token, err := oauthCfg.Exchange(context.Background(), code)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "authentication failed",
			Headers:    corsHeaders,
		}, nil
	}

	if token.Expiry.Before(time.Now()) {
		fmt.Println("Token is expired")
		if token.RefreshToken != "" {
			newToken, err := oauthCfg.TokenSource(context.Background(), token).Token()
			if err != nil {
				fmt.Printf("error refreshing token: %V\n", err)
			} else {
				token = newToken
				fmt.Println("Token refreshed successfully")
			}
		}
	} else {
		fmt.Println("Token is valid (not expired)")
	}

	// 1. Extract the raw ID token string from the OAuth2 token extras.
	idToken, ok := token.Extra("id_token").(string)
	if !ok || idToken == "" {
		// If there's no id_token present, bail out early.
		fmt.Println("no id_token found in token extras")
	} else {
		// 2. A JWT has three parts separated by dots: header.payload.signature
		parts := strings.Split(idToken, ".")
		if len(parts) < 2 {
			// Malformed JWT: expect at least header and payload.
			fmt.Println("invalid JWT format; missing parts")
		} else {
			// 3. Base64-URL decode the payload (the middle part).
			payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
			if err != nil {
				// Decoding error: payload wasn't valid base64-URL.
				fmt.Printf("error decoding payload: %v\n", err)
			} else {
				// 4. Define the subset of claims you want to check.
				//    Add fields for issuer (iss), audience (aud), and expiration (exp).
				var claims struct {
					Iss string   `json:"iss"` // Issuer Identifier
					Aud []string `json:"aud"` // Audience(s) this token is intended for
					Exp int64    `json:"exp"` // Expiration time (unix seconds)
				}

				// 5. Unmarshal the JSON payload into your claims struct.
				err = json.Unmarshal(payloadJSON, &claims)
				if err != nil {
					// JSON parsing error.
					fmt.Printf("error parsing claims: %v\n", err)
				} else {
					// 6. Validate the issuer is exactly Google's OAuth 2.0 issuer.
					expectedIssuer := "https://accounts.google.com"
					if claims.Iss != expectedIssuer {
						fmt.Printf("invalid issuer: %q, expected: %q\n", claims.Iss, expectedIssuer)
					} else {
						fmt.Println("issuer validation passed")
					}

					// 7. Validate the audience includes your client ID.
					//    Replace YOUR_CLIENT_ID with the actual client ID string.
					yourClientID := "384543079988-lillmo592a40vt43dg3etuf3ghbg8s74.apps.googleusercontent.com"
					audValid := false
					for _, aud := range claims.Aud {
						if aud == yourClientID {
							audValid = true
							break
						}
					}
					if !audValid {
						fmt.Printf("invalid audience; none match %q\n", yourClientID)
					} else {
						fmt.Println("audience validation passed")
					}

					// 8. Validate the token has not expired.
					now := time.Now().Unix()
					if claims.Exp < now {
						fmt.Printf("token expired at %d; now is %d\n", claims.Exp, now)
					} else {
						fmt.Println("expiration validation passed")
					}
				}
			}
		}
	}
	client := &http.Client{Timeout: 10 * time.Second}
	tokenInfoURL := fmt.Sprintf(
		"https://oauth2.googleapis.com/tokeninfo?access_token=%s",
		token.AccessToken,
	)

	// 1. Perform the HTTP GET
	resp, err := client.Get(tokenInfoURL)
	if err != nil {
		// Log and return or handle the error immediately
		fmt.Printf("error checking token revocation: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "failed to verify token",
			Headers:    corsHeaders,
		}, nil
	}
	defer resp.Body.Close() // 2. Safe: resp is guaranteed non-nil here

	// 3. Now it's safe to inspect resp.StatusCode
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Token appears to be revoked or invalid")
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Printf("Response: %s\n", string(body))
	} else {
		fmt.Println("Token is not revoked")
	}

	err = a.databaseStore.StoreToken(token)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "error inserting into database",
			Headers:    corsHeaders,
		}, nil
	}

	googleClient := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(token))

	resp, err = googleClient.Get("https://www.googleapis.com/oauth2/v2/userinfo")

	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "failed to make request",
			Headers:    corsHeaders,
		}, nil
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "error reading user info",
			Headers:    corsHeaders,
		}, nil
	}

	var userInfo *types.UserInfo
	if err := json.Unmarshal([]byte(body), &userInfo); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "error unmarhaling userinfo json",
			Headers:    corsHeaders,
		}, nil
	}

	fmt.Printf("User ID: %v", userInfo.ID)
	fmt.Printf("Email: %v", userInfo.Email)
	fmt.Printf("Name: %v", userInfo.Name)

	fmt.Printf("response user info: %v\n", resp.Body)

	// Log the decoded values
	fmt.Printf("Decoded nonce: %s\n", composite.Nonce)
	fmt.Printf("Decoded spreadsheet_id: %s\n", composite.SpreadsheetID)

	sheetsService, err := sheets.NewService(context.Background(), option.WithHTTPClient(googleClient))
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "service unavailable",
			Headers:    corsHeaders,
		}, nil
	}

	if !a.isValidSpreadSheetID(composite.SpreadsheetID) {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "invalid spreedsheet id",
			Headers:    corsHeaders,
		}, nil
	}

	readRange := "Sheet1!A1:E10"

	value, err := a.getValuesSpreedSheet(sheetsService, composite.SpreadsheetID, readRange)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "error getting spreadsheet values",
			Headers:    corsHeaders,
		}, nil
	}

	fmt.Printf("Raw spreadsheet values: %v\n", value)
	fmt.Printf("Number of rows from spreadsheet: %d\n", len(value))

	var docs []Document
	for _, row := range value {
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
		doc := NewDoc(documentName, issueDate, expiryDate, durationValue, status)
		docs = append(docs, *doc)
	}

	fmt.Printf("this is the docs %v\n", docs)

	// Add debug logging
	fmt.Printf("Number of documents parsed: %d\n", len(docs))
	for i, doc := range docs {
		fmt.Printf("Document %d: DocumentName=%s, IssueDate=%v, ExpiryDate=%v\n",
			i, doc.DocumentName, doc.IssueDate, doc.ExpiryDate)
	}

	var bodyContent string
	bodyContent += "Document Summary\n"
	bodyContent += "===============\n\n"

	for _, doc := range docs {
		bodyContent += fmt.Sprintf("Document: %s\nIssue Date: %s\nExpiry Date: %s\nStatus: %s\n\n",
			doc.DocumentName,
			doc.IssueDate.Format("2006-01-02"),
			doc.ExpiryDate.Format("2006-01-02"),
			doc.Status)
	}

	// Add debug logging for the email content
	fmt.Printf("Email content to be sent:\n%s\n", bodyContent)

	// Initialize Gmail service
	gmailService, err := gmail.NewService(context.Background(), option.WithHTTPClient(googleClient))
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "failed to initialize Gmail service",
			Headers:    corsHeaders,
		}, nil
	}

	// Send the email using Gmail API
	subject := "Document Summary"

	// Create the email
	var emailBuilder strings.Builder
	emailBuilder.WriteString("From: me\r\n")
	emailBuilder.WriteString(fmt.Sprintf("To: %s\r\n", userInfo.Email))
	emailBuilder.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	emailBuilder.WriteString("MIME-Version: 1.0\r\n")
	emailBuilder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	emailBuilder.WriteString(bodyContent)

	// Encode the email using RawURLEncoding (without padding) as required by Gmail API
	emailRaw := base64.RawURLEncoding.EncodeToString([]byte(emailBuilder.String()))

	// Debug the email content
	fmt.Printf("Email headers and first 100 chars: %s\n", emailBuilder.String()[:100])

	// Send email with the properly formatted raw string
	_, err = gmailService.Users.Messages.Send("me", &gmail.Message{
		Raw: emailRaw,
	}).Do()

	if err != nil {
		log.Printf("Error sending email: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "something went wrong while sending email",
			Headers:    corsHeaders,
		}, nil
	}

	fmt.Println("Email sent successfully")

	callbackURL := fmt.Sprintf(
		"http://localhost:3000/summary",
	)

	fmt.Printf("value %v", value[0]...)

	// Return a redirect instead of JSON:
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusFound, // 302
		Headers: map[string]string{
			"Location":                         callbackURL,
			"Content-Type":                     "application/json",
			"Access-Control-Allow-Origin":      "http://localhost:3000",
			"Access-Control-Allow-Credentials": "true",
			"Access-Control-Allow-Methods":     "GET,POST,PUT,DELETE,OPTIONS",
			"Access-Control-Allow-Headers":     "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token",
		},
		Body: "",
	}, nil
}

func (a *Api) getValuesSpreedSheet(sheetsService *sheets.Service, spreadsheetID string, readRange string) ([][]interface{}, error) {
	resp, err := sheetsService.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve data from sheet: %v", err)
	}

	if len(resp.Values) == 0 {
		fmt.Println("No data found.")
		return nil, nil
	}

	// Print values for debugging
	for i, row := range resp.Values {
		fmt.Printf("Row %d: %v\n", i, row)
	}

	return resp.Values, nil
}

func (a *Api) isValidSpreadSheetID(id string) bool {
	if len(id) < 10 || len(id) > 100 {
		return false
	}

	// Simple regex to validate spreadsheet ID format
	validID := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(id)
	return validID
}
