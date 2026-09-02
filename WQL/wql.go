// Package wql is a stub replacement module for github.com/wedb/wedb/WQL.
// The full WQL implementation is not part of this build; this stub keeps
// `go build ./...` working until the real module is restored.
package wql

// Database is a marker interface matching the historical WQL contract.
// Real callers should use github.com/wedb/wedb/internal/api.Database.
type Database interface {
	Close() error
}
