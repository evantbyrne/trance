package trance

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"golang.org/x/exp/maps"
)

type JoinClause struct {
	Direction string
	On        []FilterClause
	Table     string
}

type JsonValuer interface {
	JsonValue() any
}

type QueryConfig struct {
	Count        bool
	Context      context.Context
	FetchRelated []string
	Fields       map[string]reflect.StructField
	Filters      []FilterClause
	Joins        []JoinClause
	Limit        any
	Offset       any
	Params       []any
	Selected     []any
	Sort         []string
	Table        string
	Transaction  *sql.Tx
}

type QueryStream[T any] struct {
	Config QueryConfig
	Error  error
	Weave  *Weave[T]
	Rows   *sql.Rows

	dialect Dialect
}

func (query *QueryStream[T]) All() *WeaveListStreamer[T] {
	result := &WeaveListStreamer[T]{
		Error:       query.Error,
		WeaveConfig: query.Weave.Config,
	}
	if result.Error != nil {
		return result
	}

	db := Database()
	if db == nil {
		result.Error = UseDatabaseError{}
		return result
	}
	query.detectDialect()
	query.configure()

	queryString, args, err := query.dialect.BuildSelect(query.Config)
	if err != nil {
		result.Error = err
		result.Values = make([]*T, 0)
		return result
	}

	rows, err := query.dbQuery(db, queryString, args...)
	if err != nil {
		result.Error = err
		result.Values = make([]*T, 0)
		return result
	}
	query.Rows = rows
	result.Values, result.Error = query.slice()
	return result
}

func (query *QueryStream[T]) AllToMap() *MapListStream {
	result := &MapListStream{
		Error: query.Error,
	}
	if result.Error != nil {
		return result
	}

	db := Database()
	if db == nil {
		result.Error = UseDatabaseError{}
		return result
	}
	query.detectDialect()
	query.configure()

	queryString, args, err := query.dialect.BuildSelect(query.Config)
	if err != nil {
		result.Error = err
		return result
	}

	rows, err := query.dbQuery(db, queryString, args...)
	if err != nil {
		result.Error = err
		return result
	}
	query.Rows = rows
	defer query.Rows.Close()

	mapped := make([]map[string]any, 0)
	for query.Rows.Next() {
		data, err := query.ScanToMap(query.Rows)
		if err != nil {
			result.Error = err
			return result
		}
		mapped = append(mapped, data)
	}

	if query.Config.Context != nil {
		select {
		default:
		case <-query.Config.Context.Done():
			result.Error = query.Config.Context.Err()
			return result
		}
	}

	result.Values = mapped
	return result
}

func (query *QueryStream[T]) configure() {
	query.Config.Fields = query.Weave.Fields
	query.Config.Table = query.Weave.Table
}

func (query *QueryStream[T]) Context(context context.Context) *QueryStream[T] {
	if query.Error != nil {
		return query
	}
	query.Config.Context = context
	return query
}

func (query *QueryStream[T]) Count() (uint, error) {
	var count uint

	db := Database()
	if db == nil {
		return count, UseDatabaseError{}
	}
	query.detectDialect()
	query.configure()

	query.Config.Count = true

	queryString, args, err := query.dialect.BuildSelect(query.Config)
	if err != nil {
		return count, err
	}

	if query.Config.Transaction != nil {
		if query.Config.Context != nil {
			err = query.Config.Transaction.QueryRowContext(query.Config.Context, queryString, args...).Scan(&count)
		} else {
			err = query.Config.Transaction.QueryRow(queryString, args...).Scan(&count)
		}
	} else if query.Config.Context != nil {
		err = db.QueryRowContext(query.Config.Context, queryString, args...).Scan(&count)
	} else {
		err = db.QueryRow(queryString, args...).Scan(&count)
	}
	if err != nil {
		return count, err
	}

	return count, nil
}

func (query *QueryStream[T]) Collect() ([]*T, error) {
	return query.All().Collect()
}

func (query *QueryStream[T]) CollectFirst() (*T, error) {
	return query.First().Collect()
}

