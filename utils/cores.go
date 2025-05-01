package utils

import (
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emersion/go-message"
	gomsgmail "github.com/emersion/go-message/mail"
)

var MailInfoCacheIns *MailInfoCache

func hashSubject(data []byte) uint64 {
	h := fnv.New64a()
	h.Write(data)
	return h.Sum64()
}

func AuthHandler(remoteAddr net.Addr, mechanism string, username []byte, password []byte, shared []byte) (ok bool, err error) { // 使用命名返回值
	defer func() {
		if r := recover(); r != nil {
			info := fmt.Sprintf("AuthHandler panic: %v", r)
			slog.Error(info)
			ok, err = false, errors.New(info)
		}
	}()

	// mechanism = strings.ToLower(mechanism)
	value, ok := CFG.SmtpdAuth.Mechanisms[mechanism]
	if !(ok && value) {
		slog.Warn(fmt.Sprintf("Unsupported authentication method %s", mechanism))
		return false, nil
	}
	user := string(username)
	pass := string(password)

	// check username and password
	if CFG.SmtpdAuth.AllowAnyAuth {
		slog.Warn(fmt.Sprintf("AllowAnyAuth Authentication successful method %s", mechanism), "Username", user)
		MailInfoCacheIns.SetUserPass(user, pass)
		return true, nil
	}
	if storedPass, ok := CFG.UserDB[user]; ok && storedPass == pass {
		slog.Info(fmt.Sprintf("Authentication successful method %s", mechanism), "Username", user)
		MailInfoCacheIns.SetUserPass(user, pass)
		return true, nil
	}
	slog.Error(fmt.Sprintf("Authentication failed method %s", mechanism), "Username", user)
	return false, nil
}

func MailHandler(remoteAddr net.Addr, from string, to []string, data []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			info := fmt.Sprintf("MailHandler panic: %v", r)
			slog.Error(info)
			err = errors.New(info)
		}
	}()

	ip, err := GetIPFromAddr(remoteAddr)
	if err != nil {
		slog.Error(err.Error())
		TriggerErrNotification(err.Error(), ip, from, to, data)
		return err
	}

	r := strings.NewReader(string(data))
	msg, err := message.Read(r)
	if err != nil {
		slog.Error(err.Error())
		TriggerErrNotification(err.Error(), ip, from, to, data)
		return err
	}

	// get mail header
	mailHeader := gomsgmail.Header{Header: msg.Header}
	// fromList, _ := mailHeader.Text("From")
	// to = mailHeader.Text("To") + mailHeader.Text("Cc")
	toList, _ := mailHeader.Text("To")
	ccList, _ := mailHeader.Text("Cc")
	subject, _ := mailHeader.Subject()

	slog.Info("Received an email", "ClientIP", ip, "From", from, "To", strings.Join(to, "; "), "email header To", toList, "email header Cc", ccList, "Subject", subject)
	slog.Info(fmt.Sprintf("Email size is %d bytes", len(data)))

	ValidateEmail := NewValidateEmail(ip, from, to, 0, 0, 0)
	// validate email sender client ip
	if err = ValidateEmail.ValidateEmailClientIP(); err != nil {
		TriggerErrNotification(err.Error(), ip, from, to, data)
		return err
	}

	// validate email sender
	if err = ValidateEmail.ValidateEmailSender(); err != nil {
		TriggerErrNotification(err.Error(), ip, from, to, data)
		return err
	}

	// validate email recipient
	if err = ValidateEmail.ValidateEmailRecipient(); err != nil {
		TriggerErrNotification(err.Error(), ip, from, to, data)
		return err
	}

	// Handle the content of the email.
	r = strings.NewReader(string(data))
	body, err := gomsgmail.CreateReader(r)
	if err != nil {
		TriggerErrNotification(err.Error(), ip, from, to, data)
		slog.Error(err.Error())
		return err
	}

	// Loop through reading each part of the body.
	mailPartType := NewMailPartType()
	mailBodyCount := 0
	for {
		p, err := body.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Error(err.Error())
			TriggerErrNotification(err.Error(), ip, from, to, data)
			return err
		}

		contentType := p.Header.Get("Content-Type")
		mailPartSize, err := CalculateReaderSize(p.Body)
		if err != nil {
			info := fmt.Sprintf("Failed to calculate the size of contentType: %s, error: %s", contentType, err.Error())
			slog.Error(info)
			TriggerErrNotification(err.Error(), ip, from, to, data)
			return errors.New(info)
		}

		currentPartType, err := mailPartType.CheckMailPartType(p)
		if err != nil {
			info := fmt.Sprintf("from user %s(%s) failed to check mail part type: %s, error: %s", from, ip, contentType, err.Error())
			slog.Error(info)
			TriggerErrNotification(err.Error(), ip, from, to, data)
			return errors.New(info)
		}
		if currentPartType == mailPartType.Body {
			// This is the message's text (can be plain-text or HTML)
			mailBodyCount += 1
			if mailBodyCount > 1 {
				info := "the email has more than one body, please check it"
				slog.Error(info)
				TriggerErrNotification(info, ip, from, to, data)
				return errors.New(info)
			}

			ValidateEmail.BodySize = mailPartSize
		} else if currentPartType == mailPartType.EmbeddedContent {
			ValidateEmail.EmbeddedContentSize += mailPartSize
		} else if currentPartType == mailPartType.Attachment {
			ValidateEmail.AttachmentSize += mailPartSize
		} else {
			info := "unknown header type"
			slog.Error(info)
			TriggerErrNotification(info, ip, from, to, data)
			return errors.New(info)
		}
	}

	// Validate the email body size
	if err = ValidateEmail.ValidateBodySize(); err != nil {
		TriggerErrNotification(err.Error(), ip, from, to, data)
		return err
	}
	// Validate the email attachment size
	if err = ValidateEmail.ValidateAttachments(); err != nil {
		TriggerErrNotification(err.Error(), ip, from, to, data)
		return err
	}
	// Validate the email embedded content size
	if err = ValidateEmail.ValidateEmbeddedContent(); err != nil {
		TriggerErrNotification(err.Error(), ip, from, to, data)
		return err
	}

	// After all the verifications have been passed, the email will be sent out.
	err = SendMailData(from, to, data)
	if err != nil {
		TriggerErrNotification(err.Error(), ip, from, to, data)
		return err
	}
	return nil
}

