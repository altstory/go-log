package log

const (
	// DefaultLogPath 日志文件的默认路径。
	DefaultLogPath = "./log/all.log"

	// DefaultLogLevel 是日志的默认级别。
	DefaultLogLevel = "info"

	// DefaultErrorLogPath 错误日志文件的默认路径。
	DefaultErrorLogPath = "./log/error.log"

	// DefaultErrorLogLevel 是错误日志的默认级别。
	DefaultErrorLogLevel = "warn"

	// DefaultBufferedLines 是内存中缓存的日志行数。
	DefaultBufferedLines = 1 << 18
)

// Config 代表日志配置。
type Config struct {
	LogPath       string `config:"log_path"`        // LogPath 是日志文件名，默认写到 DefaultLogPath 里面。
	LogLevel      string `config:"log_level"`       // LogLevel 是日志级别，默认是 DefaultLogLevel。
	ErrorLogPath  string `config:"error_log_path"`  // ErrorLogPath 是错误日志文件名，默认写到 DefaultErrorLogPath 里面。
	ErrorLogLevel string `config:"error_log_level"` // ErrorLogLevel 是错误日志级别，当错误级别不大于这个级别时写入错误日志，默认是 DefaultErrorLogLevel。

	PackagePrefix string `config:"package_prefix"` // PackagePrefix 设置最常用的 package 前缀，输出调用栈的时候会用 "." 代替这一长串字符，让日志看起来更简洁。
	BufferedLines int    `config:"buffered_lines"` // BufferedLines 设置最多在内存中缓存的日志行数，默认是 DefaultBufferedLines。
}
