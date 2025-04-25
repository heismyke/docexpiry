package app

import (
	"lambda/api"
	"lambda/database"
)

type Application struct{
	ApiHandler *api.Api
}


func NewApplication() (*Application, error){
	db := database.NewDynamoDBStore()
	apiHandler := api.NewApi(db)
	return &Application{
		ApiHandler: apiHandler,
	}, nil
}