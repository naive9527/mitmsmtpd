package utils

import (
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
)

type Notifier interface {
	GenContent(content, clientip, from, emailFilename string) string
	Send(content string) error
}

// 检查字段是否启用（根据 yaml 配置中的 enabled 字段）
func fieldIsEnabled(fieldType reflect.StructField, field reflect.Value) bool {
	enabledField := field.Elem().FieldByName("Enabled") // 假设子结构体有 Enabled 字段
	return enabledField.IsValid() && enabledField.Bool()
}
func TriggerErrNotification(content, clientip, from string, to []string, data []byte) error {
	var senderror = strings.Builder{}
	emailFile, err := SaveMail(data)
	if err != nil {
		content = fmt.Sprintf("%s\n%s", content, err.Error())
	}

	v := reflect.ValueOf(&CFG.Notification).Elem() // 获取结构体的反射值（注意指针解引用）
	t := v.Type()                                  // 获取结构体类型信息
	// 遍历 Notification 结构体并执行 Send 方法
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)     // 获取字段的反射值
		fieldType := t.Field(i) // 获取字段的类型信息

		// 跳过未启用或空指针字段
		if field.IsNil() || !fieldIsEnabled(fieldType, field) {
			info := fmt.Sprintf("TriggerErrNotification: %s is not enabled", fieldType.Name)
			slog.Warn(info)
			continue
		}

		// 将字段转换为 Notifier 接口
		notifier, ok := field.Interface().(Notifier)
		if !ok {
			info := fmt.Sprintf("TriggerErrNotification: %s is not Notifier", fieldType.Name)
			slog.Error(info)
			continue // 未实现接口则跳过
		}

		// 调用 Send 方法
		newContent := notifier.GenContent(content, clientip, from, emailFile)
		err = notifier.Send(newContent)
		if err != nil {
			senderror.WriteString(fmt.Sprintf("TriggerErrNotification channel %s ,error: %s \n", fieldType.Name, err.Error()))
		}
	}
	if senderror.Len() > 0 {
		slog.Error(senderror.String())
		return errors.New(senderror.String())
	}
	return nil
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
