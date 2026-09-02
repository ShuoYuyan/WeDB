package main

// bridge.go adapts the storage package to the handle.Database interface
// expected by drivers/odbc/sql. We can't directly use *storage.WeDBDatabase
// because its method signatures don't all match the driver interface
// (notably GetTableSchema returns *api.TableSchema, an interface value the
// driver wants as `interface{}`).

import (
	"github.com/wedb/wedb/drivers/odbc/handle"
	"github.com/wedb/wedb/internal/api"
	"github.com/wedb/wedb/internal/storage"
)

// storageAdapter wraps *storage.WeDBDatabase and exposes it as
// handle.Database.
type storageAdapter struct {
	db *storage.WeDBDatabase
}

// newStorageAdapter returns the adapter.
func newStorageAdapter(db *storage.WeDBDatabase) *storageAdapter {
	return &storageAdapter{db: db}
}

// Compile-time check: storageAdapter must implement handle.Database.
var _ handle.Database = (*storageAdapter)(nil)

func (a *storageAdapter) Ping() error               { return a.db.Ping() }
func (a *storageAdapter) Close() error              { return a.db.Close() }
func (a *storageAdapter) IsClosed() bool            { return a.db.IsClosed() }
func (a *storageAdapter) ListTables() []string      { return a.db.ListTables() }
func (a *storageAdapter) TableExists(n string) bool { return a.db.TableExists(n) }

func (a *storageAdapter) GetTableSchema(name string) (interface{}, error) {
	return a.db.GetTableSchema(name)
}

// CreateTable accepts either an *api.TableSchema (preferred) or a
// generic map[string]interface{} (from the SQL parser). We convert.
func (a *storageAdapter) CreateTable(schema interface{}) error {
	switch v := schema.(type) {
	case *api.TableSchema:
		return a.db.CreateTable(v)
	case map[string]interface{}:
		return a.db.CreateTable(mapToTableSchema(v))
	}
	// try a round-trip through JSON-like conversion
	return a.db.CreateTable(mapToTableSchema(schema.(map[string]interface{})))
}

func (a *storageAdapter) DropTable(name string) error { return a.db.DropTable(name) }

func (a *storageAdapter) ScanTable(name string) ([]map[string]interface{}, error) {
	return a.db.ScanTable(name)
}
func (a *storageAdapter) ScanTableWithColumns(name string, cols []string) ([]map[string]interface{}, error) {
	return a.db.ScanTableWithColumns(name, cols)
}
func (a *storageAdapter) InsertRow(name string, row map[string]interface{}) error {
	return a.db.InsertRow(name, row)
}
func (a *storageAdapter) UpdateRow(name string, row map[string]interface{}, where string) error {
	return a.db.UpdateRow(name, row, where)
}
func (a *storageAdapter) DeleteRow(name string, where string) error {
	return a.db.DeleteRow(name, where)
}
func (a *storageAdapter) Count(n, w string) (int64, error) { return a.db.Count(n, w) }
func (a *storageAdapter) Min(n, c, w string) (interface{}, error) {
	return a.db.Min(n, c, w)
}
func (a *storageAdapter) Max(n, c, w string) (interface{}, error) {
	return a.db.Max(n, c, w)
}
func (a *storageAdapter) Sum(n, c, w string) (float64, error) { return a.db.Sum(n, c, w) }
func (a *storageAdapter) Avg(n, c, w string) (float64, error) { return a.db.Avg(n, c, w) }
