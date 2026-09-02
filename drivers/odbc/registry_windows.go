//go:build windows

package main

import (
	"errors"

	"golang.org/x/sys/windows/registry"
)

func readDSNRegistry(dsn string) (string, error) {
	for _, root := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
		k, err := registry.OpenKey(root, `SOFTWARE\ODBC\ODBC.INI\`+dsn, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		path, _, err := k.GetStringValue("DBQ")
		_ = k.Close()
		if err == nil {
			return path, nil
		}
	}
	return "", errNoDSN
}

var errNoDSN = errors.New("DSN not found in registry")
