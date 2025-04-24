package utils

import (
	"log/slog"
	"os"
	"regexp"

	"github.com/spf13/viper"
)

var CFG Config // Global configuration instance

type User struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}
type AttachmentRule struct {
	Allowed bool `mapstructure:"allowed"`
	MaxSize int  `mapstructure:"maxSize"`
}
type Config struct {
	SmptdServer struct {
		Address  string `mapstructure:"address"`  // Service listening address
		Debug    bool   `mapstructure:"debug"`    // Enable debug mode
		Appname  string `mapstructure:"appname"`  // Server application name
		Hostname string `mapstructure:"hostname"` // Server hostname (empty for auto-detection)
	} `mapstructure:"smptdServer"`

	SmtpdAuth struct {
		Mechanisms map[string]bool `mapstructure:"mechanisms"` // Supported authentication mechanisms
		Required   bool            `mapstructure:"required"`   // Authentication required
	} `mapstructure:"smtpdAuth"`

	SmtpdTLS struct {
		TLSEnabled bool   `mapstructure:"enabled"` // Enable TLS
		Cert       string `mapstructure:"cert"`    // Path to TLS certificate
		Key        string `mapstructure:"key"`     // Path to TLS private key
	} `mapstructure:"smtpdTLS"`

	Logging struct {
		Path     string `mapstructure:"path"`     // Log directory
		Filename string `mapstructure:"filename"` // Log filename
	} `mapstructure:"logging"`

	UserDB    []User            `mapstructure:"userDB"` // User database (username/password pairs)
	UserDBMap map[string]string `mapstructure:"-"`      // The newly added mapping field (not involved in deserialization)

	VerificationRules struct {
		Sender          string         `mapstructure:"sender"`
		Recipient       string         `mapstructure:"recipient"`
		SenderIP        string         `mapstructure:"senderIP"`
		EmailBodySize   int            `mapstructure:"emailBodySize"`
		Attachment      AttachmentRule `mapstructure:"attachment"`
		EmbeddedContent AttachmentRule `mapstructure:"embeddedContent"`

		SenderRegexp    *regexp.Regexp `mapstructure:"-"`
		RecipientRegexp *regexp.Regexp `mapstructure:"-"`
		SenderIPRegexp  *regexp.Regexp `mapstructure:"-"`
	} `mapstructure:"verificationRules"`
}

func InitConfig() {
	v := viper.New()
	v.SetConfigName("config")    // Config file name (without extension)
	v.SetConfigType("yaml")      // Specify config format
	v.AddConfigPath(".")         // Config search path: current directory
	v.AddConfigPath("./configs") // Config search path: configs subdirectory

	// Allow environment variables override
	v.AutomaticEnv()

	// Set default values (optional)
	v.SetDefault("smtpdTLS.enabled", false)
	v.SetDefault("smtpdAuth.required", false)

	// Read configuration
	if err := v.ReadInConfig(); err != nil {
		slog.Error("Failed to read config", "error", err)
		os.Exit(1)
	}

	// Unmarshal to struct
	if err := v.Unmarshal(&CFG); err != nil {
		slog.Error("Failed to unmarshal config", "error", err)
		os.Exit(1)
	}

	// convert UserDB to UserDBMap
	CFG.UserDBMap = make(map[string]string)
	for _, user := range CFG.UserDB {
		// Check for duplicate usernames
		if _, exists := CFG.UserDBMap[user.Username]; exists {
			slog.Warn("Duplicate username detected", "username", user.Username)
		}
		CFG.UserDBMap[user.Username] = user.Password
	}

	CFG.VerificationRules.SenderRegexp, _ = regexp.Compile(CFG.VerificationRules.Sender)
	CFG.VerificationRules.RecipientRegexp, _ = regexp.Compile(CFG.VerificationRules.Recipient)
	CFG.VerificationRules.SenderIPRegexp, _ = regexp.Compile(CFG.VerificationRules.SenderIP)

}
