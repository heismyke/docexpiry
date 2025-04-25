package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"lambda/api/auth"
	"lambda/database"
	"lambda/types"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"golang.org/x/oauth2"
)

type compositeState struct {
	Nonce         string `json:"nonce"`
	SpreadsheetID string `json:"spreadsheet_id"`
}

type CallBackHandler struct {
	Auth          *auth.AuthConfig
	databaseStore *database.DynamoDBStore
}

func NewCallbackHandler(dbStore *database.DynamoDBStore) *CallBackHandler {
	return &CallBackHandler{
		databaseStore: dbStore,
	}
}

func (cb *CallBackHandler) OauthCallback(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	corsHeaders := map[string]string{
		"Content-Type":                     "application/json",
		"Access-Control-Allow-Origin":      "http://localhost:3000",
		"Access-Control-Allow-Credentials": "true",
		"Access-Control-Allow-Methods":     "GET,POST,PUT,DELETE,OPTIONS",
		"Access-Control-Allow-Headers":     "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token",
	}

	// Handle state parameter and decode it
	stateParam := strings.TrimSpace(request.QueryStringParameters["state"])
	fmt.Println("CallbackHandler invoked with state:", stateParam)

	// Get composite state from query parameter
	composite, err := decodeState(stateParam)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, err.Error(), corsHeaders), nil
	}
	fmt.Printf("composite state %v", composite)

	// Get authorization code
	code := request.QueryStringParameters["code"]
	if code == "" {
		return errorResponse(http.StatusBadRequest, "missing code", corsHeaders), nil
	}

	fmt.Printf("code %v", code)

	// Exchange code for token
	token, err := getOAuthToken(code)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "authentication failed", corsHeaders), nil
	}

	// Verify token validity
	if !isTokenValid(token) {
		return errorResponse(http.StatusInternalServerError, "invalid token", corsHeaders), nil
	}

	// Store token in database
	err = cb.databaseStore.StoreToken(token)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "error inserting into database", corsHeaders), nil
	}

	// Initialize Google services
	googleServices, err := NewGoogleServices(token)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "failed to initialize services", corsHeaders), nil
	}

	// Get user info
	userInfo, err := getUserInfo(googleServices.Client)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "failed to get user info", corsHeaders), nil
	}

	// Process spreadsheet data
	sheetProcessor := NewSheetProcessor(googleServices.SheetsService)
	docs, err := sheetProcessor.ProcessSheetData(composite.SpreadsheetID, "Sheet1!A1:E10")
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "error processing spreadsheet data", corsHeaders), nil
	}

	// Send email with document summary
	emailSender := NewEmailSender(googleServices.GmailService, userInfo)
	err = emailSender.SendDocumentSummary(docs)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "error sending email", corsHeaders), nil
	}

	// Redirect to summary page
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusFound, // 302
		Headers: map[string]string{
			"Location":                         "http://localhost:3000/summary",
			"Content-Type":                     "application/json",
			"Access-Control-Allow-Origin":      "http://localhost:3000",
			"Access-Control-Allow-Credentials": "true",
			"Access-Control-Allow-Methods":     "GET,POST,PUT,DELETE,OPTIONS",
			"Access-Control-Allow-Headers":     "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token",
		},
		Body: "",
	}, nil
}

// Helper function to decode state parameter
func decodeState(stateParam string) (*compositeState, error) {
	raw, err := base64.URLEncoding.DecodeString(stateParam)
	if err != nil {
		return nil, fmt.Errorf("failed to decode state string: %v", err)
	}

	var composite compositeState
	err = json.Unmarshal(raw, &composite)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON from decoded state: %v", err)
	}

	return &composite, nil
}

// Helper function to get OAuth token
func getOAuthToken(code string) (*oauth2.Token, error) {
	oauthCfg := auth.NewAuthConfig().ToOAuth2Config()
	token, err := oauthCfg.Exchange(context.Background(), code)
	if err != nil {
		return nil, err
	}

	if token.Expiry.Before(time.Now()) && token.RefreshToken != "" {
		newToken, err := oauthCfg.TokenSource(context.Background(), token).Token()
		if err == nil {
			token = newToken
		}
	}

	return token, nil
}

// Helper function to verify token validity
func isTokenValid(token *oauth2.Token) bool {
	client := &http.Client{Timeout: 10 * time.Second}
	tokenInfoURL := fmt.Sprintf(
		"https://oauth2.googleapis.com/tokeninfo?access_token=%s",
		token.AccessToken,
	)

	resp, err := client.Get(tokenInfoURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		return false
	}
	defer resp.Body.Close()

	return true
}

// Helper function to get user info
func getUserInfo(client *http.Client) (*types.UserInfo, error) {
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var userInfo *types.UserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, err
	}

	return userInfo, nil
}

// Helper function for error responses
func errorResponse(statusCode int, message string, headers map[string]string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Body:       message,
		Headers:    headers,
	}
}
