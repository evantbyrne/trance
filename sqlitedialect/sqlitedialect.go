package sqlitedialect

import (
	"database/sql"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/evantbyrne/trance"
	"golang.org/x/exp/maps"
)

type SqliteDialect struct {
	PreserveBooleans bool
}

func (dialect SqliteDialect) BuildDelete(config trance.QueryConfig) (string, []any, error) {
	args := append([]any(nil), config.Params...)
	var queryString strings.Builder
	queryString.WriteString("DELETE FROM ")

	// TABLE
	from, err := dialect.buildTable(config)
	if err != nil {
		return "", nil, err
	}
	queryString.WriteString(from)

	// WHERE
	where, args, err := dialect.buildWhere(config, args)
	if err != nil {
		return "", nil, err
	}
	if where != "" {
		queryString.WriteString(where)
	}

	// ORDER BY
	if len(config.Sort) > 0 {
		queryString.WriteString(" ORDER BY ")
		for i, column := range config.Sort {
			if i > 0 {
				queryString.WriteString(", ")
			}
			if strings.HasPrefix(column, "-") {
				queryString.WriteString(dialect.QuoteIdentifier(column[1:]))
				queryString.WriteString(" DESC")
			} else {
				queryString.WriteString(dialect.QuoteIdentifier(column))
				queryString.WriteString(" ASC")
			}
		}
	}

	// LIMIT
	if config.Limit != nil {
		args = append(args, config.Limit)
		queryString.WriteString(" LIMIT ")
		queryString.WriteString(dialect.Param(len(args)))
	}

	// OFFSET
	if config.Offset != nil {
		return "", nil, fmt.Errorf("trance: DELETE does not support OFFSET")
	}

	return queryString.String(), args, nil
}

func (dialect SqliteDialect) BuildInsert(config trance.QueryConfig, rowMap map[string]any, columns ...string) (string, []any, error) {
	args := make([]any, 0)
	var queryString strings.Builder

	queryString.WriteString("INSERT INTO ")

	// TABLE
	from, err := dialect.buildTable(config)
	if err != nil {
		return "", nil, err
	}
	queryString.WriteString(from)

	queryString.WriteString(" (")
	first := true
	for _, column := range columns {
		if arg, ok := rowMap[column]; ok {
			if _, ok := config.Fields[column]; !ok {
				return "", nil, fmt.Errorf("trance: field for column '%s' not found on model for table '%s'", column, config.Table)
			}
			args = append(args, arg)
			if first {
				first = false
			} else {
				queryString.WriteString(",")
			}
			queryString.WriteString(dialect.QuoteIdentifier(column))
		} else {
			return "", nil, fmt.Errorf("trance: invalid column '%s' on INSERT", column)
		}
	}

	queryString.WriteString(") VALUES (")
	for i := 1; i <= len(rowMap); i++ {
		if i > 1 {
			queryString.WriteString(",")
		}
		queryString.WriteString(dialect.Param(i))
	}
	queryString.WriteString(")")

	return queryString.String(), args, nil
}

func (dialect SqliteDialect) buildJoins(config trance.QueryConfig, args []any) (string, []any, error) {
	var queryPart strings.Builder
	if len(config.Joins) > 0 {
		for _, join := range config.Joins {
			if len(join.On) > 0 {
				queryPart.WriteString(fmt.Sprintf(" %s JOIN %s ON", join.Direction, dialect.QuoteIdentifier(join.Table)))
				for _, where := range join.On {
					queryWhere, whereArgs, err := where.StringWithArgs(dialect, args)
					if err != nil {
						return "", nil, err
					}
					args = whereArgs
					queryPart.WriteString(queryWhere)
				}
			}
		}
	}
	return queryPart.String(), args, nil
}

