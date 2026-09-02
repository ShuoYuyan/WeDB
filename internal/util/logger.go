package util

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	// DEBUG 调试级别
	DEBUG LogLevel = iota
	// INFO 信息级别
	INFO
	// WARN 警告级别
	WARN
	// ERROR 错误级别
	ERROR
	// FATAL 致命错误级别
	FATAL
)

// String 返回日志级别的字符串表示
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Color 返回日志级别的颜色代码
func (l LogLevel) Color() string {
	switch l {
	case DEBUG:
		return "\033[36m" // 青色
	case INFO:
		return "\033[32m" // 绿色
	case WARN:
		return "\033[33m" // 黄色
	case ERROR:
		return "\033[31m" // 红色
	case FATAL:
		return "\033[35m" // 紫色
	default:
		return "\033[0m"
	}
}

// ResetColor 重置颜色
func ResetColor() string {
	return "\033[0m"
}

// LogEntry 日志条目
type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Message   string
	Fields    map[string]interface{}
	File      string
	Line      int
	Function  string
}

// Logger 日志记录器接口
type Logger interface {
	// Debug 记录调试信息
	Debug(msg string, fields ...map[string]interface{})
	// Info 记录信息
	Info(msg string, fields ...map[string]interface{})
	// Warn 记录警告
	Warn(msg string, fields ...map[string]interface{})
	// Error 记录错误
	Error(msg string, fields ...map[string]interface{})
	// Fatal 记录致命错误
	Fatal(msg string, fields ...map[string]interface{})
	// WithFields 添加字段
	WithFields(fields map[string]interface{}) Logger
	// SetLevel 设置日志级别
	SetLevel(level LogLevel)
	// GetLevel 获取当前日志级别
	GetLevel() LogLevel
	// Close 关闭日志记录器
	Close() error
}

// Appender 日志输出器接口
type Appender interface {
	Write(entry *LogEntry) error
	Close() error
}

// ConsoleAppender 控制台输出器
type ConsoleAppender struct {
	mu       sync.Mutex
	out      io.Writer
	colorize bool
}

// NewConsoleAppender 创建新的控制台输出器
func NewConsoleAppender(colorize bool) *ConsoleAppender {
	return &ConsoleAppender{
		out:      os.Stdout,
		colorize: colorize,
	}
}

// Write 写入日志条目
func (ca *ConsoleAppender) Write(entry *LogEntry) error {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	var levelColor, resetColor string
	if ca.colorize {
		levelColor = entry.Level.Color()
		resetColor = ResetColor()
	}

	// 格式化时间
	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05.000")

	// 构建日志消息
	msg := fmt.Sprintf("%s [%s] %s", timestamp, entry.Level.String(), entry.Message)

	// 添加字段
	if len(entry.Fields) > 0 {
		msg += " |"
		for k, v := range entry.Fields {
			msg += fmt.Sprintf(" %s=%v", k, v)
		}
	}

	// 添加位置信息
	if entry.File != "" {
		msg += fmt.Sprintf(" | %s:%d", entry.File, entry.Line)
	}

	// 输出到控制台
	if ca.colorize {
		fmt.Fprintf(ca.out, "%s%s%s\n", levelColor, msg, resetColor)
	} else {
		fmt.Fprintf(ca.out, "%s\n", msg)
	}

	return nil
}

// Close 关闭控制台输出器
func (ca *ConsoleAppender) Close() error {
	return nil
}

// FileAppender 文件输出器
type FileAppender struct {
	mu           sync.Mutex
	file         *os.File
	filePath     string
	maxSize      int64
	currentSize  int64
	maxBackups   int
	compress     bool
}

// NewFileAppender 创建新的文件输出器
func NewFileAppender(filePath string, maxSize int64, maxBackups int, compress bool) (*FileAppender, error) {
	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// 打开或创建日志文件
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// 获取当前文件大小
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to get file size: %w", err)
	}

	return &FileAppender{
		file:        file,
		filePath:    filePath,
		maxSize:     maxSize,
		currentSize: stat.Size(),
		maxBackups:  maxBackups,
		compress:    compress,
	}, nil
}

