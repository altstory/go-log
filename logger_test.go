package log

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"
)

func doLog(t *testing.T) {
	ctx := context.Background()
	Debugf(ctx, "debug 1+1=%v 2+2=%v", 2, 4)
	Infof(WithMoreInfo(ctx, Info{Key: "key1", Value: 123}, Info{Key: "key2", Value: "value2"}), "info 1+1=%v 2+2=%v", 2, 4)
	Tracef(WithTag(ctx, "trace"), "trace 1+1=%v 2+2=%v", 2, 4)
	Warnf(ctx, "warn 1+1=%v 2+2=%v", 2, 4)
	Errorf(ctx, "error 1+1=%v 2+2=%v", 2, 4)

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("Fatalf should trigger a panic.")
			}
		}()

		Fatalf(ctx, "debug 1+1=%v 2+2=%v", 2, 4)
	}()

	Printf(ctx, "print 1+1=%v 2+2=%v", 2, 4)
}

func TestLogger(t *testing.T) {
	now := "2019-07-03T12:34:56.789+08:00"
	fakeNow, _ = time.Parse(logTimeFormat, now)
	os.Remove(DefaultLogPath)
	os.Remove(DefaultErrorLogPath)

	defer func() {
		fakeNow = time.Time{}
	}()

	pkgPath := reflect.TypeOf(Config{}).PkgPath()

	// 普通的初始化一下。
	Init(&Config{
		LogLevel:      "debug",
		PackagePrefix: pkgPath,
	})
	doLog(t)

	// 测试默认是否关闭了 debug 级别日志。
	Init(&Config{})
	doLog(t)
	Flush()

	all, err := os.Open(DefaultLogPath)

	if err != nil {
		t.Fatalf("fail to read %v. [err:%v]", DefaultLogPath, err)
	}

	defer all.Close()
	expectedAllLogs := []byte(`[DEBUG][2019-07-03T12:34:56.789+08:00][logger_test.go:15@go-log.doLog] *||debug 1+1=2 2+2=4
[INFO][2019-07-03T12:34:56.789+08:00][logger_test.go:16@go-log.doLog] *||key1=123||key2=value2||info 1+1=2 2+2=4
[TRACE][2019-07-03T12:34:56.789+08:00][logger_test.go:17@go-log.doLog] trace||trace 1+1=2 2+2=4
[WARN][2019-07-03T12:34:56.789+08:00][logger_test.go:18@go-log.doLog] *||warn 1+1=2 2+2=4
[ERROR][2019-07-03T12:34:56.789+08:00][logger_test.go:19@go-log.doLog] *||error 1+1=2 2+2=4
[FATAL][2019-07-03T12:34:56.789+08:00][logger_test.go:28@go-log.doLog.func1] *||debug 1+1=2 2+2=4
print 1+1=2 2+2=4
[INFO][2019-07-03T12:34:56.789+08:00][logger_test.go:16@<std>/go-log.doLog] *||key1=123||key2=value2||info 1+1=2 2+2=4
[TRACE][2019-07-03T12:34:56.789+08:00][logger_test.go:17@<std>/go-log.doLog] trace||trace 1+1=2 2+2=4
[WARN][2019-07-03T12:34:56.789+08:00][logger_test.go:18@<std>/go-log.doLog] *||warn 1+1=2 2+2=4
[ERROR][2019-07-03T12:34:56.789+08:00][logger_test.go:19@<std>/go-log.doLog] *||error 1+1=2 2+2=4
[FATAL][2019-07-03T12:34:56.789+08:00][logger_test.go:28@<std>/go-log.doLog.func1] *||debug 1+1=2 2+2=4
print 1+1=2 2+2=4
`)
	actualAllLogs, err := ioutil.ReadAll(all)

	if bytes.Compare(expectedAllLogs, actualAllLogs) != 0 {
		t.Fatalf("invalid log content in %v.\n  expected:\n%v\n  actual:\n%v", DefaultLogPath, string(expectedAllLogs), string(actualAllLogs))
	}

	wf, err := os.Open(DefaultErrorLogPath)

	if err != nil {
		t.Fatalf("fail to read %v. [err:%v]", DefaultErrorLogPath, err)
	}

	defer wf.Close()
	expectedWFLogs := []byte(`[WARN][2019-07-03T12:34:56.789+08:00][logger_test.go:18@go-log.doLog] *||warn 1+1=2 2+2=4
[ERROR][2019-07-03T12:34:56.789+08:00][logger_test.go:19@go-log.doLog] *||error 1+1=2 2+2=4
[FATAL][2019-07-03T12:34:56.789+08:00][logger_test.go:28@go-log.doLog.func1] *||debug 1+1=2 2+2=4
[WARN][2019-07-03T12:34:56.789+08:00][logger_test.go:18@<std>/go-log.doLog] *||warn 1+1=2 2+2=4
[ERROR][2019-07-03T12:34:56.789+08:00][logger_test.go:19@<std>/go-log.doLog] *||error 1+1=2 2+2=4
[FATAL][2019-07-03T12:34:56.789+08:00][logger_test.go:28@<std>/go-log.doLog.func1] *||debug 1+1=2 2+2=4
`)
	actualWFLogs, err := ioutil.ReadAll(wf)

	if bytes.Compare(expectedWFLogs, actualWFLogs) != 0 {
		t.Fatalf("invalid log content in %v.\n  expected:\n%v\n  actual:\n%v", DefaultErrorLogPath, string(expectedWFLogs), string(actualWFLogs))
	}
}

func BenchmarkWriteLog(b *testing.B) {
	Init(&Config{})
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Infof(ctx, "key1=%v||key2=%v||a line of log", "abcdef", 1234)
	}

	b.StopTimer()
	Flush()
}
