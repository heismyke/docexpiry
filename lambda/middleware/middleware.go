package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"golang.org/x/oauth2"
	"lambda/database"
	"lambda/types"
	"net/http"
	"os"
	"strings"
	"time"
)

type TokenMiddleware struct {
	DB *database.DynamoDBStore
}

func NewTokenMiddleware() *TokenMiddleware {
	return &TokenMiddleware{DB: database.NewDynamoDBStore()}
}

func (tm *TokenMiddleware) HandleRequest(
	ctx context.Context,
	request events.APIGatewayProxyRequest,
	handler func(ctx2 context.Context, proxyRequest events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error),
) (events.APIGatewayProxyResponse, error) {
	// function body

	userID := getUserIDFromRequest(request)
	if userID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusUnauthorized,
			Body:       "Unauthorized: Missing user identification",
		}, nil
	}
	token, err := tm.GetToken(userID)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusUnauthorized,
			Body:       "Unauthorized: Cannot retrieve token",
		}, nil
	}
	oauthToken := &oauth2.Token{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}

	if oauthToken.Expiry.Before(time.Now().Add(5 * time.Minute)) {
		// Token is expired or about to expire, refresh it
		newOauthToken, err := refreshToken(oauthToken)
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusUnauthorized,
				Body:       "Unauthorized: Token refresh failed",
			}, nil
		}

		// Update our token struct with new values
		token.AccessToken = newOauthToken.AccessToken
		token.TokenType = newOauthToken.TokenType
		token.RefreshToken = newOauthToken.RefreshToken
		token.Expiry = newOauthToken.Expiry

		// Save the new token to database
		if err := tm.DB.StoreToken(token); err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       "Internal server error: Failed to save refreshed token",
			}, nil
		}

		// Update token for this request
		oauthToken = newOauthToken
	} else if !isTokenValid(oauthToken) {
		// Additional validation to check if token is still valid with Google
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusUnauthorized,
			Body:       "Unauthorized: Invalid token",
		}, nil
	}

	ctxWithToken := context.WithValue(ctx, "oauth_token", oauthToken)
	ctxWithUserToken := context.WithValue(ctxWithToken, "user_token", token)

	// Call the next handler with our new context
	return handler(ctxWithUserToken, request)

}
func (tm *TokenMiddleware) GetToken(userID string) (*types.Token, error) {

	result, err := tm.DB.DB.Query(&dynamodb.QueryInput{
		TableName:              aws.String(database.TABLE_NAME),
		IndexName:              aws.String("UserID-index"),
		KeyConditionExpression: aws.String("UserID = id"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":uid": {
				S: aws.String(userID),
			},
		},
		ScanIndexForward: aws.Bool(false),
		Limit:            aws.Int64(1),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query token %w", err)
	}
	if len(result.Items) == 0 {
		return nil, fmt.Errorf("no token found for user %s", userID)
	}

	item := result.Items[0]

	expiry, err := time.Parse(time.RFC3339, *item["Expiry"].S)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expiry: %w", err)
	}
	token := &types.Token{
		UserID:       *item["UserID"].S,
		Email:        *item["Email"].S,
		AccessToken:  *item["AccessToken"].S,
		TokenType:    *item["TokenType"].S,
		RefreshToken: *item["RefreshToken"].S,
		Expiry:       expiry,
	}

	return token, nil
}
func isTokenValid(token *oauth2.Token) bool {
	client := &http.Client{Timeout: 10 * time.Second}
	tokenInfoURL := fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?access_token=%s", token.AccessToken)

	resp, err := client.Get(tokenInfoURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		return false
	}
	defer resp.Body.Close()
	return true
}
func refreshToken(oldToken *oauth2.Token) (*oauth2.Token, error) {
	if oldToken.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	refreshURL := "https://oauth2.googleapis.com/token"
	reqBody := fmt.Sprintf(
		"refresh_token=%s&client_id=%s&client_secret=%s&grant_type=refresh_token",
		oldToken.RefreshToken,
		getClientID(), // Implement this function to get your client ID
		getClientSecret(),
	)

	req, err := http.NewRequest("POST", refreshURL, strings.NewReader(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed with status: %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	newToken := &oauth2.Token{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		RefreshToken: oldToken.RefreshToken, // Keep the refresh token
		Expiry:       time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}

	return newToken, nil
}

func getClientID() string {
	// Return your OAuth client ID from environment or config
	return os.Getenv("GOOGLE_CLIENT_ID")
}

func getClientSecret() string {
	// Return your OAuth client secret from environment or config
	return os.Getenv("GOOGLE_CLIENT_SECRET")
}

// Helper function to extract user ID from request
func getUserIDFromRequest(req events.APIGatewayProxyRequest) string {
	// Extract from headers, query parameters, or JWT token
	// This is an example - customize to your auth system
	if userID, ok := req.Headers["X-User-ID"]; ok {
		return userID
	}

	// Or extract from Authorization header if you're using JWTs
	return ""
}
