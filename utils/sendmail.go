package utils

import (
	"errors"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
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

// func main() {
// 	mailFile := "golang__10415814528984713763.eml"
// 	mechanisms := "LOGIN" // "PLAIN LOGIN CRAM-MD5"
// 	from := "it-report@x-epic.com"
// 	password := "xxx"
// 	smtpServer := "smtp.office365.com"
// 	smtpPort := 587
// 	to := []string{"kevinwu@x-epic.com"}
// 	cc := []string{"kevinwutest@x-epic.com"}
// 	recipient := append(append([]string{}, to...), cc...)

// 	data, err := os.ReadFile(mailFile)
// 	if err != nil {
// 		fmt.Printf("读取文件失败: %v\n", err)
// 		return
// 	}

// 	err = SendMailExt(smtpServer, smtpPort, mechanisms, from, password, recipient, data)
// 	if err != nil {
// 		fmt.Printf("邮件发送失败: %v\n", err)
// 		return
// 	}
// 	fmt.Println("邮件发送成功")
// }
