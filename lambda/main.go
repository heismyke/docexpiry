package main

import (
	"fmt"
	"lambda/app"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type TestEvent struct {
	ID    string `json:"id"`
	Gmail string `json:"gmail"`
}

func TestEventHandler(event TestEvent) (string, error) {
	if event.ID == "" {
		return "", fmt.Errorf("id field is empty")
	}
	return fmt.Sprintf("successfully called by %s", event.Gmail), nil
}

func main() {
	myApp, err := app.NewApplication()
	if err != nil {
		panic(err)
	}
	lambda.Start(func(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		if request.HTTPMethod == "OPTIONS" {
			return events.APIGatewayProxyResponse{
				StatusCode: 200,
				Headers: map[string]string{
					"Access-Control-Allow-Origin":      "http://localhost:3000",
					"Access-Control-Allow-Credentials": "true",
					"Access-Control-Allow-Methods":     "GET,POST,PUT,DELETE,OPTIONS",
					"Access-Control-Allow-Headers":     "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token",
				},
				Body: "",
			}, nil
		}

		switch request.Path {
		case "/login":
			return myApp.LoginHandler.GetSpreedSheetAndRedirect(request)
		case "/oauth2callback":
			return myApp.CallbackHandler.OauthCallback(request)
		default:
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusBadRequest,
				Body:       "Invalid Request",
				Headers: map[string]string{
					"Content-Type":                     "application/json",
					"Access-Control-Allow-Origin":      "http://localhost:3000",
					"Access-Control-Allow-Credentials": "true",
					"Access-Control-Allow-Methods":     "GET,POST,PUT,DELETE,OPTIONS",
					"Access-Control-Allow-Headers":     "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token",
				},
			}, nil
		}
	})
}
