package trance

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
	"time"
)

func ScanFieldsToMap(rows *sql.Rows, fields map[string]reflect.StructField) (map[string]any, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	pointers := make([]any, len(columns))
	for i, column := range columns {
		field, ok := fields[column]
		if !ok {
			return nil, fmt.Errorf("trance: column '%s' not found on struct map '%#v'", column, fields)
		}
		fieldType := field.Type
		if strings.HasPrefix(fieldType.String(), "trance.ForeignKey[") || strings.HasPrefix(fieldType.String(), "trance.NullForeignKey[") {
			fk := reflect.New(fieldType)
			q := fk.MethodByName("Weave").Call(nil)
			fkPrimaryField := reflect.Indirect(q[0]).FieldByName("PrimaryField").Interface().(string)
			pointers[i] = reflect.New(reflect.Indirect(reflect.Indirect(fk).FieldByName("Row")).FieldByName(fkPrimaryField).Type()).Interface()
			if strings.HasPrefix(fieldType.String(), "trance.NullForeignKey[") {
				switch pointers[i].(type) {
				case *bool:
					pointers[i] = new(sql.NullBool)
				case *byte:
					pointers[i] = new(sql.NullByte)
				case *int, *int64:
					pointers[i] = new(sql.NullInt64)
				case *int32:
					pointers[i] = new(sql.NullInt32)
				case *int8, *int16:
					pointers[i] = new(sql.NullInt16)
				case *float32, *float64:
					pointers[i] = new(sql.NullFloat64)
				case *string:
					pointers[i] = new(sql.NullString)
				case *time.Time:
					pointers[i] = new(sql.NullTime)
				}
			}
		} else {
			pointers[i] = reflect.New(fieldType).Interface()
		}
	}

	if err := rows.Scan(pointers...); err != nil {
		return nil, err
	}

	row := make(map[string]any)
	for i, column := range columns {
		switch vt := reflect.ValueOf(pointers[i]).Elem().Interface().(type) {
		case driver.Valuer:
			row[column], _ = vt.Value()
		default:
			row[column] = vt
		}
	}

	return row, nil
}