func (query *QueryStream[T]) dbExec(db *sql.DB, queryString string, args ...any) (sql.Result, error) {
	if query.Config.Transaction != nil {
		if query.Config.Context != nil {
			return query.Config.Transaction.ExecContext(query.Config.Context, queryString, args...)
		}
		return query.Config.Transaction.Exec(queryString, args...)
	}

	if query.Config.Context != nil {
		return db.ExecContext(query.Config.Context, queryString, args...)
	}
	return db.Exec(queryString, args...)
}

func (query *QueryStream[T]) dbQuery(db *sql.DB, queryString string, args ...any) (*sql.Rows, error) {
	if query.Config.Transaction != nil {
		if query.Config.Context != nil {
			return query.Config.Transaction.QueryContext(query.Config.Context, queryString, args...)
		}
		return query.Config.Transaction.Query(queryString, args...)
	}

	if query.Config.Context != nil {
		return db.QueryContext(query.Config.Context, queryString, args...)
	}
	return db.Query(queryString, args...)
}

func (query *QueryStream[T]) Delete() *QueryResultStreamer[T] {
	result := &QueryResultStreamer[T]{
		Error:       query.Error,
		WeaveConfig: query.Weave.Config,
	}
	if result.Error != nil {
		return result
	}

	db := Database()
	if db == nil {
		result.Error = UseDatabaseError{}
		return result
	}
	query.detectDialect()
	query.configure()

	queryString, args, err := query.dialect.BuildDelete(query.Config)
	if err != nil {
		result.Error = err
		return result
	}
	result.Result, result.Error = query.dbExec(db, queryString, args...)
	return result
}

func (query *QueryStream[T]) detectDialect() {
	if query.dialect == nil {
		if defaultDialect != nil {
			query.dialect = defaultDialect
		} else {
			panic("trance: no dialect registered. Use trance.SetDialect(dialect trance.Dialect) to register a default for SQL queries")
		}
	}
}

func (query *QueryStream[T]) Dialect(dialect Dialect) *QueryStream[T] {
	if query.Error != nil {
		return query
	}
	query.dialect = dialect
	return query
}

func (query *QueryStream[T]) Exists() (bool, error) {
	db := Database()
	if db == nil {
		return false, UseDatabaseError{}
	}
	query.detectDialect()
	query.configure()

	query.Config.Limit = 1

	queryString, args, err := query.dialect.BuildSelect(query.Config)
	if err != nil {
		return false, err
	}

	rows, err := query.dbQuery(db, queryString, args...)
	if err != nil {
		return false, err
	}
	return rows.Next(), nil
}

func (query *QueryStream[T]) FetchRelated(columns ...string) *QueryStream[T] {
	if query.Error != nil {
		return query
	}
	query.Config.FetchRelated = columns
	return query
}

func (query *QueryStream[T]) Filter(column any, operator string, value any) *QueryStream[T] {
	if query.Error != nil {
		return query
	}
	if len(query.Config.Filters) > 0 {
		query.Config.Filters = append(query.Config.Filters, FilterClause{Rule: "AND"})
	}
	query.Config.Filters = append(query.Config.Filters, Q(column, operator, value))
	return query
}

func (query *QueryStream[T]) FilterAnd(clauses ...any) *QueryStream[T] {
	if query.Error != nil {
		return query
	}
	flat := make([]FilterClause, 0)
	for _, clause := range clauses {
		flat = flattenFilterClause(flat, clause)
	}

	if len(query.Config.Filters) > 0 {
		query.Config.Filters = append(query.Config.Filters, FilterClause{Rule: "AND"})
	}
	query.Config.Filters = append(query.Config.Filters, FilterClause{Rule: "("})
	indent := 0
	for i, clause := range flat {
		if i > 0 && indent == 0 && flat[i-1].Rule != "NOT" {
			query.Config.Filters = append(query.Config.Filters, FilterClause{Rule: "AND"})
		}
		if clause.Rule == "(" {
			indent++
		} else if clause.Rule == ")" {
			indent--
		}
		query.Config.Filters = append(query.Config.Filters, clause)
	}

	query.Config.Filters = append(query.Config.Filters, FilterClause{Rule: ")"})
	return query
}

