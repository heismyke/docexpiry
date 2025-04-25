package database

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"golang.org/x/oauth2"
)

const TABLE_NAME = "Token"

type DynamoDBStore struct{
	DB *dynamodb.DynamoDB
}


func NewDynamoDBStore() *DynamoDBStore{
	dbSession := session.Must(session.NewSession())
	db := dynamodb.New(dbSession)
	return &DynamoDBStore{
		DB: db,
	}
}

func (db *DynamoDBStore) StoreToken(token *oauth2.Token) error {
	item := &dynamodb.PutItemInput{
		TableName: aws.String(TABLE_NAME),
		Item: map[string]*dynamodb.AttributeValue{
			"access_token" : {
				S: aws.String(token.AccessToken),
			},
			"token_type" : {
				S: aws.String(token.TokenType),
			},
			"expiry" : {
				S: aws.String(token.Expiry.Format(time.RFC3339)),
			},
		},
	}

	_, err := db.DB.PutItem(item)
	if err != nil{
		return fmt.Errorf("error inserting into database")
	}
	return nil
}

