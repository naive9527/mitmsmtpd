package utils

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

// ValidateEmail performs comprehensive validation based on verification rules

type ValidateEmail struct {
	clientIP            string
	Sender              string
	Recipient           []string
	BodySize            int64
	AttachmentSize      int64
	EmbeddedContentSize int64
}

func NewValidateEmail(clientIP, sender string, recipient []string, bodySize, attachmentSize, embeddedContentSize int64) *ValidateEmail {
	email := new(ValidateEmail)
	email.clientIP = strings.TrimSpace(clientIP)
	email.Sender = strings.TrimSpace(sender)
	email.Recipient = recipient
	email.BodySize = bodySize
	email.AttachmentSize = attachmentSize
	email.EmbeddedContentSize = embeddedContentSize
	return email
}

func (email *ValidateEmail) ValidateEmailSender() error {
	if !CFG.VerificationRules.SenderRegexp.MatchString(email.Sender) {
		info := fmt.Sprintf("Invalid email sender: %s", email.Sender)
		slog.Error(info)
		return errors.New(info)
	}
	return nil
}

func (email *ValidateEmail) ValidateEmailRecipient() error {
	for _, recipient := range email.Recipient {
		recipient = strings.TrimSpace(recipient)
		if !CFG.VerificationRules.RecipientRegexp.MatchString(recipient) {
			info := fmt.Sprintf("Invalid email recipient: %v", email.Recipient)
			slog.Error(info)
			return errors.New(info)
		}
	}
	return nil
}

func (email *ValidateEmail) ValidateEmailClientIP() error {
	if !CFG.VerificationRules.SenderIPRegexp.MatchString(email.clientIP) {
		info := fmt.Sprintf("Invalid email clientIP: %s", email.clientIP)
		slog.Error(info)
		return errors.New(info)
	}
	return nil
}

func (email *ValidateEmail) ValidateBodySize() error {
	// Check Email BodySize
	slog.Info(fmt.Sprintf("Mail Body Size %d bytes", email.BodySize))
	if CFG.VerificationRules.EmailBodySize == 0 {
		return nil
	}

	if email.BodySize <= int64(CFG.VerificationRules.EmailBodySize) {
		return nil
	} else {
		info := fmt.Sprintf("Email body size is too large: %d Bytes", email.BodySize)
		slog.Error(info)
		return errors.New(info)
	}
}
func (email *ValidateEmail) ValidateAttachments() error {
	// Check Email attachments
	info := ""
	if email.AttachmentSize == 0 {
		return nil
	}
	slog.Info(fmt.Sprintf("Mail Attachments Size %d bytes", email.AttachmentSize))
	if CFG.VerificationRules.Attachment.Allowed {
		if CFG.VerificationRules.Attachment.MaxSize == 0 || email.AttachmentSize <= int64(CFG.VerificationRules.Attachment.MaxSize) {
			return nil
		} else {
			info = fmt.Sprintf("Email attachment size is too large: %d Bytes", email.AttachmentSize)
		}
	} else {
		info = "Attachments are not allowed to be sent."
	}
	slog.Error(info)
	return errors.New(info)
}

func (email *ValidateEmail) ValidateEmbeddedContent() error {
	// Check Email Embedded Content
	info := ""
	if email.EmbeddedContentSize == 0 {
		return nil
	}
	slog.Info(fmt.Sprintf("Mail Embedded Content Size %d bytes", email.EmbeddedContentSize))
	if CFG.VerificationRules.EmbeddedContent.Allowed {
		if CFG.VerificationRules.EmbeddedContent.MaxSize == 0 || email.EmbeddedContentSize <= int64(CFG.VerificationRules.EmbeddedContent.MaxSize) {
			return nil
		} else {
			info = fmt.Sprintf("Email embedded content size is too large: %d Bytes", email.EmbeddedContentSize)
		}
	} else {
		info = "embedded content are not allowed to be sent."
	}
	slog.Error(info)
	return errors.New(info)
}