func (query *QueryStream[T]) FilterOr(clauses ...any) *QueryStream[T] {
	if query.Error != nil {
		return query
	}
	flat := make([]FilterClause, 0)
	for _, clause := range clauses {
		flat = flattenFilterClause(flat, clause)
	}

	if len(query.Config.Filters) > 0 {
		query.Config.Filters = append(query.Config.Filters, FilterClause{Rule: "AND"})
	}
	query.Config.Filters = append(query.Config.Filters, FilterClause{Rule: "("})
	indent := 0
	for i, clause := range flat {
		if i > 0 && indent == 0 && flat[i-1].Rule != "NOT" {
			query.Config.Filters = append(query.Config.Filters, FilterClause{Rule: "OR"})
		}
		if clause.Rule == "(" {
			indent++
		} else if clause.Rule == ")" {
			indent--
		}
		query.Config.Filters = append(query.Config.Filters, clause)
	}

	query.Config.Filters = append(query.Config.Filters, FilterClause{Rule: ")"})
	return query
}

func (query *QueryStream[T]) First() *WeaveStreamer[T] {
	result := &WeaveStreamer[T]{
		Error:       query.Error,
		WeaveConfig: query.Weave.Config,
	}
	if result.Error != nil {
		return result
	}

	db := Database()
	if db == nil {
		result.Error = UseDatabaseError{}
		return result
	}
	query.detectDialect()
	query.configure()

	query.Limit(1)

	queryString, args, err := query.dialect.BuildSelect(query.Config)
	if err != nil {
		result.Error = err
		return result
	}

	rows, err := query.dbQuery(db, queryString, args...)
	if err != nil {
		result.Error = err
		return result
	}
	query.Rows = rows
	values, err := query.slice()
	if err != nil {
		result.Error = err
		return result
	}

	if len(values) > 0 {
		result.Value = values[0]
	} else {
		result.Error = ErrorNotFound{}
	}
	return result
}

func (query *QueryStream[T]) FirstToMap() *MapStream {
	result := &MapStream{
		Error: query.Error,
	}
	if result.Error != nil {
		return result
	}

	db := Database()
	if db == nil {
		result.Error = UseDatabaseError{}
		return result
	}
	query.detectDialect()
	query.configure()

	query.Limit(1)

	queryString, args, err := query.dialect.BuildSelect(query.Config)
	if err != nil {
		result.Error = UseDatabaseError{}
		return result
	}

	rows, err := query.dbQuery(db, queryString, args...)
	if err != nil {
		result.Error = err
		return result
	}
	query.Rows = rows

	values, err := query.slice()
	if err != nil {
		result.Error = err
		return result
	}

	if len(values) > 0 {
		result.Value, result.Error = query.Weave.ToMap(values[0])
	} else {
		result.Error = ErrorNotFound{}
	}
	return result
}

func (query *QueryStream[T]) Insert(row *T) *QueryResultStreamer[T] {
	result := &QueryResultStreamer[T]{
		Error:       query.Error,
		Value:       row,
		WeaveConfig: query.Weave.Config,
	}
	if result.Error != nil {
		return result
	}

	db := Database()
	if db == nil {
		result.Error = UseDatabaseError{}
		return result
	}
	query.detectDialect()
	query.configure()
	rowMap, err := query.Weave.ToMap(row)
	if err != nil {
		result.Error = err
		return result
	}
	queryString, args, err := query.dialect.BuildInsert(query.Config, rowMap, maps.Keys(rowMap)...)
	if err != nil {
		result.Error = err
		return result
	}

	result.Result, result.Error = query.dbExec(db, queryString, args...)
	if result.Error != nil {
		return result
	}

	// Set primary key if zero.
	if query.Weave.PrimaryField != "" {
		primaryField := reflect.ValueOf(row).Elem().FieldByName(query.Weave.PrimaryField)
		if primaryField.IsValid() && primaryField.IsZero() {
			switch primaryField.Type().String() {
			case "int", "int8", "int16", "int32", "int64":
				id, err := result.Result.LastInsertId()
				if err != nil {
					result.Error = err
					return result
				}
				primaryField.Set(reflect.ValueOf(id))
			}
		}
	}

	return result
}