func (dialect SqliteDialect) BuildSelect(config trance.QueryConfig) (string, []any, error) {
	args := append([]any(nil), config.Params...)
	var queryString strings.Builder
	if config.Count {
		queryString.WriteString("SELECT count(*) FROM ")
	} else if len(config.Selected) > 0 {
		queryString.WriteString("SELECT ")
		for i, column := range config.Selected {
			if i > 0 {
				queryString.WriteString(",")
			}
			switch cv := column.(type) {
			case string:
				queryString.WriteString(dialect.QuoteIdentifier(cv))

			case trance.DialectStringer:
				queryString.WriteString(cv.StringForDialect(dialect))

			case fmt.Stringer:
				queryString.WriteString(cv.String())

			default:
				return "", nil, fmt.Errorf("trance: invalid column type %#v", column)
			}
		}
		queryString.WriteString(" FROM ")
	} else {
		queryString.WriteString("SELECT * FROM ")
	}

	// TABLE
	from, err := dialect.buildTable(config)
	if err != nil {
		return "", nil, err
	}
	queryString.WriteString(from)

	// JOIN
	joins, args, err := dialect.buildJoins(config, args)
	if err != nil {
		return "", nil, err
	}
	if joins != "" {
		queryString.WriteString(joins)
	}

	// WHERE
	where, args, err := dialect.buildWhere(config, args)
	if err != nil {
		return "", nil, err
	}
	if where != "" {
		queryString.WriteString(where)
	}

	// ORDER BY
	if len(config.Sort) > 0 {
		queryString.WriteString(" ORDER BY ")
		for i, column := range config.Sort {
			if i > 0 {
				queryString.WriteString(", ")
			}
			if strings.HasPrefix(column, "-") {
				queryString.WriteString(dialect.QuoteIdentifier(column[1:]))
				queryString.WriteString(" DESC")
			} else {
				queryString.WriteString(dialect.QuoteIdentifier(column))
				queryString.WriteString(" ASC")
			}
		}
	}

	// LIMIT
	if config.Limit != nil {
		args = append(args, config.Limit)
		queryString.WriteString(" LIMIT ")
		queryString.WriteString(dialect.Param(len(args)))
	}

	// OFFSET
	if config.Offset != nil {
		args = append(args, config.Offset)
		queryString.WriteString(" OFFSET ")
		queryString.WriteString(dialect.Param(len(args)))
	}

	return queryString.String(), args, nil
}

func (dialect SqliteDialect) buildTable(config trance.QueryConfig) (string, error) {
	switch tv := config.Table.(type) {
	case string:
		return dialect.QuoteIdentifier(tv), nil

	case trance.DialectStringer:
		return tv.StringForDialect(dialect), nil

	case fmt.Stringer:
		return tv.String(), nil

	default:
		return "", fmt.Errorf("trance: invalid table type %#v", config.Table)
	}
}

func (dialect SqliteDialect) BuildTableColumnAdd(config trance.QueryConfig, column string) (string, error) {
	table, err := dialect.buildTable(config)
	if err != nil {
		return "", err
	}

	field, ok := config.Fields[column]
	if !ok {
		return "", fmt.Errorf("trance: invalid column '%s' on model for table '%s'", column, config.Table)
	}

	columnType, err := dialect.ColumnType(field)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, dialect.QuoteIdentifier(column), columnType), nil
}

func (dialect SqliteDialect) BuildTableColumnDrop(config trance.QueryConfig, column string) (string, error) {
	table, err := dialect.buildTable(config)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", table, dialect.QuoteIdentifier(column)), nil
}

func (dialect SqliteDialect) BuildTableCreate(config trance.QueryConfig, tableCreateConfig trance.TableCreateConfig) (string, error) {
	var sql strings.Builder
	sql.WriteString("CREATE TABLE ")
	if tableCreateConfig.IfNotExists {
		sql.WriteString("IF NOT EXISTS ")
	}

	// TABLE
	table, err := dialect.buildTable(config)
	if err != nil {
		return "", err
	}
	sql.WriteString(table)

	sql.WriteString(" (")
	fieldNames := maps.Keys(config.Fields)
	sort.Strings(fieldNames)
	for i, fieldName := range fieldNames {
		field := config.Fields[fieldName]
		columnType, err := dialect.ColumnType(field)
		if err != nil {
			return "", err
		}
		if i > 0 {
			sql.WriteString(",")
		}
		sql.WriteString("\n\t")
		sql.WriteString(dialect.QuoteIdentifier(fieldName))
		sql.WriteString(" ")
		sql.WriteString(columnType)
	}
	sql.WriteString("\n)")

	return sql.String(), nil
}

