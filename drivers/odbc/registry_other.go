//go:build !windows

package main

// readDSNRegistry is a no-op on non-Windows; the connection string
// driver path takes over.
func readDSNRegistry(dsn string) (string, error) { return "", nil }
