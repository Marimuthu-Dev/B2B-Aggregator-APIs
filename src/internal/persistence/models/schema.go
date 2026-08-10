package models

import (
	"strings"
	"sync"
)

// DefaultSchema is used when DB_SCHEMA is unset (UrMediConnect / legacy).
const DefaultSchema = "MediAdmin"

var (
	schemaMu   sync.RWMutex
	schemaName = DefaultSchema
)

// SetSchema sets the SQL Server schema prefix used by TableName() helpers.
// Empty or whitespace values fall back to DefaultSchema.
// Call once at process start (e.g. from config.ConnectDatabase) before any queries.
func SetSchema(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = DefaultSchema
	}
	schemaMu.Lock()
	schemaName = name
	schemaMu.Unlock()
}

// Schema returns the configured SQL Server schema name.
func Schema() string {
	schemaMu.RLock()
	defer schemaMu.RUnlock()
	return schemaName
}

// Table returns a schema-qualified table name, e.g. "MediAdmin.tbl_Login".
func Table(table string) string {
	return Schema() + "." + table
}
