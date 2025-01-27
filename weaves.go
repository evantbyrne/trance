package trance

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"
)

type Weave[T any] struct {
	Config        WeaveConfig
	Fields        map[string]reflect.StructField
	PrimaryColumn string
	PrimaryField  string
	Table         string
	Type          reflect.Type
}

func (weave *Weave[T]) Clone() *Weave[T] {
	return &Weave[T]{
		Fields:        maps.Clone(weave.Fields),
		PrimaryColumn: weave.PrimaryColumn,
		PrimaryField:  weave.PrimaryField,
		Table:         weave.Table,
		Type:          weave.Type,
	}
}

func (weave *Weave[T]) ScanMap(data map[string]any) (*T, error) {
	var row T
	value := reflect.ValueOf(&row).Elem()

	for column, v := range data {
		if field, ok := weave.Fields[column]; ok {
			if field := value.FieldByName(field.Name); field.IsValid() {
				columnValue := reflect.ValueOf(v)

				if v == nil {
					// database/sql null types (NullString, etc) default to `Valid: false`.
					// trance.ForeignKey and trance.NullForeignKey also follow this convention.

				} else if columnValue.CanConvert(field.Type()) {
					field.Set(columnValue)

				} else if field.Kind() == reflect.Struct {
					if scanner, ok := reflect.New(field.Type()).Interface().(sql.Scanner); ok {
						scanner.Scan(v)
						field.Set(reflect.ValueOf(scanner).Elem())

					} else if strings.HasPrefix(field.Type().String(), "trance.ForeignKey[") || strings.HasPrefix(field.Type().String(), "trance.NullForeignKey[") {
						subModelQ := field.Addr().MethodByName("Weave").Call(nil)
						subPrimaryField := reflect.Indirect(subModelQ[0]).FieldByName("PrimaryField").Interface().(string)
						subField := field.FieldByName("Row").Elem().FieldByName(subPrimaryField)
						if subField.IsValid() {
							// TODO: Handle primary keys that are nullable types
							subField.Set(columnValue)
							field.FieldByName("Valid").SetBool(true)
						}
					} else {
						return nil, fmt.Errorf("trance: unhandled struct conversion in scan from '%s' to '%s'", columnValue.Type(), field.Type())
					}

				} else {
					return nil, fmt.Errorf("trance: unhandled type conversion in scan from '%s' to '%s'", columnValue.Type(), field.Type())
				}
			}
		}
	}

	// OneToMany relationships.
	for _, field := range weave.Fields {
		if strings.HasPrefix(field.Type.String(), "trance.OneToMany[") {
			oneToMany := value.FieldByName(field.Name)
			oneToMany.FieldByName("RelatedColumn").SetString(field.Tag.Get("@"))
			oneToMany.FieldByName("RowPk").Set(value.FieldByName(weave.PrimaryField))
		}
	}

	return &row, nil
}

func (weave *Weave[T]) ToJsonMap(row *T) map[string]any {
	result := make(map[string]any, 0)
	value := reflect.ValueOf(row).Elem()
	for _, field := range weave.Fields {
		fieldName := strings.ToLower(field.Name)
		switch fv := value.FieldByName(field.Name).Interface().(type) {
		case JsonValuer:
			result[fieldName] = fv.JsonValue()

		case driver.Valuer:
			result[fieldName], _ = fv.Value()

		default:
			result[fieldName] = fv
		}
	}

	return result
}

func (weave *Weave[T]) ToMap(row *T) (map[string]any, error) {
	args := make(map[string]any)
	value := reflect.ValueOf(*row)

	for column := range weave.Fields {
		fieldName := weave.Fields[column].Name
		field := value.FieldByName(fieldName)

		// Skip zero valued primary keys.
		if field.IsZero() && weave.Fields[column].Tag.Get("@primary") == "true" {
			continue
		}

		switch field.Kind() {
		case reflect.Struct:
			switch vv := field.Interface().(type) {
			case driver.Valuer:
				v, _ := vv.Value()
				args[column] = v

			case time.Time:
				args[column] = vv

			default:
				if strings.HasPrefix(field.Type().String(), "trance.ForeignKey[") || strings.HasPrefix(field.Type().String(), "trance.NullForeignKey[") {
					if !field.FieldByName("Valid").Interface().(bool) {
						args[column] = nil
					} else {
						q := reflect.New(field.Type()).MethodByName("Weave").Call(nil)
						fkPrimaryField := reflect.Indirect(q[0]).FieldByName("PrimaryField").Interface().(string)
						args[column] = reflect.Indirect(field.FieldByName("Row")).FieldByName(fkPrimaryField).Interface()
					}
				} else if strings.HasPrefix(field.Type().String(), "trance.OneToMany[") {
					continue
				} else {
					return nil, fmt.Errorf("trance: unsupported field type '%s' for column '%s' on table '%s'", field.Type().String(), column, weave.Table)
				}
			}

		default:
			args[column] = field.Interface()
		}
	}

	return args, nil
}

func (weave *Weave[T]) ToValuesMap(row *T) map[string]any {
	result := make(map[string]any, 0)
	value := reflect.ValueOf(row).Elem()
	for column, field := range weave.Fields {
		result[column] = value.FieldByName(field.Name).Interface()
	}
	return result
}

func (weave *Weave[T]) Validate(r *http.Request) (*T, error) {
	// TODO: Support TOML, YAML, XML, and gRPC content types.
	switch r.Header.Get("Content-Type") {
	case "application/x-www-form-urlencoded", "multipart/form-data":
		return weave.ValidateEncoded(r)
	case "application/json":
		return weave.ValidateJson(r)
	}
	return nil, ErrorBadRequest{Message: "Unsupported content type"}
}

