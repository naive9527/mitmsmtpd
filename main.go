package main

import (
	"fmt"
	"log/slog"

	// Avoid conflicts with net/mail
	"github.com/naive9527/mitmsmtpd/smtpd" // It is actually a modified version of https://github.com/mhale/smtpd.
)

var AuthMechs = map[string]bool{"PLAIN": true, "LOGIN": true}

func ListenAndServeTLSAuth(addr string, certFile string, keyFile string, handler smtpd.Handler, appname string, hostname string, authHandler smtpd.AuthHandler) error {
	srv := &smtpd.Server{Addr: addr, Handler: handler, Appname: appname, Hostname: hostname, AuthHandler: authHandler, AuthRequired: true,
		AuthMechs: AuthMechs}
	err := srv.ConfigureTLS(certFile, keyFile)
	if err != nil {
		return err
	}
	return srv.ListenAndServe()
}

func main() {
	server := ":2525"
	certFile := "/opt/mygo/mitmsmtpd/tls/mail.pem"
	keyFile := "/opt/mygo/mitmsmtpd/tls/mail-key.pem"
	fmt.Printf("Starting SMTP server on %s\n", server)
	// err := smtpd.ListenAndServe(":2525", mailHandler, "MyServerApp", "mail.example.com")
	// err := smtpd.ListenAndServe(":2525", mailHandler, "MyServerApp", "")
	// err := smtpd.ListenAndServeTLS(server, certFile, keyFile, mailHandler, "MyServerApp", "")
	err := ListenAndServeTLSAuth(server, certFile, keyFile, mailHandler, "MyServerApp", "", authHandler)
	if err != nil {
		slog.Error(err.Error())
	}

}
