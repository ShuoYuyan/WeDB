package main

import (
	"github.com/wedb/wedb/drivers/odbc/handle"
	sqlpkg "github.com/wedb/wedb/drivers/odbc/sql"
	"github.com/wedb/wedb/internal/api"
)

// sqlParse wraps the SQL parser to keep the main API file small.
func sqlParse(s string) (*sqlpkg.Statement, error) {
	return sqlpkg.Parse(s)
}

// sqlNewExecutor builds an executor over the given handle.Database
// (which is always a *storageAdapter in production).
func sqlNewExecutor(db handle.Database) *sqlpkg.Executor {
	return sqlpkg.NewExecutor(db)
}

// mapToTableSchema converts a parsed CREATE TABLE map into an
// api.TableSchema for the storage layer.
func mapToTableSchema(m map[string]interface{}) *api.TableSchema {
	ts := &api.TableSchema{
		TableName: asString(m["table_name"]),
	}
	if pk, ok := m["primary_key"].(string); ok {
		ts.PrimaryKey = pk
	}
	if ai, ok := m["auto_increment"].(bool); ok {
		ts.AutoIncrement = ai
	}
	colsAny, _ := m["columns"].([]map[string]interface{})
	for _, c := range colsAny {
		ts.Columns = append(ts.Columns, api.ColumnSchema{
			Name:          asString(c["name"]),
			Type:          api.ColumnType(asString(c["type"])),
			Nullable:      asBool(c["nullable"], true),
			PrimaryKey:    asBool(c["primary_key"], false),
			AutoIncrement: asBool(c["auto_increment"], false),
			Unique:        asBool(c["unique"], false),
		})
	}
	return ts
}

func asString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func asBool(v interface{}, def bool) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return def
}
