package main

import (
	"fmt"
	"log/slog"

	// Avoid conflicts with net/mail
	"github.com/naive9527/mitmsmtpd/smtpd" // It is actually a modified version of https://github.com/mhale/smtpd.
)

func ListenAndServeTLSAuth(addr string, certFile string, keyFile string, handler smtpd.Handler, appname string, hostname string, authHandler smtpd.AuthHandler, authMechs map[string]bool) error {
	srv := &smtpd.Server{Addr: addr, Handler: handler, Appname: appname, Hostname: hostname, AuthHandler: authHandler, AuthRequired: true,
		AuthMechs: authMechs}
	err := srv.ConfigureTLS(certFile, keyFile)
	if err != nil {
		return err
	}
	return srv.ListenAndServe()
}

func main() {
	initConfig()
	Xlog(cfg.Logging.Path, cfg.Logging.Filename)
	var err error

	smtpd.Debug = cfg.SmptdServer.Debug
	server := cfg.SmptdServer.Address
	certFile := cfg.SmtpdTLS.Cert
	keyFile := cfg.SmtpdTLS.Key
	appName := cfg.SmptdServer.Appname
	hostname := cfg.SmptdServer.Hostname

	slog.Info(fmt.Sprintf("Starting SMTP server on server %s", server))
	if cfg.SmtpdAuth.Required && cfg.SmtpdTLS.TLSEnabled {
		err = ListenAndServeTLSAuth(server, certFile, keyFile, mailHandler, appName, hostname, authHandler, cfg.SmtpdAuth.Mechanisms)
	} else if !cfg.SmtpdAuth.Required && cfg.SmtpdTLS.TLSEnabled {
		err = smtpd.ListenAndServeTLS(server, certFile, keyFile, mailHandler, appName, hostname)
	} else if !cfg.SmtpdAuth.Required && !cfg.SmtpdTLS.TLSEnabled {
		err = smtpd.ListenAndServe(server, mailHandler, appName, hostname)
	} else {
		slog.Error("Invalid configuration")
	}

	if err != nil {
		slog.Error(err.Error())
	}
}