func (query *QueryStream[T]) InsertMap(data map[string]any) *QueryResultStreamer[T] {
	result := &QueryResultStreamer[T]{
		Error:       query.Error,
		WeaveConfig: query.Weave.Config,
	}
	result.Value, _ = query.Weave.ScanMap(data)
	if result.Error != nil {
		return result
	}

	db := Database()
	if db == nil {
		result.Error = UseDatabaseError{}
		return result
	}
	query.detectDialect()
	query.configure()
	queryString, args, err := query.dialect.BuildInsert(query.Config, data, maps.Keys(data)...)
	if err != nil {
		result.Error = err
		return result
	}
	result.Result, result.Error = query.dbExec(db, queryString, args...)
	return result
}

func (query *QueryStream[T]) Join(table string, clauses ...any) *QueryStream[T] {
	if query.Error != nil {
		return query
	}
	flat := make([]FilterClause, 0)
	for _, clause := range clauses {
		flat = flattenFilterClause(flat, clause)
	}

	query.Config.Joins = append(query.Config.Joins, JoinClause{
		Direction: "INNER",
		On:        flat,
		Table:     table,
	})
	return query
}

func (query *QueryStream[T]) JoinFull(table string, clauses ...any) *QueryStream[T] {
	if query.Error != nil {
		return query
	}
	flat := make([]FilterClause, 0)
	for _, clause := range clauses {
		flat = flattenFilterClause(flat, clause)
	}

	query.Config.Joins = append(query.Config.Joins, JoinClause{
		Direction: "FULL",
		On:        flat,
		Table:     table,
	})
	return query
}

func (query *QueryStream[T]) JoinLeft(table string, clauses ...any) *QueryStream[T] {
	if query.Error != nil {
		return query
	}
	flat := make([]FilterClause, 0)
	for _, clause := range clauses {
		flat = flattenFilterClause(flat, clause)
	}

	query.Config.Joins = append(query.Config.Joins, JoinClause{
		Direction: "LEFT",
		On:        flat,
		Table:     table,
	})
	return query
}

func (query *QueryStream[T]) JoinRight(table string, clauses ...any) *QueryStream[T] {
	if query.Error != nil {
		return query
	}
	flat := make([]FilterClause, 0)
	for _, clause := range clauses {
		flat = flattenFilterClause(flat, clause)
	}

	query.Config.Joins = append(query.Config.Joins, JoinClause{
		Direction: "RIGHT",
		On:        flat,
		Table:     table,
	})
	return query
}

func (query *QueryStream[T]) Limit(limit any) *QueryStream[T] {
	if query.Error == nil {
		query.Config.Limit = limit
	}
	return query
}

func (query *QueryStream[T]) Offset(offset any) *QueryStream[T] {
	if query.Error == nil {
		query.Config.Offset = offset
	}
	return query
}

func (query *QueryStream[T]) Scan(rows *sql.Rows) (*T, error) {
	data, err := query.ScanToMap(rows)
	if err != nil {
		return nil, err
	}
	return query.Weave.ScanMap(data)
}

func (query *QueryStream[T]) ScanToMap(rows *sql.Rows) (map[string]any, error) {
	return ScanFieldsToMap(rows, query.Weave.Fields)
}

func (query *QueryStream[T]) Select(columns ...any) *QueryStream[T] {
	if query.Error == nil {
		query.Config.Selected = columns
	}
	return query
}

