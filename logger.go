package log

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	logTimeFormat   = "2006-01-02T15:04:05.999Z07:00"
	maxLogLine      = 4096
	loggerSkipLevel = 3

	replaceStdPackagePrefix = "<std>"
)

var (
	stdPackagePrefix string
	fakeNow          time.Time
	logSeparator     = []byte("||")
)

func init() {
	t := reflect.TypeOf(Config{})
	stdPackagePrefix = t.PkgPath()

	if idx := strings.LastIndex(stdPackagePrefix, "/"); idx >= 0 {
		stdPackagePrefix = stdPackagePrefix[:idx]
	}
}

// Logger 代表一个标准日志接口。
type Logger interface {
	io.Closer

	// Debugf 输出调试日志，默认情况日志级别下不会输出，通过修改配置中的 LogLevel，将级别设置为 LogDebug 来显示这个级别的日志。
	Debugf(ctx context.Context, fmt string, args ...interface{})

	// Infof 输出普通日志，通常的业务日志多数都为这种格式。
	Infof(ctx context.Context, fmt string, args ...interface{})

	// Tracef 输出跟踪日志，一般框架使用，用于输出一些可以在日志采集中使用的结构化日志。
	Tracef(ctx context.Context, fmt string, args ...interface{})

	// Warnf 输出告警日志，如果程序走到了一些不预期的分支，需要人工关注，应该用这个级别。
	Warnf(ctx context.Context, fmt string, args ...interface{})

	// Errorf 输出错误日志，如果程序发生了严重错误，应该用这个级别。
	Errorf(ctx context.Context, fmt string, args ...interface{})

	// Fatalf 直接终止程序，在业务中几乎用不到这种日志，一般只在程序启动的时候用作快速返回。
	Fatalf(ctx context.Context, fmt string, args ...interface{})

	// Printf 可以无视日志级别，始终对外输出日志，一般只用于框架，业务不使用。
	Printf(ctx context.Context, fmt string, args ...interface{})
}

type logger struct {
	maxLevel   Level
	errorLevel Level
	pkgPrefix  string

	allLogger io.Writer
	wfLogger  io.Writer

	files   []*lumberjack.Logger
	writers []*AsyncWriter

	pcCache sync.Map
}

type stack struct {
	line         []byte
	panicContext string
}

const (
	maxLogFileSize = 1 << 32 // 4GB
)

// newLogger 创建一个新的日志实例。
// 如果 config 不为空，日志写入到指定文件，否则写入到 stdout/stderr。
func newLogger(config *Config) *logger {
	var allLogger io.WriteCloser = dummyCloser{Writer: os.Stdout}
	var wfLogger io.WriteCloser = dummyCloser{Writer: os.Stderr}

	if config == nil {
		return &logger{
			maxLevel:  logMax,
			allLogger: allLogger,
			wfLogger:  wfLogger,
		}
	}

	logPath := config.LogPath
	logLevelString := config.LogLevel
	errorLogPath := config.ErrorLogPath
	errorLogLevelString := config.ErrorLogLevel
	bufferedLines := config.BufferedLines
	pkgPrefix := config.PackagePrefix

	if logPath == "" {
		logPath = DefaultLogPath
	}

	if logLevelString == "" {
		logLevelString = DefaultLogLevel
	}

	if errorLogPath == "" {
		errorLogPath = DefaultErrorLogPath
	}

	if errorLogLevelString == "" {
		errorLogLevelString = DefaultErrorLogLevel
	}

	if bufferedLines <= 0 {
		bufferedLines = DefaultBufferedLines
	}

	if pkgPrefix != "" {
		idx := strings.LastIndex(pkgPrefix, "/")

		if idx >= 0 {
			pkgPrefix = pkgPrefix[:idx+1]
		}
	}

	var files []*lumberjack.Logger
	var writers []*AsyncWriter

	allFile := &lumberjack.Logger{
		Filename: logPath,
		MaxSize:  maxLogFileSize,
	}
	files = append(files, allFile)
	w := NewAsyncWriter(allFile, bufferedLines)
	writers = append(writers, w)
	allLogger = w

	if errorLogPath != logPath {
		wfFile := &lumberjack.Logger{
			Filename: errorLogPath,
			MaxSize:  maxLogFileSize,
		}
		files = append(files, wfFile)
		w := NewAsyncWriter(wfFile, bufferedLines)
		writers = append(writers, w)
		wfLogger = w
	} else {
		wfLogger = allLogger
	}

	return &logger{
		maxLevel:   parseLevel(logLevelString),
		errorLevel: parseLevel(errorLogLevelString),
		pkgPrefix:  pkgPrefix,

		allLogger: allLogger,
		wfLogger:  wfLogger,

		files:   files,
		writers: writers,
	}
}

func (l *logger) Debugf(ctx context.Context, fmt string, args ...interface{}) {
	l.log(ctx, LogDebug, fmt, args...)
}

func (l *logger) Infof(ctx context.Context, fmt string, args ...interface{}) {
	l.log(ctx, LogInfo, fmt, args...)
}

func (l *logger) Tracef(ctx context.Context, fmt string, args ...interface{}) {
	l.log(ctx, LogTrace, fmt, args...)
}