// Write 写入日志条目
func (fa *FileAppender) Write(entry *LogEntry) error {
	fa.mu.Lock()
	defer fa.mu.Unlock()

	// 格式化日志条目
	line := fmt.Sprintf("[%s] [%s] %s",
		entry.Timestamp.Format("2006-01-02 15:04:05.000"),
		entry.Level.String(),
		entry.Message)

	// 添加字段
	if len(entry.Fields) > 0 {
		line += " |"
		for k, v := range entry.Fields {
			line += fmt.Sprintf(" %s=%v", k, v)
		}
	}

	// 添加位置信息
	if entry.File != "" {
		line += fmt.Sprintf(" | %s:%d", entry.File, entry.Line)
	}

	// 写入文件
	line += "\n"
	n, err := fa.file.WriteString(line)
	if err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	fa.currentSize += int64(n)

	// 检查是否需要轮转
	if fa.maxSize > 0 && fa.currentSize >= fa.maxSize {
		if err := fa.rotate(); err != nil {
			return fmt.Errorf("failed to rotate log file: %w", err)
		}
	}

	return nil
}

// rotate 轮转日志文件
func (fa *FileAppender) rotate() error {
	// 关闭当前文件
	if err := fa.file.Close(); err != nil {
		return fmt.Errorf("failed to close log file: %w", err)
	}

	// 轮转备份文件
	for i := fa.maxBackups; i >= 1; i-- {
		oldPath := fmt.Sprintf("%s.%d", fa.filePath, i-1)
		newPath := fmt.Sprintf("%s.%d", fa.filePath, i)
		
		if i == 1 {
			// 原文件变成.1
			if err := os.Rename(fa.filePath, oldPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to rotate log file: %w", err)
			}
		} else {
			// 轮转其他备份文件
			if err := os.Rename(oldPath, newPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to rotate log file: %w", err)
			}
		}
	}

	// 创建新日志文件
	file, err := os.OpenFile(fa.filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %w", err)
	}

	fa.file = file
	fa.currentSize = 0

	return nil
}

// Close 关闭文件输出器
func (fa *FileAppender) Close() error {
	fa.mu.Lock()
	defer fa.mu.Unlock()

	if fa.file != nil {
		return fa.file.Close()
	}
	return nil
}

// AuditLogger 审计日志记录器
type AuditLogger struct {
	logger    Logger
	file      string
	txID      string // 事务ID
	user      string // 用户ID
	mu        sync.RWMutex
}

// NewAuditLogger 创建新的审计日志记录器
func NewAuditLogger(logger Logger, file string) *AuditLogger {
	return &AuditLogger{
		logger: logger,
		file:   file,
	}
}

// SetTxID 设置事务ID
func (al *AuditLogger) SetTxID(txID string) {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.txID = txID
}

// SetUser 设置用户ID
func (al *AuditLogger) SetUser(user string) {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.user = user
}

// LogAudit 记录审计日志
func (al *AuditLogger) LogAudit(operation string, object string, details map[string]interface{}) {
	al.mu.RLock()
	defer al.mu.RUnlock()

	fields := make(map[string]interface{})
	for k, v := range details {
		fields[k] = v
	}

	if al.txID != "" {
		fields["tx_id"] = al.txID
	}
	if al.user != "" {
		fields["user"] = al.user
	}

	fields["operation"] = operation
	fields["object"] = object
	fields["timestamp"] = time.Now().Unix()

	al.logger.Info(fmt.Sprintf("AUDIT: %s %s", operation, object), fields)
}

// StdLogger 标准日志记录器
type StdLogger struct {
	mu       sync.RWMutex
	level    LogLevel
	appenders []Appender
	name     string
}

// NewStdLogger 创建新的标准日志记录器
func NewStdLogger(name string, level LogLevel) *StdLogger {
	return &StdLogger{
		level:    level,
		appenders: make([]Appender, 0),
		name:     name,
	}
}