func (query *QueryStream[T]) slice() ([]*T, error) {
	rows := make([]*T, 0)
	if query.Error != nil {
		return rows, query.Error
	}
	defer query.Rows.Close()

	relatedPks := make(map[string]relatedPk)
	for query.Rows.Next() {
		row, err := query.Scan(query.Rows)
		if err != nil {
			return rows, err
		}
		if len(query.Config.FetchRelated) > 0 {
			value := reflect.ValueOf(row).Elem()
			for _, column := range query.Config.FetchRelated {
				valueFk := value.FieldByName(column)
				if !valueFk.IsValid() {
					return rows, fmt.Errorf("trance: invalid field '%s' for fetching related. Field does not exist on model", column)
				}
				if strings.HasPrefix(valueFk.Type().String(), "trance.ForeignKey[") || strings.HasPrefix(valueFk.Type().String(), "trance.NullForeignKey[") {
					if valueFk.FieldByName("Valid").Interface().(bool) {
						r := reflect.New(valueFk.Type()).MethodByName("Weave").Call(nil)
						rpk, ok := relatedPks[column]
						if !ok {
							rpk = relatedPk{
								RelatedColumn: reflect.Indirect(r[0]).FieldByName("PrimaryColumn").Interface().(string),
								RelatedField:  reflect.Indirect(r[0]).FieldByName("PrimaryField").Interface().(string),
								RelatedValues: make([]any, 0),
							}
						}
						rpk.RelatedValues = append(rpk.RelatedValues, valueFk.FieldByName("Row").Elem().FieldByName(rpk.RelatedField).Interface())
						relatedPks[column] = rpk
					}
				} else if strings.HasPrefix(valueFk.Type().String(), "trance.OneToMany[") {
					rpk, ok := relatedPks[column]
					if !ok {
						relatedColumn := valueFk.FieldByName("RelatedColumn").Interface().(string)
						r := reflect.New(valueFk.Type()).MethodByName("Weave").Call(nil)
						fkModelFields := reflect.Indirect(r[0]).FieldByName("Fields").MapRange()
						var relatedField string
						for fkModelFields.Next() {
							fkModelField := fkModelFields.Value().FieldByName("Tag").MethodByName("Get").Call([]reflect.Value{reflect.ValueOf("@")})[0].Interface().(string)
							if fkModelField == relatedColumn {
								relatedField = fkModelFields.Value().FieldByName("Name").Interface().(string)
								break
							}
						}

						if relatedField == "" {
							return rows, fmt.Errorf("trance: invalid db tag of '%s' for fetching related on field '%s'. No fields with a matching column exist on the related model", relatedColumn, column)
						}

						rpk = relatedPk{
							RelatedColumn: relatedColumn,
							RelatedField:  relatedField,
							RelatedValues: make([]any, 0),
						}
					}
					rpk.RelatedValues = append(rpk.RelatedValues, value.FieldByName(query.Weave.PrimaryField).Interface())
					relatedPks[column] = rpk
				} else {
					return rows, fmt.Errorf("trance: invalid field '%s' for fetching related. Field must be of type trance.ForeignKey[To], trance.NullForeignKey[To], or trance.OneToMany[To, From]", column)
				}
			}
		}
		rows = append(rows, row)
	}

	if len(relatedPks) > 0 {
		var temp T
		modelValue := reflect.ValueOf(&temp).Elem()

		for column, rpk := range relatedPks {
			if len(rpk.RelatedValues) > 0 {
				fk := reflect.New(modelValue.FieldByName(column).Type())

				q := fk.MethodByName("Query").Call(nil)
				q = q[0].MethodByName("Filter").Call([]reflect.Value{
					reflect.ValueOf(rpk.RelatedColumn),
					reflect.ValueOf("IN"),
					reflect.ValueOf(rpk.RelatedValues),
				})
				q = q[0].MethodByName("Collect").Call([]reflect.Value{})
				rowsValue := reflect.ValueOf(rows)
				for i := 0; i < rowsValue.Len(); i++ {
					value := rowsValue.Index(i).Elem()
					valueFk := value.FieldByName(column)

					if strings.HasPrefix(fk.Type().String(), "*trance.OneToMany[") {
						mq := fk.MethodByName("Weave").Call(nil)
						fkPrimaryField := reflect.Indirect(mq[0]).FieldByName("PrimaryField").Interface().(string)
						for j := 0; j < q[0].Len(); j++ {
							fkRow := q[0].Index(j).Elem()
							relatedFieldId := fkRow.FieldByName(rpk.RelatedField).FieldByName("Row").Elem().FieldByName(fkPrimaryField).Interface()
							if value.FieldByName(query.Weave.PrimaryField).Interface() == relatedFieldId {
								valueFk.FieldByName("Rows").Set(reflect.Append(valueFk.FieldByName("Rows"), fkRow.Addr()))
							}
						}
					} else if valueFk.FieldByName("Valid").Interface().(bool) {
						mq := fk.MethodByName("Weave").Call(nil)
						fkPrimaryField := reflect.Indirect(mq[0]).FieldByName("PrimaryField").Interface().(string)
						for j := 0; j < q[0].Len(); j++ {
							fkRow := q[0].Index(j)
							if valueFk.FieldByName("Row").Elem().FieldByName(query.Weave.PrimaryField).Interface() == fkRow.Elem().FieldByName(fkPrimaryField).Interface() {
								valueFk.FieldByName("Row").Set(fkRow)
								break
							}
						}
					}
				}
			}
		}
	}

	if query.Config.Context != nil {
		select {
		default:
		case <-query.Config.Context.Done():
			return nil, query.Config.Context.Err()
		}
	}

	return rows, nil
}

