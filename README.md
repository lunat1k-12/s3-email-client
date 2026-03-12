# S3 Email Client

A terminal-based email client for viewing and responding to emails stored in Amazon S3 buckets. Built with Go and the Bubble Tea TUI framework.

## Overview

S3 Email Client provides a text-based user interface (TUI) for managing emails that are stored in S3 via Amazon SES inbound email receiving. It allows you to browse, view, and respond to emails directly from your terminal.

## Features

- 📧 Browse emails stored in S3 buckets
- 📖 View email content with HTML-to-text rendering
- 📎 Handle email attachments
- ✍️ Compose and send email responses via Amazon SES
- ⌨️ Keyboard-driven navigation
- 🎨 Clean terminal UI built with Bubble Tea
- ⚡ Optional in-memory caching for performance

## How It Works

This application integrates with AWS services in the following way:

1. **Amazon SES Inbound Email**: SES receives emails and stores them as raw MIME files in an S3 bucket
2. **S3 Storage**: Emails are stored as objects in your configured S3 bucket
3. **S3 Email Client**: This application lists, downloads, and parses emails from S3
4. **Amazon SES Outbound**: When you reply to emails, responses are sent via SES

```
┌─────────────┐      ┌─────────────┐      ┌──────────────────┐
│   Incoming  │─────▶│  Amazon SES │─────▶│   S3 Bucket      │
│   Email     │      │  (Inbound)  │      │  (Email Storage) │
└─────────────┘      └─────────────┘      └──────────────────┘
                                                    │
                                                    ▼
                                           ┌──────────────────┐
                                           │  S3 Email Client │
                                           │  (This App)      │
                                           └──────────────────┘
                                                    │
                                                    ▼
                                           ┌──────────────────┐
                                           │   Amazon SES     │
                                           │   (Outbound)     │
                                           └──────────────────┘
                                                    │
                                                    ▼
                                           ┌──────────────────┐
                                           │  Outgoing Email  │
                                           └──────────────────┘
```

## Prerequisites

### 1. AWS Account and Credentials

You need an AWS account with credentials configured. Set up credentials using one of these methods:

- **AWS CLI**: Run `aws configure`
- **Environment variables**: Set `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`
- **IAM role**: If running on EC2, use an IAM instance role

### 2. Amazon SES Setup

#### Verify Your Email Address (Required for Sending)

Before you can send email responses, you must verify your email address in SES:

1. Go to the [AWS SES Console](https://console.aws.amazon.com/ses/)
2. Navigate to **Verified identities**
3. Click **Create identity**
4. Select **Email address** and enter your email
5. Click **Create identity**
6. Check your inbox for a verification email from AWS
7. Click the verification link in the email

**Note**: If your SES account is in sandbox mode, you can only send emails to verified addresses. To send to any address, request production access in the SES console.

#### Configure SES Inbound Email (Required for Receiving)

To receive emails and store them in S3, you need to set up an SES receipt rule set:

1. **Create an S3 Bucket** (if you don't have one):
   ```bash
   aws s3 mb s3://my-email-bucket --region us-east-1
   ```

2. **Grant SES Permission to Write to S3**:
   
   Create a bucket policy that allows SES to put objects:
   ```bash
   aws s3api put-bucket-policy --bucket my-email-bucket --policy '{
     "Version": "2012-10-17",
     "Statement": [
       {
         "Sid": "AllowSESPuts",
         "Effect": "Allow",
         "Principal": {
           "Service": "ses.amazonaws.com"
         },
         "Action": "s3:PutObject",
         "Resource": "arn:aws:s3:::my-email-bucket/*",
         "Condition": {
           "StringEquals": {
             "AWS:SourceAccount": "YOUR_AWS_ACCOUNT_ID"
           }
         }
       }
     ]
   }'
   ```

3. **Create a Receipt Rule Set**:
   ```bash
   aws ses create-receipt-rule-set --rule-set-name my-email-rules --region us-east-1
   ```

4. **Set it as Active**:
   ```bash
   aws ses set-active-receipt-rule-set --rule-set-name my-email-rules --region us-east-1
   ```

5. **Create a Receipt Rule**:
   ```bash
   aws ses create-receipt-rule \
     --rule-set-name my-email-rules \
     --rule '{
       "Name": "store-emails-in-s3",
       "Enabled": true,
       "Recipients": ["your-domain.com"],
       "Actions": [
         {
           "S3Action": {
             "BucketName": "my-email-bucket",
             "ObjectKeyPrefix": "emails/"
           }
         }
       ]
     }' \
     --region us-east-1
   ```

6. **Verify Your Domain** (for receiving emails):
   - In the SES console, go to **Verified identities**
   - Click **Create identity** and select **Domain**
   - Follow the instructions to add DNS records to your domain

### 3. IAM Permissions

Your AWS credentials need the following permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:ListBucket",
        "s3:GetObject",
        "s3:DeleteObject"
      ],
      "Resource": [
        "arn:aws:s3:::my-email-bucket",
        "arn:aws:s3:::my-email-bucket/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "ses:SendEmail",
        "ses:SendRawEmail"
      ],
      "Resource": "*"
    }
  ]
}
```

## Installation

### From Source

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd s3emailclient
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   ```

