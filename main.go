package main

import (
	"fmt"
	"log/slog"

	// Avoid conflicts with net/mail
	"github.com/naive9527/mitmsmtpd/smtpd" // It is actually a modified version of https://github.com/mhale/smtpd.
	"github.com/naive9527/mitmsmtpd/utils"
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
	utils.Xlog(utils.CFG.Logging.Path, utils.CFG.Logging.Filename)
	var err error

	smtpd.Debug = utils.CFG.SmptdServer.Debug
	server := utils.CFG.SmptdServer.Address
	certFile := utils.CFG.SmtpdTLS.Cert
	keyFile := utils.CFG.SmtpdTLS.Key
	appName := utils.CFG.SmptdServer.Appname
	hostname := utils.CFG.SmptdServer.Hostname

	slog.Info(fmt.Sprintf("Starting SMTP server on server %s", server))
	if utils.CFG.SmtpdAuth.Required && utils.CFG.SmtpdTLS.TLSEnabled {
		err = ListenAndServeTLSAuth(server, certFile, keyFile, utils.MailHandler, appName, hostname, utils.AuthHandler, utils.CFG.SmtpdAuth.Mechanisms)
	} else if !utils.CFG.SmtpdAuth.Required && utils.CFG.SmtpdTLS.TLSEnabled {
		err = smtpd.ListenAndServeTLS(server, certFile, keyFile, utils.MailHandler, appName, hostname)
	} else if !utils.CFG.SmtpdAuth.Required && !utils.CFG.SmtpdTLS.TLSEnabled {
		err = smtpd.ListenAndServe(server, utils.MailHandler, appName, hostname)
	} else {
		slog.Error("Invalid configuration")
	}

	if err != nil {
		slog.Error(err.Error())
	}
}