func (query *QueryStream[T]) Sort(columns ...string) *QueryStream[T] {
	if query.Error == nil {
		query.Config.Sort = columns
	}
	return query
}

func (query *QueryStream[T]) SqlAll(sql string, args ...any) *WeaveListStreamer[T] {
	result := &WeaveListStreamer[T]{
		Error:       query.Error,
		WeaveConfig: query.Weave.Config,
	}
	if result.Error != nil {
		return result
	}

	db := Database()
	if db == nil {
		result.Error = UseDatabaseError{}
		return result
	}
	var err error
	query.Rows, err = db.Query(sql, args...)
	if err != nil {
		result.Error = err
		return result
	}
	result.Values, result.Error = query.slice()
	return result
}

func (query *QueryStream[T]) SqlAllToMap(sql string, args ...any) *MapListStream {
	result := &MapListStream{
		Error: query.Error,
	}
	if result.Error != nil {
		return result
	}

	db := Database()
	if db == nil {
		result.Error = UseDatabaseError{}
		return result
	}
	var err error
	query.Rows, err = db.Query(sql, args...)
	if err != nil {
		result.Error = err
		return result
	}
	defer query.Rows.Close()

	mapped := make([]map[string]any, 0)
	for query.Rows.Next() {
		data, err := query.ScanToMap(query.Rows)
		if err != nil {
			result.Error = err
			return result
		}
		mapped = append(mapped, data)
	}

	result.Values = mapped
	return result
}

func (query QueryStream[T]) StringWithArgs(dialect Dialect, args []any) (string, []any, error) {
	query.dialect = dialect
	query.configure()
	query.Config.Params = args
	return query.dialect.BuildSelect(query.Config)
}

func (query *QueryStream[T]) TableColumnAdd(column string) *QueryResultStreamer[T] {
	result := &QueryResultStreamer[T]{
		Error:       query.Error,
		WeaveConfig: query.Weave.Config,
	}
	if result.Error != nil {
		return result
	}

	db := Database()
	if db == nil {
		result.Error = UseDatabaseError{}
		return result
	}
	query.detectDialect()
	query.configure()
	queryString, err := query.dialect.BuildTableColumnAdd(query.Config, column)
	if err != nil {
		result.Error = err
		return result
	}
	result.Result, result.Error = query.dbExec(db, queryString)
	return result
}

func (query *QueryStream[T]) TableColumnDrop(column string) *QueryResultStreamer[T] {
	result := &QueryResultStreamer[T]{
		Error:       query.Error,
		WeaveConfig: query.Weave.Config,
	}
	if result.Error != nil {
		return result
	}

	db := Database()
	if db == nil {
		result.Error = UseDatabaseError{}
		return result
	}
	query.detectDialect()
	query.configure()
	queryString, err := query.dialect.BuildTableColumnDrop(query.Config, column)
	if err != nil {
		result.Error = err
		return result
	}
	result.Result, result.Error = query.dbExec(db, queryString)
	return result
}

func (query *QueryStream[T]) TableCreate(tableCreateConfig ...TableCreateConfig) *QueryResultStreamer[T] {
	result := &QueryResultStreamer[T]{
		Error:       query.Error,
		WeaveConfig: query.Weave.Config,
	}
	if result.Error != nil {
		return result
	}

	db := Database()
	if db == nil {
		result.Error = UseDatabaseError{}
		return result
	}
	query.detectDialect()
	query.configure()
	var config TableCreateConfig
	if len(tableCreateConfig) > 0 {
		config = tableCreateConfig[0]
	}
	queryString, err := query.dialect.BuildTableCreate(query.Config, config)
	if err != nil {
		result.Error = err
		return result
	}
	result.Result, result.Error = query.dbExec(db, queryString)
	return result
}