func (l *logger) Warnf(ctx context.Context, fmt string, args ...interface{}) {
	l.log(ctx, LogWarn, fmt, args...)
}

func (l *logger) Errorf(ctx context.Context, fmt string, args ...interface{}) {
	l.log(ctx, LogError, fmt, args...)
}

func (l *logger) Fatalf(ctx context.Context, fmt string, args ...interface{}) {
	l.log(ctx, LogFatal, fmt, args...)
}

func (l *logger) Printf(ctx context.Context, fmt string, args ...interface{}) {
	l.log(ctx, logPrint, fmt, args...)
}

func (l *logger) log(ctx context.Context, level Level, format string, args ...interface{}) {
	if l.maxLevel < level {
		return
	}

	panicContext := ""

	// 日志格式：
	//     [INFO] 2019-07-03T12:34:56.789Z08:00 *||key1=value1||this is custom log text
	buf := &bytes.Buffer{}

	if level != logPrint {
		// 输出 `[level]`
		levelName := "UNKNOWN"

		switch level {
		case LogDebug:
			levelName = "DEBUG"
		case LogInfo:
			levelName = "INFO"
		case LogTrace:
			levelName = "TRACE"
		case LogWarn:
			levelName = "WARN"
		case LogError:
			levelName = "ERROR"
		case LogFatal:
			levelName = "FATAL"
		}

		buf.WriteByte('[')
		buf.WriteString(levelName)
		buf.WriteByte(']')

		// 输出时间戳。
		now := time.Now()

		if !fakeNow.IsZero() {
			now = fakeNow
		}

		buf.WriteByte('[')
		buf.WriteString(now.Format(logTimeFormat))
		buf.WriteByte(']')

		// 输出调用栈。
		if pc, _, _, ok := runtime.Caller(loggerSkipLevel); ok {
			var st stack

			if cache, ok := l.pcCache.Load(pc); ok {
				st = cache.(stack)
			} else {
				st = l.parsePC(pc)
				l.pcCache.Store(pc, st)
			}

			buf.Write(st.line)
			panicContext = st.panicContext
		}

		// 输出 tag。
		tag := tag(ctx)

		if tag == "" {
			tag = "*"
		}

		buf.WriteByte(' ')
		buf.WriteString(tag)

		// 准备开始输出用户日志。
		buf.Write(logSeparator)

		// 输出 ctx 中的各种信息。
		more := findMoreInfo(ctx)

		for _, info := range more {
			fmt.Fprintf(buf, "%s=%v", info.Key, info.Value)
			buf.Write(logSeparator)
		}
	}

	fmt.Fprintf(buf, format, args...)

	buf.WriteByte('\n')
	line := buf.Bytes()

	if len(line) > maxLogLine {
		line = line[:maxLogLine]
	}

	if level > l.errorLevel || level == logPrint {
		l.allLogger.Write(line)

		if isStdoutTerminal {
			os.Stdout.Write(line)
		}
	} else {
		l.allLogger.Write(line)

		if l.wfLogger != l.allLogger {
			l.wfLogger.Write(line)
		}

		if isStderrTerminal {
			os.Stderr.Write(line)
		}
	}

	if level == LogFatal {
		l.Flush()
		panic(panicContext)
	}
}

func (l *logger) parsePC(pc uintptr) stack {
	f := runtime.FuncForPC(pc)
	file, line := f.FileLine(pc)
	file = path.Base(file)
	name := f.Name()
	prefix := ""

	// 简化日志中的 package 路径，避免输出过多无用信息。
	if l.pkgPrefix != "" && strings.HasPrefix(name, l.pkgPrefix) {
		name = name[len(l.pkgPrefix):]
	} else if stdPackagePrefix != "" && strings.HasPrefix(name, stdPackagePrefix) {
		name = name[len(stdPackagePrefix):]
		prefix = replaceStdPackagePrefix
	}

	lineBuf := &bytes.Buffer{}
	lineBuf.WriteByte('[')
	lineBuf.WriteString(file)
	lineBuf.WriteByte(':')
	lineBuf.WriteString(strconv.Itoa(line))
	lineBuf.WriteByte('@')

	if prefix != "" {
		lineBuf.WriteString(prefix)
	}

	lineBuf.WriteString(name)
	lineBuf.WriteByte(']')

	panicContext := fmt.Sprintf("go-log: log.Fatalf at %v:%v@%v%v", file, line, prefix, name)

	return stack{
		line:         lineBuf.Bytes(),
		panicContext: panicContext,
	}
}

// Rotate 重新打开所有的日志文件，方便做日志切割。
func (l *logger) Rotate() (err error) {
	for _, f := range l.files {
		if e := f.Rotate(); e != nil {
			err = e
		}
	}

	return
}

// Flush 将所有缓冲区的内容强制写入磁盘。
func (l *logger) Flush() (err error) {
	// 先确保当前缓冲区的数据写入了内部文件。
	for _, w := range l.writers {
		if e := w.Flush(); e != nil {
			err = e
		}
	}

	return
}

// Close 关闭所有日志并且确保所有日志可以落盘。
func (l *logger) Close() (err error) {
	for _, w := range l.writers {
		if e := w.Close(); e != nil {
			err = e
		}
	}

	return
}

type dummyCloser struct {
	io.Writer
}

func (dummyCloser) Close() error {
	return nil
}
