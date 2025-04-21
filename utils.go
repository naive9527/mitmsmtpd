package main

import (
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

// User database (stored in memory; in production environment, it should be replaced by a database)
var userDB = map[string]string{
	"user01@example.com": "123456",
}

func authHandler(remoteAddr net.Addr, mechanism string, username []byte, password []byte, shared []byte) (bool, error) {
	value, ok := AuthMechs[mechanism]
	if !(ok && value == true) {
		slog.Warn(fmt.Sprintf("Unsupported authentication method %s", mechanism))
		return false, nil
	}
	user := string(username)
	pass := string(password)

	// 验证用户名和密码
	if storedPass, ok := userDB[user]; ok && storedPass == pass {
		slog.Info(fmt.Sprintf("Authentication successful method %s", mechanism), "Username", user)
		return true, nil
	}
	slog.Error(fmt.Sprintf("Authentication failed method %s", mechanism), "Username", user)
	return false, nil
}

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

	slog.Info("Received an email", "From", from, "To", strings.Join(to, "; "), "email header To", toList, "email header Cc", ccList, "Subject", subject)

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
			slog.Info(fmt.Sprintf("Body: %s", string(b)))

		case *gomail.AttachmentHeader:
			// This is an attachment (Including the attached files and the pictures in the document)
			filename, err := h.Filename()
			contentType := p.Header.Get("Content-Type")
			if err != nil || filename == "" {
				cid := strings.Trim(h.Get("Content-Id"), "<>")
				filename = cid
				slog.Warn("The file embedded in the email body", "Fileanem", filename, "contentType", contentType)
			} else {
				slog.Warn("Attachment", "Fileanem", filename, "contentType", contentType)
			}

		default:
			slog.Error("Unknown header type")
		}

	}
	return nil
}

func Xlog(logPath string, logName string) *slog.Logger {
	file, _ := os.OpenFile(filepath.Join(logPath, logName), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	// defer file.Close()

	// 创建组合输出流（文件 + 控制台）
	multiWriter := io.MultiWriter(file, os.Stdout)

	// 初始化slog
	logger := slog.New(slog.NewJSONHandler(multiWriter, nil))
	slog.SetDefault(logger)

	// 记录日志（同时输出到文件和控制台）
	// for i := 0; i < 10; i++ {
	//      // time.Sleep(time.Second * 2)
	//      slog.Info("日志轮转测试", "count", i)
	// }

	return logger
}