func (dialect SqliteDialect) BuildTableDrop(config trance.QueryConfig, tableDropConfig trance.TableDropConfig) (string, error) {
	var queryString strings.Builder
	queryString.WriteString("DROP TABLE ")
	if tableDropConfig.IfExists {
		queryString.WriteString("IF EXISTS ")
	}

	// TABLE
	from, err := dialect.buildTable(config)
	if err != nil {
		return "", err
	}
	queryString.WriteString(from)

	return queryString.String(), nil
}

func (dialect SqliteDialect) BuildUpdate(config trance.QueryConfig, rowMap map[string]any, columns ...string) (string, []any, error) {
	args := append([]any(nil), config.Params...)
	var queryString strings.Builder

	queryString.WriteString("UPDATE ")

	// TABLE
	from, err := dialect.buildTable(config)
	if err != nil {
		return "", nil, err
	}
	queryString.WriteString(from)

	queryString.WriteString(" SET ")

	first := true
	for _, column := range columns {
		if arg, ok := rowMap[column]; ok {
			args = append(args, arg)
			if first {
				first = false
			} else {
				queryString.WriteString(",")
			}
			queryString.WriteString(dialect.QuoteIdentifier(column))
			queryString.WriteString(" = ")
			queryString.WriteString(dialect.Param(len(args)))
		} else {
			return "", nil, fmt.Errorf("trance: invalid column '%s' on UPDATE", column)
		}
	}

	// WHERE
	where, args, err := dialect.buildWhere(config, args)
	if err != nil {
		return "", nil, err
	}
	if where != "" {
		queryString.WriteString(where)
	}

	// ORDER BY
	if len(config.Sort) > 0 {
		queryString.WriteString(" ORDER BY ")
		for i, column := range config.Sort {
			if i > 0 {
				queryString.WriteString(", ")
			}
			if strings.HasPrefix(column, "-") {
				queryString.WriteString(dialect.QuoteIdentifier(column[1:]))
				queryString.WriteString(" DESC")
			} else {
				queryString.WriteString(dialect.QuoteIdentifier(column))
				queryString.WriteString(" ASC")
			}
		}
	}

	// LIMIT
	if config.Limit != nil {
		args = append(args, config.Limit)
		queryString.WriteString(" LIMIT ")
		queryString.WriteString(dialect.Param(len(args)))
	}

	// OFFSET
	if config.Offset != nil {
		return "", nil, fmt.Errorf("trance: UPDATE does not support OFFSET")
	}

	return queryString.String(), args, nil
}

func (dialect SqliteDialect) buildWhere(config trance.QueryConfig, args []any) (string, []any, error) {
	var queryPart strings.Builder
	if len(config.Filters) > 0 {
		queryPart.WriteString(" WHERE")
		for _, where := range config.Filters {
			queryWhere, whereArgs, err := where.StringWithArgs(dialect, args)
			if err != nil {
				return "", nil, err
			}
			args = whereArgs
			queryPart.WriteString(queryWhere)
		}
	}
	return queryPart.String(), args, nil
}

