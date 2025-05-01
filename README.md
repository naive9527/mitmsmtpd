**Read this in other languages: [English](README.md), [中文](README_ZH.md).**

## Start the Service
go build .      # Compile to generate mitmsmtpd executable  
Copy config_sample.yaml to config.yaml and modify with your own configuration
./mitmsmtpd     # Start the service

## Business Workflow
For ease of explanation, assume the following information:

Mail Server: smtp.example.com
Mail Users: user01@example.com, user02@example.com
MUA's IP: 10.10.20.10
mitmsmtpd's IP: 10.10.20.111

### Normal Scenario
1、user01@example.com composes an email on 10.10.20.10 with recipient user02@example.com
2、Successfully logs into smtp.example.com
3、Delivers the email successfully

### With mitmsmtpd
1、Modify DNS resolution of smtp.example.com on 10.10.20.10 to point to mitmsmtpd's IP (or directly specify mitmsmtpd's IP as the SMTP server address)
2、user01@example.com composes an email on 10.10.20.10 with recipient user02@example.com
3、Successfully logs into mitmsmtpd
4、mitmsmtpd validates the email against predefined rules. If invalid, returns an error; if valid, proceeds
5、mitmsmtpd uses the sender's credentials to log into the real SMTP server (determined by sender user01@example.com and config.yaml)
6、If login fails, returns an error; if successful, proceeds
7、mitmsmtpd relays the email
8、If any step fails:
    Returns error information
    Saves the entire email as an .eml file
    Notifies administrators (email notification currently implemented)

## TLS Configuration
### Use Real TLS Certificate
Apply for an official TLS certificate and private key to secure SMTP service.

### Use Self-Signed Certificate (e.g., using cfssl)
Generate a private CA and configure it on MUAs
Generate a certificate using the private CA and configure it on mitmsmtpd


