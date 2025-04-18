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
	err := SaveMail(data)
	if err != nil {
		return err
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

	fmt.Println("From:", from)
	fmt.Println("To:", to)
	fmt.Println("Header To:", toList)
	fmt.Println("Header Cc:", ccList)
	fmt.Println("Subject:", subject)

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
			cid := strings.Trim(h.Get("Content-Id"), "<>")
			if err != nil || filename == "" {
				filename = cid
			}
			fmt.Println("Attachment:", filename)
		default:
			slog.Info("Unknown header type")
		}

	}
	return nil
}

func main() {
	fmt.Println("Starting SMTP server on :2525")
	err := smtpd.ListenAndServe(":2525", mailHandler, "MyServerApp", "mail.example.com")
	if err != nil {
		slog.Error(err.Error())
	}
}