func (weave *Weave[T]) ValidateEncoded(r *http.Request) (*T, error) {
	errorsMap := make(map[string]error, 0)
	value := make(map[string]any, 0)

	for column := range weave.Fields {
		fv := r.FormValue(column)
		if fv == "" {
			errorsMap[column] = errors.New("This field is required.")
		} else {
			value[column] = fv
		}
	}

	record, err := weave.ScanMap(value)
	if err != nil {
		errorsMap["_global_"] = err
	}

	if len(errorsMap) > 0 {
		return record, FormErrors{
			Errors:     errorsMap,
			StatusCode: http.StatusBadRequest,
		}
	}

	return record, nil
}

func (weave *Weave[T]) ValidateJson(r *http.Request) (*T, error) {
	var unmarshalErr *json.UnmarshalTypeError
	var record T

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	errorsMap := make(map[string]error, 0)
	if err := decoder.Decode(&record); err != nil {
		if errors.As(err, &unmarshalErr) {
			for column := range weave.Fields {
				if weave.Fields[column].Name == unmarshalErr.Field {
					errorsMap[column] = errors.New("Wrong type provided.")
					break
				}
			}
		} else {
			if err.Error() == "EOF" {
				// Allow missing JSON bodies for forms that don't have any fields.
				if len(weave.Fields) > 0 {
					errorsMap["_global_"] = errors.New("JSON body required.")
				}
			} else {
				errorsMap["_global_"] = err
			}
		}
	}

	value := reflect.ValueOf(record)
	for column := range weave.Fields {
		if _, exists := errorsMap[column]; !exists {
			if value.FieldByName(weave.Fields[column].Name).IsZero() {
				errorsMap[column] = errors.New("This field is required.")
			}
		}
	}

	if len(errorsMap) > 0 {
		return &record, FormErrors{
			Errors:     errorsMap,
			StatusCode: http.StatusBadRequest,
		}
	}

	return &record, nil
}

func (weave *Weave[T]) ValidateParams(r *http.Request) (map[string]any, error) {
	data := make(map[string]any, 0)
	errorsMap := make(map[string]error, 0)
	for column := range weave.Fields {
		value := r.URL.Query().Get(column)
		if value == "" {
			errorsMap[column] = errors.New("This field is required.")
			if r.URL.Query().Has(column) {
				data[column] = value
			}
		} else {
			// TODO: Scanning.
			data[column] = value
		}
	}

	if len(errorsMap) > 0 {
		return data, FormErrors{
			Errors:     errorsMap,
			StatusCode: http.StatusBadRequest,
		}
	}

	return data, nil
}

func (weave *Weave[T]) Zero() T {
	var zero T
	return zero
}

type WeaveConfig struct {
	NoCache bool
	Table   string
}

type WeaveConfigurable interface {
	WeaveConfig() WeaveConfig
}

var weavesCache = &sync.Map{}

func PurgeWeaves() {
	weavesCache.Range(func(key, value any) bool {
		weavesCache.Delete(key)
		return true
	})
}

func Transform[From any, To any](from *From) (*To, error) {
	data, err := Use[From]().ToMap(from)
	if err != nil {
		return nil, err
	}
	return UseWith[To](WeaveConfig{}).ScanMap(data)
}

func TransformWith[From any, To any](from *From, toConfig WeaveConfig) (*To, error) {
	data, err := Use[From]().ToMap(from)
	if err != nil {
		return nil, err
	}
	return UseWith[To](toConfig).ScanMap(data)
}

func Use[T any]() *Weave[T] {
	var model T
	if weaveConfigurable, ok := any(model).(WeaveConfigurable); ok {
		return UseWith[T](weaveConfigurable.WeaveConfig())
	}
	return UseWith[T](WeaveConfig{})
}

func UseWith[T any](config WeaveConfig) *Weave[T] {
	var model T
	modelType := reflect.TypeOf(model)
	modelTypeStr := fmt.Sprintf("%s%+v", modelType.String(), config)

	if !config.NoCache {
		if existing, ok := weavesCache.Load(modelTypeStr); ok {
			if weave, ok := existing.(*Weave[T]); ok {
				return weave
			}
		}
	}

	var primaryColumn string
	var primaryField string
	fields := make(map[string]reflect.StructField, 0)

	for _, field := range reflect.VisibleFields(modelType) {
		if column, ok := field.Tag.Lookup("@"); ok {
			if strings.HasPrefix(field.Type.String(), "trance.OneToMany[") {
				fields[field.Name] = field
			} else {
				fields[column] = field
				if field.Tag.Get("@primary") == "true" {
					primaryColumn = column
					primaryField = field.Name
				}
			}
		}
	}

	weave := &Weave[T]{
		Config:        config,
		Fields:        fields,
		PrimaryColumn: primaryColumn,
		PrimaryField:  primaryField,
		Type:          modelType,
	}
	if config.Table == "" {
		weave.Table = strings.ToLower(modelType.Name())
	} else {
		weave.Table = config.Table
	}
	if !config.NoCache {
		weavesCache.Store(modelTypeStr, weave)
	}
	return weave
}

func ValidateTransform[From any, To any](r *http.Request) (*To, error) {
	return ValidateTransformWith[From, To](r, WeaveConfig{})
}

func ValidateTransformWith[From any, To any](r *http.Request, toConfig WeaveConfig) (*To, error) {
	from, err := Use[From]().Validate(r)
	if err != nil {
		return nil, err
	}
	return TransformWith[From, To](from, toConfig)
}
