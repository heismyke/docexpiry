package app

import (
	"lambda/api"
	"lambda/database"
)

type Application struct {
	LoginHandler    *api.LoginHandler
	CallbackHandler *api.CallBackHandler
}

func NewApplication() (*Application, error) {
	db := database.NewDynamoDBStore()
	callbackHandler := api.NewCallbackHandler(db)
	return &Application{
		CallbackHandler: callbackHandler,
	}, nil
}
