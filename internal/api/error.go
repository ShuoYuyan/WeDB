package api

import "fmt"

// ErrorCode 是 WeDB 的错误码定义
// 对应 SQLite 的错误码
type ErrorCode int

const (
	OK               ErrorCode = 0   // 成功
	Error            ErrorCode = 1   // SQL 错误或缺少数据库
	Internal         ErrorCode = 2   // 内部逻辑错误
	Perm             ErrorCode = 3   // 访问权限被拒绝
	Abort            ErrorCode = 4   // 回调函数请求中止
	Busy             ErrorCode = 5   // 数据库文件被锁定
	Locked           ErrorCode = 6   // 数据库表被锁定
	NOMEM            ErrorCode = 7   // 内存分配失败
	ReadOnly         ErrorCode = 8   // 尝试写入只读数据库
	Interrupt        ErrorCode = 9   // 操作被中断
	IOErr            ErrorCode = 10  // 磁盘 I/O 错误
	Corrupt          ErrorCode = 11  // 数据库磁盘映像格式错误
	Full             ErrorCode = 13  // 数据库已满
	CantOpen         ErrorCode = 14  // 无法打开数据库文件
	Protocol         ErrorCode = 15  // 数据库锁定协议错误
	Empty            ErrorCode = 16  // 数据库为空
	Schema           ErrorCode = 17  // 数据库模式更改
	TooBig           ErrorCode = 18  // 字符串或 BLOB 超出大小限制
	Constraint       ErrorCode = 19  // 违反约束
	Mismatch         ErrorCode = 20  // 数据类型不匹配
	Misuse           ErrorCode = 21  // 库使用错误
	Range            ErrorCode = 25  // 绑定参数索引越界
	NotADB           ErrorCode = 26  // 文件不是数据库文件
	Notice           ErrorCode = 27  // 来自日志的通知
	Warning          ErrorCode = 28  // 来自日志的警告
	Row              ErrorCode = 100  // sqlite3_step() 有另一行准备好
	Done             ErrorCode = 101  // sqlite3_step() 执行完成
)

// ErrorString 返回错误码的字符串描述
func (e ErrorCode) String() string {
	switch e {
	case OK:
		return "not an error"
	case Error:
		return "SQL error or missing database"
	case Internal:
		return "internal logic error in SQLite"
	case Perm:
		return "access permission denied"
	case Abort:
		return "callback routine requested an abort"
	case Busy:
		return "database is locked"
	case Locked:
		return "database table is locked"
	case NOMEM:
		return "out of memory"
	case ReadOnly:
		return "attempt to write a readonly database"
	case Interrupt:
		return "operation terminated by sqlite3_interrupt()"
	case IOErr:
		return "disk I/O error"
	case Corrupt:
		return "database disk image is malformed"
	case Full:
		return "database or disk is full"
	case CantOpen:
		return "unable to open database file"
	case Protocol:
		return "locking protocol"
	case Empty:
		return "database is empty"
	case Schema:
		return "database schema has changed"
	case TooBig:
		return "string or blob too big"
	case Constraint:
		return "constraint failed"
	case Mismatch:
		return "data type mismatch"
	case Misuse:
		return "library used incorrectly"
	case Range:
		return "bind parameter index out of range"
	case NotADB:
		return "file is encrypted or is not a database"
	case Notice:
		return "notifications from sqlite3_log()"
	case Warning:
		return "warnings from sqlite3_log()"
	case Row:
		return "sqlite3_step() has another row ready"
	case Done:
		return "sqlite3_step() has finished executing"
	default:
		return "unknown error"
	}
}

// IsError 判断是否为错误
func (e ErrorCode) IsError() bool {
	return e != OK && e != Row && e != Done
}

// IsBusy 判断是否为忙错误
func (e ErrorCode) IsBusy() bool {
	return e == Busy || e == Locked
}

// IsIOError 判断是否为 I/O 错误
func (e ErrorCode) IsIOError() bool {
	return e == IOErr || e == Corrupt || e == Full || e == CantOpen
}

// IsConstraint 判断是否为约束错误
func (e ErrorCode) IsConstraint() bool {
	return e == Constraint
}

// Result 是 SQL 执行结果
type Result struct {
	ErrorCode ErrorCode
	ErrorMsg  string
	LastInsertRowid int64
	RowsAffected    int64
}

// NewResult 创建新的结果
func NewResult(code ErrorCode, msg string) *Result {
	return &Result{
		ErrorCode: code,
		ErrorMsg:  msg,
	}
}

// OK 创建成功结果
func OKResult() *Result {
	return &Result{
		ErrorCode: OK,
		ErrorMsg:  "",
	}
}

// Error 创建错误结果
func ErrorResult(code ErrorCode, msg string) *Result {
	return &Result{
		ErrorCode: code,
		ErrorMsg:  msg,
	}
}

// IsOK 判断是否成功
func (r *Result) IsOK() bool {
	return r.ErrorCode == OK
}

// IsError 判断是否为错误
func (r *Result) IsError() bool {
	return r.ErrorCode.IsError()
}

// IsRow 判断是否还有行
func (r *Result) IsRow() bool {
	return r.ErrorCode == Row
}

// IsDone 判断是否完成
func (r *Result) IsDone() bool {
	return r.ErrorCode == Done
}

// GetErrorCode 获取错误码
func (r *Result) GetErrorCode() ErrorCode {
	return r.ErrorCode
}

// GetErrorMsg 获取错误消息
func (r *Result) GetErrorMsg() string {
	if r.ErrorMsg == "" {
		return r.ErrorCode.String()
	}
	return r.ErrorMsg
}

// String 返回结果的字符串表示
func (r *Result) String() string {
	if r.IsOK() {
		return "OK"
	}
	return fmt.Sprintf("%s: %s", r.ErrorCode.String(), r.ErrorMsg)
}

// DatabaseError 是数据库错误
type DatabaseError struct {
	Code    ErrorCode
	Message string
}

// Error 实现 error 接口
func (e *DatabaseError) Error() string {
	if e.Message == "" {
		return e.Code.String()
	}
	return fmt.Sprintf("%s: %s", e.Code.String(), e.Message)
}

// NewDatabaseError 创建新的数据库错误
func NewDatabaseError(code ErrorCode, message string) *DatabaseError {
	return &DatabaseError{
		Code:    code,
		Message: message,
	}
}

// NewDBError 创建简化的数据库错误
func NewDBError(code ErrorCode) *DatabaseError {
	return &DatabaseError{
		Code:    code,
		Message: "",
	}
}

// WrapError 包装错误
func WrapError(err error, code ErrorCode, message string) *DatabaseError {
	return &DatabaseError{
		Code:    code,
		Message: fmt.Sprintf("%s: %v", message, err),
	}
}