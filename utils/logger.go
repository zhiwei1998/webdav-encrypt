package utils

import (
	"log"
	"os"
	"strings"
)

// LogLevel 日志级别类型
type LogLevel int

// 定义日志级别枚举
const (
	LogLevelTrace LogLevel = iota
	LogLevelDebug
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// 日志级别名称映射
var logLevelNames = map[LogLevel]string{
	LogLevelTrace: "TRACE",
	LogLevelDebug: "DEBUG",
	LogLevelInfo:  "INFO",
	LogLevelWarn:  "WARN",
	LogLevelError: "ERROR",
	LogLevelFatal: "FATAL",
}

// String 返回日志级别的字符串表示
func (level LogLevel) String() string {
	if name, ok := logLevelNames[level]; ok {
		return name
	}
	return "UNKNOWN"
}

// ParseLogLevel 解析日志级别字符串
func ParseLogLevel(level string) LogLevel {
	level = strings.ToUpper(level)
	switch level {
	case "TRACE":
		return LogLevelTrace
	case "DEBUG":
		return LogLevelDebug
	case "INFO":
		return LogLevelInfo
	case "WARN", "WARNING":
		return LogLevelWarn
	case "ERROR":
		return LogLevelError
	case "FATAL":
		return LogLevelFatal
	default:
		return LogLevelInfo // 默认信息级别
	}
}

// GetLogLevel 根据字符串配置和debug标志获取日志级别，处理向后兼容
func GetLogLevel(logLevel string, debug bool) LogLevel {
	// 如果设置了logLevel，优先使用
	if logLevel != "" {
		return ParseLogLevel(logLevel)
	}
	// 否则根据debug标志决定
	if debug {
		return LogLevelDebug
	}
	return LogLevelInfo
}

// Logger 日志接口
type Logger interface {
	Trace(format string, args ...interface{})
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Fatal(format string, args ...interface{})
}

// DefaultLogger 默认日志实现
type DefaultLogger struct {
	logger *log.Logger
	level  LogLevel
	debug  bool // 保持向后兼容
}

// NewDefaultLogger 创建默认日志器（保持向后兼容）
func NewDefaultLogger(debug bool) *DefaultLogger {
	level := LogLevelInfo
	if debug {
		level = LogLevelDebug
	}
	return &DefaultLogger{
		logger: log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile),
		level:  level,
		debug:  debug,
	}
}

// NewLogger 创建指定级别日志器
func NewLogger(level LogLevel) *DefaultLogger {
	return &DefaultLogger{
		logger: log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile),
		level:  level,
		debug:  level <= LogLevelDebug,
	}
}

// log 内部日志输出方法
func (l *DefaultLogger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	prefix := "[" + level.String() + "] "
	l.logger.Printf(prefix+format, args...)
}

// Trace 跟踪日志
func (l *DefaultLogger) Trace(format string, args ...interface{}) {
	l.log(LogLevelTrace, format, args...)
}

// Debug 调试日志
func (l *DefaultLogger) Debug(format string, args ...interface{}) {
	l.log(LogLevelDebug, format, args...)
}

// Info 信息日志
func (l *DefaultLogger) Info(format string, args ...interface{}) {
	l.log(LogLevelInfo, format, args...)
}

// Warn 警告日志
func (l *DefaultLogger) Warn(format string, args ...interface{}) {
	l.log(LogLevelWarn, format, args...)
}

// Error 错误日志
func (l *DefaultLogger) Error(format string, args ...interface{}) {
	l.log(LogLevelError, format, args...)
}

// Fatal 致命错误日志
func (l *DefaultLogger) Fatal(format string, args ...interface{}) {
	l.log(LogLevelFatal, format, args...)
	os.Exit(1)
}