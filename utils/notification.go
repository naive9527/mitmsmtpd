package utils

import (
	"fmt"
)

type Notifier interface {
	GenContent(content, clientip, from, emailFilename string) string
	Send(content string) error
}

func TriggerErrNotification(content, clientip, from string, to []string, data []byte, notifier Notifier) error {
	emailFile, err := SaveMail(data)
	if err != nil {
		content = fmt.Sprintf("%s\n%s", content, err.Error())
	}

	newContent := notifier.GenContent(content, clientip, from, emailFile)
	return notifier.Send(newContent)
}

type NotificationEmailStruct struct {
	Enabled  bool     `yaml:"enabled"`
	From     string   `yaml:"from"`
	Password string   `yaml:"password"`
	Server   string   `yaml:"server"`
	Port     int      `yaml:"port"`
	To       []string `yaml:"to"`
	Cc       []string `yaml:"cc"`
	Subject  string   `yaml:"subject"`
}

func (msgsender *NotificationEmailStruct) GenContent(content, clientip, from, emailFile string) string {
	return GenMailContent(content, clientip, from, emailFile)
}

func (msgsender *NotificationEmailStruct) Send(content string) error {
	return SendMailMsg(msgsender.Server, msgsender.Port, msgsender.From, msgsender.Password, msgsender.To, msgsender.Cc, msgsender.Subject, content)
}