func (query *QueryStream[T]) TableDrop(tableDropConfig ...TableDropConfig) *QueryResultStreamer[T] {
	result := &QueryResultStreamer[T]{
		Error:       query.Error,
		WeaveConfig: query.Weave.Config,
	}
	if result.Error != nil {
		return result
	}

	db := Database()
	if db == nil {
		result.Error = UseDatabaseError{}
		return result
	}
	query.detectDialect()
	query.configure()
	var config TableDropConfig
	if len(tableDropConfig) > 0 {
		config = tableDropConfig[0]
	}
	queryString, err := query.dialect.BuildTableDrop(query.Config, config)
	if err != nil {
		result.Error = err
		return result
	}
	result.Result, result.Error = query.dbExec(db, queryString)
	return result
}

func (query *QueryStream[T]) Transaction(transaction *sql.Tx) *QueryStream[T] {
	query.Config.Transaction = transaction
	return query
}

func (query *QueryStream[T]) Update(row *T, columns ...string) *QueryResultStreamer[T] {
	result := &QueryResultStreamer[T]{
		Error:       query.Error,
		Value:       row,
		WeaveConfig: query.Weave.Config,
	}
	if result.Error != nil {
		return result
	}

	db := Database()
	if db == nil {
		result.Error = UseDatabaseError{}
		return result
	}
	query.detectDialect()
	query.configure()

	if len(columns) == 0 {
		result.Error = fmt.Errorf("trance: no columns specified for update")
		return result
	}

	rowMap, err := query.Weave.ToMap(row)
	if err != nil {
		result.Error = err
		return result
	}

	queryString, args, err := query.dialect.BuildUpdate(query.Config, rowMap, columns...)
	if err != nil {
		result.Error = err
		return result
	}
	result.Result, result.Error = query.dbExec(db, queryString, args...)
	return result
}

func (query *QueryStream[T]) UpdateMap(data map[string]any) *QueryResultStreamer[T] {
	result := &QueryResultStreamer[T]{
		Error:       query.Error,
		WeaveConfig: query.Weave.Config,
	}
	result.Value, _ = query.Weave.ScanMap(data)
	if result.Error != nil {
		return result
	}

	db := Database()
	if db == nil {
		result.Error = UseDatabaseError{}
		return result
	}
	query.detectDialect()
	query.configure()

	if len(data) == 0 {
		result.Error = fmt.Errorf("trance: no columns specified for update")
		return result
	}

	columns := make([]string, 0)
	for column := range data {
		columns = append(columns, column)
	}

	queryString, args, err := query.dialect.BuildUpdate(query.Config, data, columns...)
	if err != nil {
		result.Error = err
		return result
	}
	result.Result, result.Error = query.dbExec(db, queryString, args...)
	return result
}

func (query *QueryStream[T]) ViewSelect(ctx context.Context) *QueryViewStream[T] {
	result := &QueryViewStream[T]{
		Context:     ctx,
		Error:       query.Error,
		Query:       query,
		WeaveConfig: query.Weave.Config,
	}
	if result.Error != nil {
		return result
	}

	zero := query.Weave.Zero()
	if temp, ok := any(zero).(Viewer); ok {
		result.View = temp.ViewSelect(result.Context)
	} else {
		result.Error = ErrorInternalServer{Message: fmt.Sprintf("trance: model '%T' does not implement trance.Viewer", zero)}
	}

	return result
}

type relatedPk struct {
	RelatedColumn string
	RelatedField  string
	RelatedValues []any
}

type TableCreateConfig struct {
	IfNotExists bool
}

type TableDropConfig struct {
	IfExists bool
}

type UseDatabaseError struct{}

func (err UseDatabaseError) Error() string {
	return "trance: missing database connection. Register with `trance.UseDatabase(db *sql.DB)`"
}

func Query[T any]() *QueryStream[T] {
	return &QueryStream[T]{
		Weave: Use[T](),
	}
}

func QueryWith[T any](config WeaveConfig) *QueryStream[T] {
	return &QueryStream[T]{
		Weave: UseWith[T](config),
	}
}
