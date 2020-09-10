package log

import (
	"context"
	"os"
	"sync/atomic"
	"unsafe"

	"golang.org/x/crypto/ssh/terminal"
)

var (
	defaultLoggerPtr = unsafe.Pointer(newLogger(nil))

	isStdoutTerminal bool
	isStderrTerminal bool
)

// Init 初始化日志配置。
func Init(config *Config) {
	isStdoutTerminal = terminal.IsTerminal(int(os.Stdout.Fd()))
	isStderrTerminal = terminal.IsTerminal(int(os.Stderr.Fd()))

	l := newLogger(config)
	old := (*logger)(atomic.SwapPointer(&defaultLoggerPtr, unsafe.Pointer(l)))

	if old != nil {
		old.Close()
	}
}

func defaultLogger() *logger {
	return (*logger)(atomic.LoadPointer(&defaultLoggerPtr))
}

// Debugf 输出调试日志，默认情况日志级别下不会输出，通过修改配置中的 LogLevel，将级别设置为 LogDebug 来显示这个级别的日志。
func Debugf(ctx context.Context, fmt string, args ...interface{}) {
	defaultLogger().Debugf(ctx, fmt, args...)
}

// Infof 输出普通日志，通常的业务日志多数都为这种格式。
func Infof(ctx context.Context, fmt string, args ...interface{}) {
	defaultLogger().Infof(ctx, fmt, args...)
}

// Tracef 输出跟踪日志，一般框架使用，用于输出一些可以在日志采集中使用的结构化日志。
func Tracef(ctx context.Context, fmt string, args ...interface{}) {
	defaultLogger().Tracef(ctx, fmt, args...)
}

// Warnf 输出告警日志，如果程序走到了一些不预期的分支，需要人工关注，应该用这个级别。
func Warnf(ctx context.Context, fmt string, args ...interface{}) {
	defaultLogger().Warnf(ctx, fmt, args...)
}

// Errorf 输出错误日志，如果程序发生了严重错误，应该用这个级别。
func Errorf(ctx context.Context, fmt string, args ...interface{}) {
	defaultLogger().Errorf(ctx, fmt, args...)
}

// Fatalf 直接终止程序，在业务中几乎用不到这种日志，一般只在程序启动的时候用作快速返回。
func Fatalf(ctx context.Context, fmt string, args ...interface{}) {
	defaultLogger().Fatalf(ctx, fmt, args...)
}

// Printf 可以无视日志级别，始终对外输出日志，一般只用于框架，业务不使用。
func Printf(ctx context.Context, fmt string, args ...interface{}) {
	defaultLogger().Printf(ctx, fmt, args...)
}

// Flush 将所有缓冲区的内容强制写入磁盘。
func Flush() error {
	return defaultLogger().Flush()
}

// Rotate 重新打开所有的日志文件，方便做日志切割。
func Rotate() error {
	return defaultLogger().Rotate()
}
