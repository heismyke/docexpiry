package reminders

import (
	"fmt"
	"net/smtp"
	"time"
)

// Document represents a record with issue and expiration dates
type Document struct {
    ID          string
    Name        string
    IssueDate   time.Time
    ExpireDate  time.Time
    ClientEmail string
}

// EmailConfig holds SMTP configuration
type EmailConfig struct {
    Host     string
    Port     int
    Username string
    Password string
    From     string
}

// SendEmail sends an email via SMTP
func SendEmail(cfg EmailConfig, to, subject, body string) error {
    addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
    msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", cfg.From, to, subject, body)
    auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
    return smtp.SendMail(addr, auth, cfg.From, []string{to}, []byte(msg))
}

// CheckAndNotify checks documents for upcoming expiration and sends reminders
func CheckAndNotify(docs []Document, cfg EmailConfig) {
    now := time.Now()
    for _, doc := range docs {
        daysUntilExpiry := int(doc.ExpireDate.Sub(now).Hours() / 24)
        if daysUntilExpiry >= 0 && daysUntilExpiry <= 7 {
            subject := fmt.Sprintf("Reminder: '%s' expires in %d days", doc.Name, daysUntilExpiry)
            body := fmt.Sprintf("Document %s (ID: %s) will expire on %s.", doc.Name, doc.ID, doc.ExpireDate.Format("2006-01-02"))
            // Send to admin
            if err := SendEmail(cfg, cfg.From, subject, body); err != nil {
                fmt.Printf("Failed to send admin reminder for %s: %v\n", doc.ID, err)
            }
            // Send to client
            if err := SendEmail(cfg, doc.ClientEmail, subject, body); err != nil {
                fmt.Printf("Failed to send client reminder for %s: %v\n", doc.ID, err)
            }
        }
    }
}

// WeeklySummary sends a summary email of all upcoming expirations within a window
func WeeklySummary(docs []Document, cfg EmailConfig) {
    now := time.Now()
    cutoff := now.AddDate(0, 0, 7)
    body := "Weekly Expiration Summary:\n"
    count := 0
    for _, doc := range docs {
        if doc.ExpireDate.After(now) && doc.ExpireDate.Before(cutoff) {
            body += fmt.Sprintf("- %s (ID: %s) expires on %s, client: %s\n",
                doc.Name, doc.ID, doc.ExpireDate.Format("2006-01-02"), doc.ClientEmail)
            count++
        }
    }
    if count == 0 {
        body += "No documents expiring within the next week."
    }
    subject := "Weekly Document Expiration Summary"
    if err := SendEmail(cfg, cfg.From, subject, body); err != nil {
        fmt.Printf("Failed to send weekly summary: %v\n", err)
    }
}

// Example of scheduling (e.g., with a cron job or AWS EventBridge) calling these functions
func SchedulerExample() {
    // Load docs from database or spreadsheet
    docs := fetchDocuments()
    cfg := loadEmailConfig()

    // Daily check (run every day)
    CheckAndNotify(docs, cfg)

    // Weekly summary (run once a week)
    WeeklySummary(docs, cfg)
}

// Placeholder functions
func fetchDocuments() []Document {
    // Implement fetching and parsing values from your sheet
    return nil
}

func loadEmailConfig() EmailConfig {
    // Load SMTP config from env or config file
    return EmailConfig{}
}
