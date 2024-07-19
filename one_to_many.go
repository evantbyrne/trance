package trance

import (
	"encoding/json"
)

type OneToMany[To any] struct {
	RelatedColumn string
	RowPk         any
	Rows          []*To
}

func (field *OneToMany[To]) All() ([]*To, error) {
	return field.Query().Filter(field.RelatedColumn, "=", field.RowPk).Collect()
}

func (field OneToMany[To]) JsonValue() any {
	weave := field.Weave()
	results := make([]map[string]any, len(field.Rows))
	for i := range field.Rows {
		results[i] = weave.ToJsonMap(field.Rows[i])
	}
	return results
}

func (field OneToMany[To]) MarshalJSON() ([]byte, error) {
	weave := field.Weave()
	results := make([]map[string]any, len(field.Rows))
	for i, row := range field.Rows {
		results[i] = weave.ToJsonMap(row)
	}
	return json.Marshal(results)
}

func (field *OneToMany[To]) Weave() *Weave[To] {
	return Use[To]()
}

func (field *OneToMany[To]) Query() *QueryStream[To] {
	return Query[To]()
}