type mailPartType struct {
	Body            string
	Attachment      string
	EmbeddedContent string
	Unknown         string
}

func NewMailPartType() *mailPartType {
	return &mailPartType{"Body", "Attachment", "EmbeddedContent", "Unknown"}
}

func (mailPT *mailPartType) CheckMailPartType(p *gomsgmail.Part) (ret string, err error) {
	ret = mailPT.Unknown
	defer func() {
		if r := recover(); r != nil {
			info := fmt.Sprintf("CheckMailPartType panic: %v", r)
			slog.Error(info)
			err = errors.New(info)
		}
	}()

	contentType := p.Header.Get("Content-Type")
	contentId := p.Header.Get("Content-Id")

	switch h := p.Header.(type) {
	case *gomsgmail.InlineHeader:
		// This is the message's text (can be plain-text or HTML)
		if contentId != "" {
			slog.Warn("The file embedded in the email body", "Filename", contentId, "ContentType", contentType)
			return mailPT.EmbeddedContent, nil
		} else {
			// mail body
			slog.Info("The email body", "ContentType", contentType)
			return mailPT.Body, nil
		}
	case *gomsgmail.AttachmentHeader:
		// This is an attachment (Including the attached files and the pictures in the document)
		// h.ContentDisposition()
		filename, err := h.Filename()
		if err != nil || filename == "" {
			// filename = strings.Trim(contentId, "<>")
			slog.Warn("The file embedded in the email body", "Filename", contentId, "contentType", contentType)
			return mailPT.EmbeddedContent, nil
		} else {
			slog.Warn("Email attachment", "Filename", filename, "contentType", contentType)
			return mailPT.Attachment, nil
		}

	default:
		slog.Error("Unknown header type")
		return mailPT.Unknown, nil
	}

}

func SaveMail(data []byte) (file string, err error) {
	const EmailPath string = "emails"
	timestamp := time.Now().Format("20060102_150405")
	subjectHash := hashSubject(data)
	filename := fmt.Sprintf("%s_%d.eml", timestamp, subjectHash)

	err = os.MkdirAll(EmailPath, 0755)
	if err != nil {
		info := fmt.Sprintf("create email path %s failed: %s", EmailPath, err.Error())
		slog.Error(info)
		return "", errors.New(info)
	}

	file = filepath.Join(EmailPath, filename)
	err = os.WriteFile(file, data, 0644)
	if err != nil {
		info := fmt.Sprintf("saveMail failed: %s", err.Error())
		slog.Error(info)
		return file, errors.New(info)
	}
	return file, nil
}
