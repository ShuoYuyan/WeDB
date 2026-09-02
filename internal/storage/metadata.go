package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/wedb/wedb/internal/types"
)

const (
	metadataPageSize = 4096
	magicNumber     = "WeDB"
	metadataVersion = 1
)

// Metadata 数据库元数据
type Metadata struct {
	MagicNumber    string
	Version        int
	TableCount     int
	Tables         map[string]TableMetadata
	TableRowIDs    map[string]int64 // 表级别的自增ID计数器
}

// TableMetadata 表元数据
type TableMetadata struct {
	TableName  string
	RootPage   int
	Columns    []ColumnMetadata
	PrimaryKey string
}

// ColumnMetadata 列元数据
type ColumnMetadata struct {
	Name          string
	Type          string
	PrimaryKey    bool
	AutoIncrement bool
}

// SaveMetadata 保存元数据到文件
func SaveMetadata(filePath string, tables map[string]int, schema *types.Schema, tableRowIDs map[string]int64) error {
	// 创建元数据
	metadata := &Metadata{
		MagicNumber: magicNumber,
		Version:     metadataVersion,
		TableCount:  len(tables),
		Tables:      make(map[string]TableMetadata),
		TableRowIDs: tableRowIDs,
	}

	// 填充表元数据
	for tableName, rootPage := range tables {
		table := schema.GetTable(tableName)
		if table == nil {
			continue
		}

		columns := make([]ColumnMetadata, len(table.Columns))
		for i, col := range table.Columns {
			// 将 DataType 转换为字符串
			var typeStr string
			switch col.Type {
			case types.TypeInteger:
				typeStr = "INTEGER"
			case types.TypeReal:
				typeStr = "REAL"
			case types.TypeText:
				typeStr = "TEXT"
			case types.TypeBlob:
				typeStr = "BLOB"
			default:
				typeStr = "NULL"
			}

			columns[i] = ColumnMetadata{
				Name:          col.Name,
				Type:          typeStr,
				PrimaryKey:    col.PrimaryKey,
				AutoIncrement: col.AutoIncrement,
			}
		}

		primaryKey := ""
		if table.PrimaryIndex != nil && len(table.PrimaryIndex.Columns) > 0 {
			primaryKey = table.PrimaryIndex.Columns[0]
		}
		metadata.Tables[tableName] = TableMetadata{
			TableName:  tableName,
			RootPage:   rootPage,
			Columns:    columns,
			PrimaryKey: primaryKey,
		}
	}

	// 序列化元数据
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// 写入元数据文件
	metadataFile := filePath + ".metadata"
	if err := os.WriteFile(metadataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// ErrNoMetadata 表示元数据文件不存在（新数据库的正常状态）
var ErrNoMetadata = errors.New("wedb: no metadata file")

// LoadMetadata 从文件加载元数据
func LoadMetadata(filePath string) (*Metadata, error) {
	// 读取元数据文件
	metadataFile := filePath + ".metadata"
	data, err := os.ReadFile(metadataFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoMetadata // 哨兵错误，调用方以 == 判定（兼容 Go1.10 镜像）
		}
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	// 反序列化元数据
	var metadata Metadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// 验证魔数
	if metadata.MagicNumber != magicNumber {
		return nil, fmt.Errorf("invalid metadata file: wrong magic number")
	}

	return &metadata, nil
}