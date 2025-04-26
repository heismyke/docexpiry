package types

import "time"

type Token struct {
	// Primary identification
	ID     string `json:"id"`      // Unique token identifier (e.g., UUID)
	UserID string `json:"user_id"` // ID of the user this token belongs to
	Email  string `json:"email"`   // User's email for easier lookups

	// OAuth token data
	AccessToken  string    `json:"access_token"`  // The actual OAuth access token
	TokenType    string    `json:"token_type"`    // Usually "Bearer"
	RefreshToken string    `json:"refresh_token"` // Token used to get new access tokens
	Expiry       time.Time `json:"expiry"`        // When the access token expires
	ExpiresIn    int64     `json:"expires_in"`    // Seconds until expiration

	// Additional metadata
	CreatedAt time.Time `json:"created_at"` // When this token was first created
	LastUsed  time.Time `json:"last_used"`  // Last time this token was used
	Revoked   bool      `json:"revoked"`    // Flag to manually invalidate token

	// DynamoDB TTL field (Unix timestamp in seconds)
	TTL int64 `json:"ttl"` // Expiration time for DynamoDB TTL

	// Original response data
	Raw         interface{}
	ExpiryDelta time.Duration
}

type UserInfo struct {
	ID            string `json:"id"`               // Unique Google user ID
	Email         string `json:"email"`            // User’s email address
	VerifiedEmail bool   `json:"verified_email"`   // Is the email verified?
	Name          string `json:"name"`             // Full name
	GivenName     string `json:"given_name"`       // First name
	FamilyName    string `json:"family_name"`      // Last name
	Picture       string `json:"picture"`          // URL of profile picture
	Locale        string `json:"locale"`           // Preferred locale, e.g., "en"
	Link          string `json:"link,omitempty"`   // Profile URL (Google+ legacy)
	Gender        string `json:"gender,omitempty"` // Gender if available
	HD            string `json:"hd,omitempty"`     // Hosted domain for Google Workspace
}

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
