package main

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"

	"github.com/emersion/go-message"
	gomail "github.com/emersion/go-message/mail" // Avoid conflicts with net/mail
	"github.com/naive9527/mitmsmtpd/smtpd"       // It is actually a modified version of https://github.com/mhale/smtpd.
)

func mailHandler(origin net.Addr, from string, to []string, data []byte) error {
	// err := SaveMail(data)
	// if err != nil {
	// 	return err
	// }

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

	fmt.Printf("Received an email From: %s To: %s. email header To: %s. email header Cc: %s. Subject is: %s", from, strings.Join(to, "; "), toList, ccList, subject)

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
			fmt.Println("Body:", string(b))
		case *gomail.AttachmentHeader:
			// This is an attachment (Including the attached files and the pictures in the document)
			filename, err := h.Filename()
			contentType := p.Header.Get("Content-Type")
			if err != nil || filename == "" {
				cid := strings.Trim(h.Get("Content-Id"), "<>")
				filename = cid
				fmt.Printf("The file embedded in the email body:%s type %s\n", filename, contentType)
			} else {
				fmt.Printf("Attachment:%s type %s\n", filename, contentType)
			}

		default:
			slog.Info("Unknown header type")
		}

	}
	return nil
}

func main() {
	server := ":2525"
	certFile := "/opt/mygo/mitmsmtpd/tls/mail.pem"
	keyFile := "/opt/mygo/mitmsmtpd/tls/mail-key.pem"
	fmt.Printf("Starting SMTP server on %s\n", server)
	// err := smtpd.ListenAndServe(":2525", mailHandler, "MyServerApp", "mail.example.com")
	// err := smtpd.ListenAndServe(":2525", mailHandler, "MyServerApp", "")
	err := smtpd.ListenAndServeTLS(server, certFile, keyFile, mailHandler, "MyServerApp", "")
	if err != nil {
		slog.Error(err.Error())
	}
}
