package utils

import (
	"errors"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"

	"gopkg.in/gomail.v2"
)

type loginAuth struct {
	username, password string
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New("unkown fromServer")
		}
	}
	return nil, nil
}

// usage:
// auth := LoginAuth("loginname", "password")
// err := smtp.SendMail(smtpServer + ":25", auth, fromAddress, toAddresses, []byte(message))
// or
// client, err := smtp.Dial(smtpServer)
// client.Auth(LoginAuth("loginname", "password"))

func SendMailExt(smtpServer string, smtpPort int, mechanisms, from, password string, to []string, data []byte) error {
	var auth smtp.Auth
	switch {
	case strings.Contains(mechanisms, "CRAM-MD5"):
		auth = smtp.CRAMMD5Auth(from, password)
	case strings.Contains(mechanisms, "PLAIN"):
		auth = smtp.PlainAuth("", from, password, smtpServer)
	case strings.Contains(mechanisms, "LOGIN"):
		auth = LoginAuth(from, password)
	default:
		info := fmt.Sprintf("unsupported authentication type: %s,  the email can not sent out", mechanisms)
		slog.Error(info)
		return errors.New(info)
	}
	err := smtp.SendMail(fmt.Sprintf("%s:%d", smtpServer, smtpPort), auth, from, to, data)
	if err != nil {
		info := fmt.Sprintf("the email sent out error %s", err.Error())
		slog.Error(info)
		return errors.New(info)
	}
	slog.Info("the email sent out success")
	return nil
}

func SendMailData(from string, to []string, data []byte) error {
	smtpDomain := strings.Split(from, "@")[1]
	smtpServerItem, ok := CFG.EmailServer[smtpDomain]
	if !ok {
		info := fmt.Sprintf("The email server is not configured for %s", from)
		slog.Error(info)
		return errors.New(info)
	}

	password, err := MailInfoCacheIns.GetUserPass(from)
	if err != nil {
		return err
	}

	return SendMailExt(
		smtpServerItem.Server,
		smtpServerItem.Port,
		smtpServerItem.AuthMechanisms,
		from,
		password,
		to,
		data)
}

func GenMailContent(content, clientip, from, emailFile string) string {
	htmlBody := `
	<p> Mail Gateway notification</p>
	<table border="1">  
	<tr>  
		<td>ClientIP</td><td>%s</td>  
	</tr>  
 	<tr>  
		<td>From</td><td>%s</td>  
	</tr>  
	<tr>  
		<td>Error</td><td>%s</td>  
	</tr>  
	<tr>  
		<td>Saved Email File</td><td>%s</td>  
	</tr>  	
	</table>  
	`
	return fmt.Sprintf(htmlBody, clientip, from, content, emailFile)
}

func SendMailMsg(smtpServer string, smtpPort int, from, password string, to, cc []string, subject, content string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", to...)
	m.SetHeader("Cc", cc...)
	m.SetHeader("Subject", subject)

	m.SetBody("text/html", content)

	d := gomail.NewDialer(smtpServer, smtpPort, from, password)

	if err := d.DialAndSend(m); err != nil {
		info := fmt.Sprintf("the notification email sent out error %s, From: %s, To: %v, Cc: %v", err.Error(), from, to, cc)
		slog.Error(info)
		slog.Error(fmt.Sprintf("the failed send notification email content:\n %s", content))
		return errors.New(info)
	}
	info := fmt.Sprintf("the notification email sent out success, From: %s, To: %v, Cc: %v", from, to, cc)
	slog.Info(info)
	return nil
}
