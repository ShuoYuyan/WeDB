package api

// OpenFlag 是数据库打开标志
type OpenFlag int

const (
	OpenReadOnly   OpenFlag = 0x01 // 以只读模式打开
	OpenReadWrite  OpenFlag = 0x02 // 以读写模式打开
	OpenCreate     OpenFlag = 0x04 // 如果数据库不存在则创建
	OpenURI        OpenFlag = 0x40 // 文件名可以解释为 URI
	OpenMemory     OpenFlag = 0x80 // 内存数据库
	OpenNoMutex    OpenFlag = 0x8000 // 多线程模式
	OpenFullMutex  OpenFlag = 0x10000 // 串行化模式
	OpenSharedCache OpenFlag = 0x20000 // 共享缓存模式
	OpenPrivateCache OpenFlag = 0x40000 // 私有缓存模式
)

// PrepareFlag 是语句准备标志
type PrepareFlag int

const (
	PreparePersistent PrepareFlag = 0x01 // 持久化语句
	PrepareNormalize   PrepareFlag = 0x02 // 规范化 SQL
	PrepareNoVtab      PrepareFlag = 0x04 // 禁止虚拟表
)

// ConfigOption 是数据库配置选项
type ConfigOption string

const (
	ConfigSingleThread      ConfigOption = "single_thread"       // 单线程模式
	ConfigMultiThread       ConfigOption = "multi_thread"        // 多线程模式
	ConfigSerialized        ConfigOption = "serialized"          // 串行化模式
	ConfigMemStatus         ConfigOption = "mem_status"          // 内存状态
	ConfigLookaside         ConfigOption = "lookaside"           // Lookaside 配置
	ConfigPageCacheSize     ConfigOption = "page_cache_size"     // 页面缓存大小
	ConfigMmapSize          ConfigOption = "mmap_size"           // 内存映射大小
	ConfigCoveringIndexScan ConfigOption = "covering_index_scan" // 覆盖索引扫描
	ConfigSQLLog            ConfigOption = "sql_log"             // SQL 日志
	ConfigTriggerUpdates    ConfigOption = "trigger_updates"     // 触发器更新
	ConfigTransaction       ConfigOption = "transaction"         // 事务
	ConfigDefensive         ConfigOption = "defensive"           // 防御模式
)

// Pragma 是 PRAGMA 语句的选项
type Pragma string

const (
	PragmaCacheSize         Pragma = "cache_size"          // 缓存大小
	PragmaJournalMode       Pragma = "journal_mode"        // 日志模式
	PragmaSynchronous       Pragma = "synchronous"         // 同步模式
	PragmaTempStore         Pragma = "temp_store"          // 临时存储
	PragmaPageSize          Pragma = "page_size"           // 页面大小
	PragmaEncoding          Pragma = "encoding"            // 编码
	PragmaAutoVacuum        Pragma = "auto_vacuum"         // 自动清理
	PragmaForeignKeys       Pragma = "foreign_keys"        // 外键
	PragmaUserVersion       Pragma = "user_version"        // 用户版本
	PragmaApplicationID     Pragma = "application_id"      // 应用 ID
	PragmaTableInfo         Pragma = "table_info"          // 表信息
	PragmaIndexList         Pragma = "index_list"          // 索引列表
	PragmaIndexInfo         Pragma = "index_info"          // 索引信息
	PragmaDatabaseList      Pragma = "database_list"       // 数据库列表
	PragmaTableList         Pragma = "table_list"          // 表列表
	PragmaCollationList     Pragma = "collation_list"      // 排序序列列表
)

// JournalMode 是日志模式
type JournalMode string

const (
	JournalModeDelete   JournalMode = "DELETE"   // 删除日志
	JournalModeTruncate JournalMode = "TRUNCATE" // 截断日志
	JournalModePersist  JournalMode = "PERSIST"  // 保留日志
	JournalModeMemory   JournalMode = "MEMORY"   // 内存日志
	JournalModeWAL      JournalMode = "WAL"      // WAL 模式
	JournalModeOff      JournalMode = "OFF"      // 关闭日志
)

// SynchronousMode 是同步模式
type SynchronousMode int

const (
	SyncOff     SynchronousMode = 0 // 不同步
	SyncNormal  SynchronousMode = 1 // 正常同步
	SyncFull    SynchronousMode = 2 // 完全同步
	SyncExtra   SynchronousMode = 3 // 额外同步
)

// String 返回同步模式的字符串表示
func (s SynchronousMode) String() string {
	switch s {
	case SyncOff:
		return "OFF"
	case SyncNormal:
		return "NORMAL"
	case SyncFull:
		return "FULL"
	case SyncExtra:
		return "EXTRA"
	default:
		return "UNKNOWN"
	}
}

// Config 是数据库配置
type Config struct {
	ThreadMode      string    // 线程模式
	PageSize        int       // 页面大小
	CacheSize       int       // 缓存大小
	JournalMode     JournalMode // 日志模式
	Synchronous     SynchronousMode // 同步模式
	Encoding        string    // 编码
	AutoVacuum      bool      // 自动清理
	ForeignKeys     bool      // 外键支持
	MmapSize        int64     // 内存映射大小
	LookasideSize   int       // Lookaside 大小
	LookasideCount  int       // Lookaside 数量
	Defensive       bool      // 防御模式
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		ThreadMode:     "multi_thread",
		PageSize:       4096,
		CacheSize:      2000,
		JournalMode:    JournalModeDelete,
		Synchronous:    SyncFull,
		Encoding:       "UTF-8",
		AutoVacuum:     false,
		ForeignKeys:    false,
		MmapSize:       0,
		LookasideSize:  1200,
		LookasideCount: 100,
		Defensive:      true,
	}
}

// NewConfig 创建新配置
func NewConfig() *Config {
	return DefaultConfig()
}