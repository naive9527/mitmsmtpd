package main

import (
	"fmt"
	"hash/fnv"
	"log/slog"
	"net"
	"os"
	"time"
)

func SaveMail(data []byte) error {
	timestamp := time.Now().Format("20060102_150405")
	subjectHash := hashSubject(data)
	filename := fmt.Sprintf("%s_%d.eml", timestamp, subjectHash)
	return os.WriteFile(filename, data, 0644)
}

func hashSubject(data []byte) uint64 {
	h := fnv.New64a()
	h.Write(data)
	return h.Sum64()
}

// User database (stored in memory; in production environment, it should be replaced by a database)
var userDB = map[string]string{
	"user01@example.com": "securepassword123",
}

// 认证回调函数
func authHandler(remoteAddr net.Addr, mechanism string, username []byte, password []byte, shared []byte) (bool, error) {
	user := string(username)
	pass := string(password)

	// 验证用户名和密码
	if storedPass, ok := userDB[user]; ok && storedPass == pass {
		slog.Info("Authentication successful", "Username", user)
		return true, nil
	}
	slog.Warn("Authentication failed", "Username", user)
	return false, nil
}
