package utils

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

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

func GetAvailableSMTPIP(host string, port, retryInterval, maxRetry int) (string, error) {
	var err error
	var ips, ipv4Addresses []net.IP

	for attempt := 0; attempt < maxRetry; attempt++ {
		// DNS解析获取所有A记录
		ips, err = net.LookupIP(host)
		if err != nil {
			slog.Error(fmt.Sprintf("attempt %d , DNS lookup failed for host %s: %v", attempt, host, err))
			continue
		}

		for _, ip := range ips {
			if ipv4 := ip.To4(); ipv4 != nil {
				ipv4Addresses = append(ipv4Addresses, ipv4)
			}
		}

		if len(ipv4Addresses) == 0 {
			err = fmt.Errorf("no IPv4 addresses found for host %s", host)
			slog.Error(fmt.Sprintf("attempt %d ,no IPv4 addresses found for host %s", attempt, host))
			continue
		}

		// 尝试连接探测
		for _, ip := range ipv4Addresses {
			address := net.JoinHostPort(ip.String(), strconv.Itoa(port))

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", address)
			if err == nil {
				conn.Close()
				return ip.String(), nil
			} else {
				slog.Error(fmt.Sprintf("attempt %d , connection host %s failed for IP %s: %v", attempt, host, ip.String(), err))
			}
		}

		// 如果本轮所有IP都失败，则等待重试间隔
		err = fmt.Errorf("all IP addresses connect failed for host %s", host)
		slog.Error(fmt.Sprintf("attempt %d , all IP addresses connect failed for host %s", attempt, host))
		time.Sleep(time.Duration(retryInterval) * time.Second)
	}

	return "", err
}

// usage:
// auth := LoginAuth("loginname", "password")
// err := smtp.SendMail(smtpServer + ":25", auth, fromAddress, toAddresses, []byte(message))
// or
// client, err := smtp.Dial(smtpServer)
// client.Auth(LoginAuth("loginname", "password"))

func SendMailExt(smtpServer string, smtpPort int, mechanisms, from, password string, to []string, data []byte) error {
	var err error
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

	if CFG.SmtpProbe.Enable {
		ip, err := GetAvailableSMTPIP(smtpServer, smtpPort, CFG.SmtpProbe.RetryInterval, CFG.SmtpProbe.MaxRetry)
		if err != nil {
			info := fmt.Sprintf("the email can not sent out, because the SMTP server %s:%d is not available: %s", smtpServer, smtpPort, err.Error())
			slog.Error(info)
			return errors.New(info)
		}
		err = SendMailByIP(ip, smtpPort, smtpServer, auth, from, to, data)
	} else {
		err = smtp.SendMail(fmt.Sprintf("%s:%d", smtpServer, smtpPort), auth, from, to, data)
	}

	if err != nil {
		info := fmt.Sprintf("%s the email sent out error %s", smtpServer, err.Error())
		slog.Error(info)
		return errors.New(info)
	}
	slog.Info(fmt.Sprintf("%s the email sent out success", smtpServer))
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

// 将smtp.SendMail的代码复制后，进行改写，因为直接使用ip地址发送邮件时，会证书验证失败
// SendMailByIP 通过 IP 连接 SMTP，但证书校验用 domain
func SendMailByIP(ip string, port int, domain string, a smtp.Auth, from string, to []string, msg []byte) error {
	addr := net.JoinHostPort(ip, strconv.Itoa(port))
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()
	if err = c.Hello("localhost"); err != nil {
		return err
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		config := &tls.Config{ServerName: domain}
		if err = c.StartTLS(config); err != nil {
			return err
		}
	}
	if a != nil && c.Extension != nil {
		if ok, _ := c.Extension("AUTH"); !ok {
			return errors.New("smtp: server doesn't support AUTH")
		}
		if err = c.Auth(a); err != nil {
			return err
		}
	}
	if err = c.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(msg)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	return c.Quit()
}

func GenMailContent(content, clientip, from, emailFile string) string {
	htmlBody := `
	<div style="margin: 10px auto 10px 10px;">
		<p> Mail Gateway notification</p>
		<table border="2"  cellspacing="0" cellpadding="6" bordercolor="dimgray" style="min-width: 800px">
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
	</div>
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
