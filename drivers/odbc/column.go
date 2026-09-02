package main

import (
	"github.com/wedb/wedb/internal/api"
)

// columnInfo is the lightweight shape we surface from SQLColumns /
// SQLGetTypeInfo. We don't import api.TableSchema into this header
// (it would leak storage types into the public driver surface).
type columnInfo struct {
	Name     string
	Type     int16
	Size     int64
	Nullable bool
}

// extractColumns pulls a []columnInfo from an *api.TableSchema returned
// by GetTableSchema. Returns (nil, false) if schema is the wrong type.
func extractColumns(schema interface{}) ([]columnInfo, bool) {
	ts, ok := schema.(*api.TableSchema)
	if !ok {
		return nil, false
	}
	out := make([]columnInfo, 0, len(ts.Columns))
	for _, c := range ts.Columns {
		out = append(out, columnInfo{
			Name:     c.Name,
			Type:     mapColumnTypeToSQLType(c.Type),
			Size:     255,
			Nullable: c.Nullable,
		})
	}
	return out, true
}

func mapColumnTypeToSQLType(t api.ColumnType) int16 {
	switch t {
	case api.TypeInteger:
		return 4 // SQL_INTEGER
	case api.TypeReal:
		return 8 // SQL_DOUBLE
	case api.TypeText:
		return 12 // SQL_VARCHAR
	case api.TypeBlob:
		return -4 // SQL_LONGVARBINARY
	case api.TypeNull:
		return 0
	}
	return 12
}

func typeName(sqlType int16) string {
	switch sqlType {
	case 1:
		return "CHAR"
	case 12:
		return "VARCHAR"
	case -1:
		return "TEXT"
	case 4:
		return "INTEGER"
	case -5:
		return "BIGINT"
	case 5:
		return "SMALLINT"
	case -6:
		return "TINYINT"
	case -7:
		return "BIT"
	case 7:
		return "REAL"
	case 6:
		return "FLOAT"
	case 8:
		return "DOUBLE"
	case 2:
		return "NUMERIC"
	case 3:
		return "DECIMAL"
	case 9:
		return "DATE"
	case 10:
		return "TIME"
	case 11:
		return "TIMESTAMP"
	case -2:
		return "BINARY"
	case -3:
		return "VARBINARY"
	case -4:
		return "BLOB"
	}
	return "VARCHAR"
}

func nullableInt(nullable bool) int16 {
	if nullable {
		return 1 // SQL_NULLABLE
	}
	return 0 // SQL_NO_NULLS
}

func nullableStr(nullable bool) string {
	if nullable {
		return "YES"
	}
	return "NO"
}
