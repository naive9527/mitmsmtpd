package utils

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"sync"

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

type EmailServerItem struct {
	Server         string `yaml:"server"`
	Port           int    `yaml:"port"`
	AuthMechanisms string `yaml:"authMechanisms"`
}

type Config struct {
	SmptdServer struct {
		Address  string `yaml:"address"`  // Service listening address
		Debug    bool   `yaml:"debug"`    // Enable debug mode
		Appname  string `yaml:"appname"`  // Server application name
		Hostname string `yaml:"hostname"` // Server hostname (empty for auto-detection)
	} `yaml:"smptdServer"`

	SmtpProbe struct {
		Enable        bool `yaml:"enable"`        // 是否开启探测
		RetryInterval int  `yaml:"retryInterval"` // 探测失败重试间隔，单位：秒
		MaxRetry      int  `yaml:"maxRetry"`      // 最大重试次数
	} `yaml:"smtpProbe"`

	SmtpdAuth struct {
		Mechanisms   map[string]bool `yaml:"mechanisms"`   // Supported authentication mechanisms
		Required     bool            `yaml:"required"`     // Authentication required
		AllowAnyAuth bool            `yaml:"allowAnyAuth"` // Authentication required
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

	UserDB      map[string]string          `yaml:"userDB"` // User database (username/password pairs)
	EmailServer map[string]EmailServerItem `yaml:"emailServer"`

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

	Notification struct {
		// Other  *NotificationOtherStruct `yaml:"other"`
		Email *NotificationEmailStruct `yaml:"email"`
	} `yaml:"notification"`
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

func init() {
	InitConfig()
	MailInfoCacheIns = NewMailInfoCache()
}

// It is used to store the username and password for client login, so as to forward the email after verification is passed.
type MailInfoCache struct {
	mu       sync.RWMutex
	UserInfo map[string]string
}

func NewMailInfoCache() *MailInfoCache {
	return &MailInfoCache{
		UserInfo: make(map[string]string),
	}
}

func (mailInfo *MailInfoCache) GetUserPass(username string) (string, error) {
	mailInfo.mu.RLock()
	defer mailInfo.mu.RUnlock()

	if passwd, ok := mailInfo.UserInfo[username]; ok {
		return passwd, nil
	}
	info := fmt.Sprintf("user %s password cannot be obtained", username)
	slog.Error(info)
	return "", errors.New(info)
}

func (mailInfo *MailInfoCache) SetUserPass(username, passwd string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			info := fmt.Sprintf("panic in SetUserPass: %s %v", username, r)
			slog.Error(info)
			err = errors.New(info)
		}
	}()

	if curPasswd, err := mailInfo.GetUserPass(username); err != nil || curPasswd != passwd {
		mailInfo.mu.Lock()
		defer mailInfo.mu.Unlock()
		mailInfo.UserInfo[username] = passwd
	}
	return nil
}