3. **Build the application**:
   ```bash
   go build -o bin/s3emailclient ./cmd/s3emailclient
   ```

4. **Install to your PATH** (optional):
   ```bash
   sudo cp bin/s3emailclient /usr/local/bin/
   ```

## Configuration

### Create Configuration File

1. **Create the config directory**:
   ```bash
   mkdir -p ~/.config/s3emailclient
   ```

2. **Copy the example configuration**:
   ```bash
   cp config.example.yaml ~/.config/s3emailclient/config.yaml
   ```

3. **Edit the configuration**:
   ```bash
   nano ~/.config/s3emailclient/config.yaml
   ```

### Configuration Options

```yaml
# S3 Configuration (required)
bucket_name: "my-email-bucket"  # Your S3 bucket name
region: "us-east-1"             # AWS region

# AWS Profile (optional)
# aws_profile: "my-profile"     # Use a specific AWS CLI profile

# Email Response Configuration (required for sending)
source_email: "user@example.com"  # Your verified SES email address

# SES Region (optional)
# ses_region: "us-east-1"       # Defaults to S3 region if not specified

# UI Configuration (optional)
list_pane_width: 40  # Email list width as percentage (10-90)
refresh_rate: 100    # UI refresh rate in milliseconds (min: 10)

# Behavior Configuration (optional)
cache_emails: false   # Cache parsed emails in memory
max_cache_size: 50   # Maximum number of cached emails
```

### Environment Variables

You can also configure the application using environment variables:

```bash
export S3EMAIL_BUCKET_NAME="my-email-bucket"
export S3EMAIL_REGION="us-east-1"
export S3EMAIL_SOURCE_EMAIL="user@example.com"
export S3EMAIL_AWS_PROFILE="my-profile"
export S3EMAIL_SES_REGION="us-east-1"
export S3EMAIL_LIST_PANE_WIDTH=40
export S3EMAIL_REFRESH_RATE=100
export S3EMAIL_CACHE_EMAILS=true
export S3EMAIL_MAX_CACHE_SIZE=50
```

Environment variables take precedence over the config file.

## Usage

### Run the Application

```bash
# If installed to PATH
s3emailclient

# Or run directly
./bin/s3emailclient

# Or with go run
go run ./cmd/s3emailclient/main.go
```

### Keyboard Shortcuts

- `j/k` - Navigate email list
- `Enter` - View selected email
- `r` - Reply to current email
- `d` - Delete current email
- `Esc` - Go back / Cancel
- `q` - Quit application
- `Ctrl+C` - Force quit

### Composing Replies

1. Press `r` while viewing an email
2. Type your response in the compose window
3. Press `Ctrl+S` to send
4. Press `Esc` to cancel

## Development

### Project Structure

```
s3emailclient/
├── cmd/s3emailclient/    # Application entry point
├── internal/
│   ├── app/              # Application orchestration
│   ├── config/           # Configuration management
│   ├── navigation/       # Navigation and actions
│   ├── parser/           # Email MIME parsing
│   ├── response/         # Email response handling
│   ├── s3client/         # S3 operations
│   ├── sesclient/        # SES operations
│   └── tui/              # Terminal UI components
├── config.example.yaml   # Example configuration
└── go.mod                # Go module definition
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...

# Run property-based tests
go test -v ./internal/tui -run TestCompose
```

### Common Development Commands

```bash
# Format code
go fmt ./...

# Tidy dependencies
go mod tidy

# Build
go build -o bin/s3emailclient ./cmd/s3emailclient

# Run
go run ./cmd/s3emailclient/main.go
```

## Tech Stack

- **Language**: Go 1.24.6
- **TUI Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- **AWS SDK**: [aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2)
- **Email Parsing**: [enmime](https://github.com/jhillyerd/enmime)
- **Configuration**: [Viper](https://github.com/spf13/viper)
- **Testing**: [gopter](https://github.com/leanovate/gopter) (property-based testing)

## Troubleshooting

### "Configuration error: bucket_name is required"

Make sure you've created `~/.config/s3emailclient/config.yaml` and set the `bucket_name` field.

### "Failed to load email list: AccessDenied"

Check that:
1. Your AWS credentials are configured correctly
2. Your IAM user/role has `s3:ListBucket` and `s3:GetObject` permissions
3. The bucket name in your config matches the actual bucket

### "Failed to send email: MessageRejected"

Check that:
1. Your `source_email` is verified in the SES console
2. If in SES sandbox mode, the recipient email is also verified
3. Your IAM user/role has `ses:SendEmail` permissions

### "No emails found"

Check that:
1. Your SES receipt rule set is active
2. The receipt rule is configured to write to the correct S3 bucket
3. Your domain is verified in SES for receiving emails
4. DNS records for your domain are correctly configured

## License

[Add your license here]

## Contributing

[Add contribution guidelines here]
