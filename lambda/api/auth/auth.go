package auth

import (
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type AuthConfig struct{
	ClientID string
	ClientSecret string
	RedirectURL string
	Scopes []string
}

//https://spwzll7jm5.execute-api.eu-north-1.amazonaws.com/prod/oauth2callback

func NewAuthConfig() *AuthConfig{
	return &AuthConfig{
		ClientID:     "384543079988-lillmo592a40vt43dg3etuf3ghbg8s74.apps.googleusercontent.com",
        ClientSecret: "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",
        RedirectURL:  "https://spwzll7jm5.execute-api.eu-north-1.amazonaws.com/prod/oauth2callback",
        Scopes: []string{
            "https://www.googleapis.com/auth/userinfo.profile",
            "https://www.googleapis.com/auth/userinfo.email",
            "https://www.googleapis.com/auth/spreadsheets",
			"https://www.googleapis.com/auth/spreadsheets.readonly",
			"https://www.googleapis.com/auth/gmail.send",
			"https://www.googleapis.com/auth/gmail.labels",
			"https://www.googleapis.com/auth/calendar.events",
        },
	}
}


func (ac *AuthConfig) ToOAuth2Config() *oauth2.Config {
    return &oauth2.Config{
        ClientID:     ac.ClientID,
        ClientSecret: ac.ClientSecret,
        RedirectURL:  ac.RedirectURL,
        Scopes:       ac.Scopes,
        Endpoint:     google.Endpoint,
    }
}