package trance

import (
	"reflect"
)

type Dialect interface {
	BuildDelete(QueryConfig) (string, []any, error)
	BuildInsert(QueryConfig, map[string]any, ...string) (string, []any, error)
	BuildSelect(QueryConfig) (string, []any, error)
	BuildTableColumnAdd(QueryConfig, string) (string, error)
	BuildTableColumnDrop(QueryConfig, string) (string, error)
	BuildTableCreate(QueryConfig, TableCreateConfig) (string, error)
	BuildTableDrop(QueryConfig, TableDropConfig) (string, error)
	BuildUpdate(QueryConfig, map[string]any, ...string) (string, []any, error)
	ColumnType(reflect.StructField) (string, error)
	Param(i int) string
	QuoteIdentifier(string) string
}

type DialectStringer interface {
	StringForDialect(Dialect) string
}

type DialectStringerWithArgs interface {
	StringWithArgs(Dialect, []any) (string, []any, error)
}

var defaultDialect Dialect

func SetDialect(dialect Dialect) {
	defaultDialect = dialect
}
