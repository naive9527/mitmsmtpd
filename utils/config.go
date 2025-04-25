package utils

import (
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

var CFG Config // Global configuration instance

type User struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}
type AttachmentRule struct {
	Allowed bool `yaml:"allowed"`
	MaxSize int  `yaml:"maxSize"`
}
type Config struct {
	SmptdServer struct {
		Address  string `yaml:"address"`  // Service listening address
		Debug    bool   `yaml:"debug"`    // Enable debug mode
		Appname  string `yaml:"appname"`  // Server application name
		Hostname string `yaml:"hostname"` // Server hostname (empty for auto-detection)
	} `yaml:"smptdServer"`

	SmtpdAuth struct {
		Mechanisms map[string]bool `yaml:"mechanisms"` // Supported authentication mechanisms
		Required   bool            `yaml:"required"`   // Authentication required
	} `yaml:"smtpdAuth"`

	SmtpdTLS struct {
		TLSEnabled bool   `yaml:"enabled"` // Enable TLS
		Cert       string `yaml:"cert"`    // Path to TLS certificate
		Key        string `yaml:"key"`     // Path to TLS private key
	} `yaml:"smtpdTLS"`

	Logging struct {
		Path     string `yaml:"path"`     // Log directory
		Filename string `yaml:"filename"` // Log filename
	} `yaml:"logging"`

	UserDB map[string]string `yaml:"userDB"` // User database (username/password pairs)

	VerificationRules struct {
		Sender          string         `yaml:"sender"`
		Recipient       string         `yaml:"recipient"`
		SenderIP        string         `yaml:"senderIP"`
		EmailBodySize   int            `yaml:"emailBodySize"`
		Attachment      AttachmentRule `yaml:"attachment"`
		EmbeddedContent AttachmentRule `yaml:"embeddedContent"`

		SenderRegexp    *regexp.Regexp `yaml:"-"`
		RecipientRegexp *regexp.Regexp `yaml:"-"`
		SenderIPRegexp  *regexp.Regexp `yaml:"-"`
	} `yaml:"verificationRules"`
}

func InitConfig() {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(data, &CFG)
	if err != nil {
		panic(err)
	}

	CFG.VerificationRules.SenderRegexp, _ = regexp.Compile(CFG.VerificationRules.Sender)
	CFG.VerificationRules.RecipientRegexp, _ = regexp.Compile(CFG.VerificationRules.Recipient)
	CFG.VerificationRules.SenderIPRegexp, _ = regexp.Compile(CFG.VerificationRules.SenderIP)

}
