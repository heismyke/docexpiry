package database

import (
	"fmt"
	"github.com/google/uuid"
	"lambda/types"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

const TABLE_NAME = "Token"

type DynamoDBStore struct {
	DB *dynamodb.DynamoDB
}

func NewDynamoDBStore() *DynamoDBStore {
	dbSession := session.Must(session.NewSession())
	db := dynamodb.New(dbSession)
	return &DynamoDBStore{
		DB: db,
	}
}

func (db *DynamoDBStore) StoreToken(token *types.Token) error {
	// Generate a unique ID for this token
	tokenID := uuid.New().String()

	// Calculate TTL (e.g., 30 days from now)
	ttl := time.Now().Add(30 * 24 * time.Hour).Unix()

	item := &dynamodb.PutItemInput{
		TableName: aws.String(TABLE_NAME),
		Item: map[string]*dynamodb.AttributeValue{
			"ID": {
				S: aws.String(tokenID),
			},
			"UserID": {
				S: aws.String(token.UserID),
			},
			"Email": {
				S: aws.String(token.Email),
			},
			"AccessToken": {
				S: aws.String(token.AccessToken),
			},
			"TokenType": {
				S: aws.String(token.TokenType),
			},
			"RefreshToken": {
				S: aws.String(token.RefreshToken),
			},
			"Expiry": {
				S: aws.String(token.Expiry.Format(time.RFC3339)),
			},
			"ExpiresIn": {
				N: aws.String(fmt.Sprintf("%d", int64(token.Expiry.Sub(time.Now()).Seconds()))),
			},
			"CreatedAt": {
				S: aws.String(time.Now().Format(time.RFC3339)),
			},
			"LastUsed": {
				S: aws.String(time.Now().Format(time.RFC3339)),
			},
			"Revoked": {
				BOOL: aws.Bool(false),
			},
			"TTL": {
				N: aws.String(fmt.Sprintf("%d", ttl)),
			},
		},
	}

	_, err := db.DB.PutItem(item)
	if err != nil {
		return fmt.Errorf("error inserting token into database: %w", err)
	}

	return nil
}
