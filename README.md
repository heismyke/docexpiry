

## Overview  
This repository hosts a **Real-Time Document Expiration Tracker**, a serverless document management solution designed to automate the monitoring of document expiry dates. The system is implemented in **Go**, with infrastructure defined via **AWS CDK v2** in Go, provisioning a **DynamoDB** backend for low-latency state storage and **AWS Lambda** functions for scheduled scans and notifications. It integrates with **Google Sheets** and **Gmail** using OAuth2 to fetch document metadata and send weekly summary emails to stakeholders. citeturn0search0turn1search1  

## Features  
### 1. Infrastructure as Code with AWS CDK (Go)  
- **AWS CDK v2 in Go**: Defines all AWS resources—including DynamoDB tables, Lambda functions, IAM roles, and EventBridge schedules—using idiomatic Go constructs. citeturn0search0  
- **Scalable DynamoDB Tables**: Utilizes the new **TableV2** construct for single- and multi-region global tables, with auto-scaling, point-in-time recovery, and per-replica configuration. citeturn1search1  

### 2. Secure OAuth2 Integration  
- **Google OAuth2 via Go Libraries**: Leverages `golang.org/x/oauth2/google` for service-account and installed-app flows to obtain tokens for Google APIs. citeturn4search3  
- **Google Sheets API**: Uses the official Sheets client (`google.golang.org/api/sheets/v4`) to read document metadata (IDs, expiration dates, owners) from a shared spreadsheet. citeturn2search0  

### 3. Automated Weekly Summaries  
- **Scheduled Lambda**: An EventBridge rule triggers a Go-based Lambda function every week to query DynamoDB for items nearing expiration.  
- **Gmail API Dispatch**: Compiles findings into human-readable summaries and sends emails via the Gmail API (`google.golang.org/api/gmail/v1`) to configured stakeholders. citeturn3search0  

### 4. Real-Time React Dashboard  
- **Next.js Frontend**: Single-page React application served via API Gateway and Lambda, pulling live data from DynamoDB to display countdowns, alert levels, and action links (renew/archive).  

## Technology Stack  

| Component                      | Technology                                    |
|-------------------------------:|:----------------------------------------------|
| **Language**                    | Go (Golang)                                   |
| **Infrastructure as Code**      | AWS CDK v2 (Go) citeturn0search0               |
| **Data Store**                  | Amazon DynamoDB (TableV2) citeturn1search1     |
| **OAuth2 Library**              | `golang.org/x/oauth2/google` citeturn4search3 |
| **Sheets Integration**          | `google.golang.org/api/sheets/v4` citeturn2search0 |
| **Email API**                   | `google.golang.org/api/gmail/v1` citeturn3search1 |
| **Compute & Scheduling**        | AWS Lambda, EventBridge                       |
| **Frontend**                    | React (Next.js) via API Gateway & Lambda      |

## Getting Started  

1. **Clone the Repository**  
   ```bash
   git clone https://github.com/yourusername/document-expiration-tracker.git
   cd document-expiration-tracker
   ```  
2. **Bootstrap CDK (Go)**  
   ```bash
   cdk bootstrap aws://ACCOUNT-NUMBER/REGION
   ```  
3. **Configure Google Credentials**  
   - Place `credentials.json` (OAuth client or service account) in `./credentials/`.  
   - Share your target Google Sheet with the service account email.  
4. **Deploy Infrastructure**  
   ```bash
   cdk deploy
   ```  
5. **Configure Environment Variables**  
   Set Lambda environment variables for spreadsheet ID, Gmail recipients, and OAuth token storage.  
6. **Monitor & Iterate**  
   - View DynamoDB entries in the AWS console.  
   - Access the React dashboard at the deployed API Gateway endpoint.  

## License  
This project is licensed under the **Apache-2.0 License**. See [LICENSE](LICENSE) for details.

