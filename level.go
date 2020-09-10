package log

import "strings"

// Level 代表日志级别，值越小日志级别越高。
type Level int

// 各种日志级别。
const (
	LogFatal Level = 1 << iota
	LogError
	LogWarn
	LogTrace
	LogInfo
	LogDebug

	logMax   = LogDebug + 1
	logPrint = 0
)

func parseLevel(level string) Level {
	switch strings.ToLower(level) {
	case "debug":
		return LogDebug
	case "info":
		return LogInfo
	case "trace":
		return LogTrace
	case "warn", "warning":
		return LogWarn
	case "error":
		return LogError
	case "fatal":
		return LogFatal
	default:
		return LogDebug
	}
}
