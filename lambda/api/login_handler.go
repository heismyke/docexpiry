package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"golang.org/x/oauth2"
	"lambda/api/auth"
	"lambda/database"
	"net/http"
	"strings"
)

type LoginHandler struct {
	sessionStore *database.DynamoDBStore
}

func NewLoginHandler() *LoginHandler {
	return &LoginHandler{}
}

func (lh *LoginHandler) GetSpreedSheetAndRedirect(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
	authConfig := auth.NewAuthConfig()
	oauthConfig := authConfig.ToOAuth2Config()
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
