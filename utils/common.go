package utils

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
)

func Xlog(logPath string, logName string) *slog.Logger {
	file, _ := os.OpenFile(filepath.Join(logPath, logName), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	// defer file.Close()

	// 创建组合输出流（文件 + 控制台）
	multiWriter := io.MultiWriter(file, os.Stdout)

	// 初始化slog
	logger := slog.New(slog.NewJSONHandler(multiWriter, nil))
	slog.SetDefault(logger)

	// 记录日志（同时输出到文件和控制台）
	// for i := 0; i < 10; i++ {
	//      // time.Sleep(time.Second * 2)
	//      slog.Info("日志轮转测试", "count", i)
	// }

	return logger
}

func GetIPFromAddr(addr net.Addr) (string, error) {
	// Perform type assertion based on network protocol type
	switch addr := addr.(type) {
	case *net.TCPAddr:
		return addr.IP.String(), nil
	case *net.UDPAddr:
		return addr.IP.String(), nil
	case *net.IPAddr:
		return addr.IP.String(), nil
	default:
		// Handle non-IP address types (e.g. Unix domain sockets)
		if strings.Contains(addr.Network(), "unix") {
			return "", fmt.Errorf("non-IP address type: %s", addr.Network())
		}
		// Try parsing generic string format (fallback)
		host, _, err := net.SplitHostPort(addr.String())
		if err != nil {
			return "", fmt.Errorf("failed to parse address: %v", err)
		}
		return host, nil
	}
}

func CalculateReaderSize(r io.Reader) (int64, error) {
	total := int64(0)
	buf := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := r.Read(buf)
		total += int64(n)
		if err == io.EOF {
			return total, nil
		}
		if err != nil {
			return total, err
		}
	}
}
