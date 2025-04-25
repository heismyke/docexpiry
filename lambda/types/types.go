package types

import "time"

type Token struct{
	AccessToken string `json:"access_token"`
	TokenType string 	`json:"token_type"`
	Expiry time.Time `json:"expiry"`
}

type UserInfo struct {
    ID            string `json:"id"`              // Unique Google user ID
    Email         string `json:"email"`           // User’s email address
    VerifiedEmail bool   `json:"verified_email"`  // Is the email verified?
    Name          string `json:"name"`            // Full name
    GivenName     string `json:"given_name"`      // First name
    FamilyName    string `json:"family_name"`     // Last name
    Picture       string `json:"picture"`         // URL of profile picture
    Locale        string `json:"locale"`          // Preferred locale, e.g., "en"
    Link          string `json:"link,omitempty"`  // Profile URL (Google+ legacy)
    Gender        string `json:"gender,omitempty"`// Gender if available
    HD            string `json:"hd,omitempty"`    // Hosted domain for Google Workspace
}