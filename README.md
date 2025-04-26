![screencapture-localhost-3000-summary-2025-04-26-12_56_31](https://github.com/user-attachments/assets/fd9770db-da3f-4723-992d-2c86db588628)
# Real-Time Document Expiration Tracker

A serverless document management solution that automatically monitors document expiry dates using Go, AWS CDK, and Google APIs.

## Overview

This repository provides a complete solution for tracking document expirations in real-time. Built with Go and AWS serverless technologies, it connects to Google Sheets to fetch document metadata and sends notifications via Gmail when documents are approaching expiration.

## Features

- **Infrastructure as Code** - AWS CDK v2 in Go for defining all cloud resources
- **Scalable Storage** - DynamoDB TableV2 with auto-scaling and point-in-time recovery
- **Secure Google Integration** - OAuth2 for Google Sheets and Gmail APIs
- **Automated Notifications** - Weekly email summaries of expiring documents
- **Live Dashboard** - React-based frontend showing document status and actions

## Technology Stack

| Component | Technology |
|----------:|:-----------|
| **Language** | Go (Golang) |
| **Infrastructure** | AWS CDK v2 (Go) |
| **Database** | Amazon DynamoDB (TableV2) |
| **Compute** | AWS Lambda, EventBridge |
| **Google APIs** | Sheets API, Gmail API, OAuth2 |
| **Frontend** | React (Next.js) |

## Getting Started

1. **Clone the Repository**
   ```bash
   git clone https://github.com/yourusername/docexpiry.git
   cd docexpiry
   ```

2. **Bootstrap CDK**
   ```bash
   cdk bootstrap aws://ACCOUNT-NUMBER/REGION
   ```

3. **Configure Google Credentials**
   - Add `credentials.json` to `./credentials/`
   - Share your Google Sheet with the service account

4. **Deploy Infrastructure**
   ```bash
   cdk deploy
   ```

5. **Set Environment Variables**
   - Configure spreadsheet ID, recipients, and token storage

6. **Access the Dashboard**
   - Visit the API Gateway endpoint to view your document tracker

## License

Apache-2.0 License - See [LICENSE](LICENSE) for details.

