package utils

import (
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	"github.com/emersion/go-message"
	gomail "github.com/emersion/go-message/mail"
)

func SaveMail(data []byte) error {
	timestamp := time.Now().Format("20060102_150405")
	subjectHash := hashSubject(data)
	filename := fmt.Sprintf("%s_%d.eml", timestamp, subjectHash)
	return os.WriteFile(filename, data, 0644)
}

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

	mechanism = strings.ToLower(mechanism)
	value, ok := CFG.SmtpdAuth.Mechanisms[mechanism]
	if !(ok && value) {
		slog.Warn(fmt.Sprintf("Unsupported authentication method %s", mechanism))
		return false, nil
	}
	user := string(username)
	pass := string(password)

	// check username and password
	if storedPass, ok := CFG.UserDBMap[user]; ok && storedPass == pass {
		slog.Info(fmt.Sprintf("Authentication successful method %s", mechanism), "Username", user)
		return true, nil
	}
	slog.Error(fmt.Sprintf("Authentication failed method %s", mechanism), "Username", user)
	return false, nil
}

func MailHandler(remoteAddr net.Addr, from string, to []string, data []byte) error {
	// err := SaveMail(data)
	// if err != nil {
	// 	return err
	// }

	ip, err := GetIPFromAddr(remoteAddr)
	if err != nil {
		slog.Error(err.Error())
	}

	r := strings.NewReader(string(data))
	msg, err := message.Read(r)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	// get mail header
	mailHeader := gomail.Header{Header: msg.Header}
	// fromList, _ := mailHeader.Text("From")
	// to = mailHeader.Text("To") + mailHeader.Text("Cc")
	toList, _ := mailHeader.Text("To")
	ccList, _ := mailHeader.Text("Cc")
	subject, _ := mailHeader.Subject()

	slog.Info("Received an email", "ClientIP", ip, "From", from, "To", strings.Join(to, "; "), "email header To", toList, "email header Cc", ccList, "Subject", subject)

	ValidateEmail := NewValidateEmail(ip, from, to, 0, 0, 0)
	// validate email sender client ip
	if err := ValidateEmail.ValidateEmailClientIP(); err != nil {
		return err
	}

	// validate email sender
	if err := ValidateEmail.ValidateEmailSender(); err != nil {
		return err
	}

	// validate email recipient
	if err := ValidateEmail.ValidateEmailRecipient(); err != nil {
		return err
	}

	// Handle the body of the email.
	r = strings.NewReader(string(data))
	body, err := gomail.CreateReader(r)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	// Loop through reading each part of the body.
	for {
		p, err := body.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Error(err.Error())
			return err
		}

		switch h := p.Header.(type) {
		case *gomail.InlineHeader:
			// This is the message's text (can be plain-text or HTML)
			b, _ := io.ReadAll(p.Body)
			ValidateEmail.BodySize = int64(len(b))
			slog.Info(fmt.Sprintf("Body: %s", string(b)))

		case *gomail.AttachmentHeader:
			// This is an attachment (Including the attached files and the pictures in the document)
			contentType := p.Header.Get("Content-Type")
			tmpsize, err := CalculateReaderSize(p.Body)
			if err != nil {
				info := fmt.Sprintf("Failed to calculate the size of the attachment contentType: %s, error: %s", contentType, err.Error())
				slog.Error(info)
				return errors.New(info)
			}

			filename, err := h.Filename()
			if err != nil || filename == "" {
				cid := strings.Trim(h.Get("Content-Id"), "<>")
				filename = cid
				ValidateEmail.EmbeddedContentSize += tmpsize
				slog.Warn("The file embedded in the email body", "Fileanem", filename, "contentType", contentType)
			} else {
				ValidateEmail.AttachmentSize += tmpsize
				slog.Warn("Attachment", "Fileanem", filename, "contentType", contentType)
			}

		default:
			slog.Error("Unknown header type")
		}

		// Validate the email body size
		if err := ValidateEmail.ValidateBodySize(); err != nil {
			return err
		}
		// Validate the email attachment size
		if err := ValidateEmail.ValidateAttachments(); err != nil {
			return err
		}
		// Validate the email embedded content size
		if err := ValidateEmail.ValidateEmbeddedContent(); err != nil {
			return err
		}

	}
	return nil
}