// AddAppender 添加日志输出器
func (sl *StdLogger) AddAppender(appender Appender) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.appenders = append(sl.appenders, appender)
}

// SetLevel 设置日志级别
func (sl *StdLogger) SetLevel(level LogLevel) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.level = level
}

// GetLevel 获取当前日志级别
func (sl *StdLogger) GetLevel() LogLevel {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.level
}

// WithFields 添加字段
func (sl *StdLogger) WithFields(fields map[string]interface{}) Logger {
	// 创建一个新的logger实例，共享appenders
	newLogger := &StdLogger{
		level:    sl.level,
		appenders: sl.appenders,
		name:     sl.name,
	}
	return newLogger
}

// log 记录日志
func (sl *StdLogger) log(level LogLevel, msg string, fields []map[string]interface{}, file string, line int, function string) {
	if level < sl.level {
		return
	}

	// 合并所有字段
	allFields := make(map[string]interface{})
	for _, f := range fields {
		for k, v := range f {
			allFields[k] = v
		}
	}
	allFields["logger"] = sl.name

	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   msg,
		Fields:    allFields,
		File:      file,
		Line:      line,
		Function:  function,
	}

	// 写入所有输出器
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	for _, appender := range sl.appenders {
		if err := appender.Write(entry); err != nil {
			// 如果写入失败，尝试输出到stderr
			fmt.Fprintf(os.Stderr, "Failed to write log: %v\n", err)
		}
	}

	// 如果是FATAL级别，退出程序
	if level == FATAL {
		os.Exit(1)
	}
}

// Debug 记录调试信息
func (sl *StdLogger) Debug(msg string, fields ...map[string]interface{}) {
	sl.log(DEBUG, msg, fields, "", 0, "")
}

// Info 记录信息
func (sl *StdLogger) Info(msg string, fields ...map[string]interface{}) {
	sl.log(INFO, msg, fields, "", 0, "")
}

// Warn 记录警告
func (sl *StdLogger) Warn(msg string, fields ...map[string]interface{}) {
	sl.log(WARN, msg, fields, "", 0, "")
}

// Error 记录错误
func (sl *StdLogger) Error(msg string, fields ...map[string]interface{}) {
	sl.log(ERROR, msg, fields, "", 0, "")
}

// Fatal 记录致命错误
func (sl *StdLogger) Fatal(msg string, fields ...map[string]interface{}) {
	sl.log(FATAL, msg, fields, "", 0, "")
}

// Close 关闭日志记录器
func (sl *StdLogger) Close() error {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	var lastErr error
	for _, appender := range sl.appenders {
		if err := appender.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// 全局日志记录器
var (
	DefaultLogger *StdLogger
	auditLogger   *AuditLogger
)

// InitLogging 初始化日志系统
func InitLogging(level LogLevel, logFile string) error {
	// 重复初始化时先释放旧日志器持有的文件句柄，避免资源泄漏
	if DefaultLogger != nil {
		_ = DefaultLogger.Close()
		DefaultLogger = nil
	}

	DefaultLogger = NewStdLogger("WeDB", level)

	// 添加控制台输出器
	consoleAppender := NewConsoleAppender(true)
	DefaultLogger.AddAppender(consoleAppender)

	// 添加文件输出器（如果指定了日志文件）
	if logFile != "" {
		fileAppender, err := NewFileAppender(logFile, 100*1024*1024, 5, false) // 100MB, 5个备份
		if err != nil {
			return fmt.Errorf("failed to create file appender: %w", err)
		}
		DefaultLogger.AddAppender(fileAppender)
	}

	// 创建审计日志记录器
	auditLogger = NewAuditLogger(DefaultLogger, logFile)

	return nil
}

// GetLogger 获取默认日志记录器
func GetLogger() Logger {
	return DefaultLogger
}

// CloseLogging 关闭全局日志（主要供测试与进程退出前调用）
func CloseLogging() error {
	if DefaultLogger == nil {
		return nil
	}
	return DefaultLogger.Close()
}

// GetAuditLogger 获取审计日志记录器
func GetAuditLogger() *AuditLogger {
	return auditLogger
}