func (dialect SqliteDialect) ColumnType(field reflect.StructField) (string, error) {
	tagType := field.Tag.Get("@type")
	if tagType != "" {
		return tagType, nil
	}

	fieldInstance := reflect.Indirect(reflect.New(field.Type)).Interface()
	var columnNull string
	var columnPrimary string
	var columnType string

	if field.Tag.Get("@primary") == "true" {
		columnPrimary = " PRIMARY KEY"
	}

	switch fieldInstance.(type) {
	case bool:
		columnNull = " NOT NULL"
		columnType = "BOOLEAN"

	case int, int8, int16, int32, int64:
		columnNull = " NOT NULL"
		columnType = "INTEGER"

	case sql.NullBool:
		columnNull = " NULL"
		columnType = "BOOLEAN"

	case sql.NullInt16, sql.NullInt32, sql.NullInt64:
		columnNull = " NULL"
		columnType = "INTEGER"

	case float32, float64:
		columnNull = " NOT NULL"
		columnType = "REAL"

	case sql.NullFloat64:
		columnNull = " NULL"
		columnType = "REAL"

	case string:
		columnNull = " NOT NULL"
		columnType = "TEXT"

	case time.Time:
		columnNull = " NOT NULL"
		columnType = "DATETIME"

	case sql.NullString:
		columnNull = " NULL"
		columnType = "TEXT"

	case sql.NullTime:
		columnNull = " NULL"
		columnType = "DATETIME"

	default:
		if strings.HasPrefix(field.Type.String(), "trance.ForeignKey[") || strings.HasPrefix(field.Type.String(), "trance.NullForeignKey[") {
			// Foreign keys.
			fv := reflect.New(field.Type).Elem()
			subModelQ := fv.Addr().MethodByName("Weave").Call(nil)
			subFields := reflect.Indirect(subModelQ[0]).FieldByName("Fields").Interface().(map[string]reflect.StructField)
			subPrimaryColumn := reflect.Indirect(subModelQ[0]).FieldByName("PrimaryColumn").Interface().(string)
			subTable := reflect.Indirect(subModelQ[0]).FieldByName("Table").Interface().(string)
			columnTypeTemp, err := dialect.ColumnType(subFields[subPrimaryColumn])
			if err != nil {
				return "", err
			}
			columnType = strings.SplitN(columnTypeTemp, " ", 2)[0]

			columnNull = " NOT NULL"
			if strings.HasPrefix(field.Type.String(), "trance.NullForeignKey[") {
				columnNull = " NULL"
			}
			columnNull = fmt.Sprintf("%s REFERENCES %s (%s)", columnNull, dialect.QuoteIdentifier(subTable), dialect.QuoteIdentifier(subPrimaryColumn))

			if tagOnUpdate := field.Tag.Get("@on_update"); tagOnUpdate != "" {
				// ON UPDATE.
				columnNull = fmt.Sprint(columnNull, " ON UPDATE ", tagOnUpdate)
			}

			if tagOnDelete := field.Tag.Get("@on_delete"); tagOnDelete != "" {
				// ON DELETE.
				columnNull = fmt.Sprint(columnNull, " ON DELETE ", tagOnDelete)
			}
		}
	}

	if columnType == "" {
		return "", fmt.Errorf("trance: Unsupported column type: %T. Use the '@type' field tag to define a SQL type", fieldInstance)
	}

	if tagDefault := field.Tag.Get("@default"); tagDefault != "" {
		// DEFAULT.
		columnNull += " DEFAULT " + tagDefault
	}

	if tagUnique := field.Tag.Get("@unique"); tagUnique == "true" {
		// UNIQUE.
		columnNull += " UNIQUE"
	}

	return fmt.Sprint(columnType, columnPrimary, columnNull), nil
}

func (dialect SqliteDialect) Param(identifier int) string {
	return "?"
}

func (dialect SqliteDialect) QuoteIdentifier(identifier string) string {
	var query strings.Builder
	for i, part := range strings.Split(identifier, ".") {
		if i > 0 {
			query.WriteString(".")
		}
		query.WriteString(QuoteIdentifier(part))
	}
	return query.String()
}

func QuoteIdentifier(identifier string) string {
	return "`" + strings.Replace(identifier, "`", "``", -1) + "`"
}
