package trance

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/evantbyrne/trance/templates/forms"
)

type ForeignKey[To any] struct {
	Row   *To
	Valid bool
}

func (fk *ForeignKey[To]) Fetch() (*To, error) {
	query := Query[To]()
	value := reflect.ValueOf(fk.Row).Elem()
	id := value.FieldByName(query.Weave.PrimaryField).Interface()
	return query.Filter("id", "=", id).CollectFirst()
}

func (fk ForeignKey[To]) JsonValue() any {
	if !fk.Valid {
		return nil
	}
	return fk.Weave().ToJsonMap(fk.Row)
}

func (fk ForeignKey[To]) MarshalJSON() ([]byte, error) {
	if !fk.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(fk.Weave().ToJsonMap(fk.Row))
}

func (fk ForeignKey[To]) String() string {
	if !fk.Valid {
		return ""
	}
	return fmt.Sprint(fk.Row)
}

func (fk *ForeignKey[To]) Weave() *Weave[To] {
	if fk.Row == nil {
		var zero To
		fk.Row = &zero
	}
	return Use[To]()
}

func (fk ForeignKey[To]) WidgetOptions() ([]forms.WidgetOption, error) {
	options := make([]forms.WidgetOption, 0)
	weave := fk.Weave()

	err := fk.Query().All().ForEach(func(_ int, value *To) error {
		options = append(options, forms.WidgetOption{
			Label: fmt.Sprint(value),
			Value: weave.ToValuesMap(value)[weave.PrimaryColumn],
		})
		return nil
	}).Error

	return options, err
}

func (fk ForeignKey[To]) Query() *QueryStream[To] {
	return Query[To]()
}

type NullForeignKey[To any] struct {
	Row   *To
	Valid bool
}

func (fk *NullForeignKey[To]) Fetch() (*To, error) {
	query := Query[To]()
	value := reflect.ValueOf(fk.Row).Elem()
	id := value.FieldByName(query.Weave.PrimaryField).Interface()
	return query.Filter("id", "=", id).CollectFirst()
}

func (fk NullForeignKey[To]) JsonValue() any {
	if !fk.Valid {
		return nil
	}
	return fk.Weave().ToJsonMap(fk.Row)
}

func (fk NullForeignKey[To]) MarshalJSON() ([]byte, error) {
	if !fk.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(fk.Weave().ToJsonMap(fk.Row))
}

func (fk NullForeignKey[To]) String() string {
	if !fk.Valid {
		return ""
	}
	return fmt.Sprint(fk.Row)
}

func (fk *NullForeignKey[To]) Weave() *Weave[To] {
	if fk.Row == nil {
		var zero To
		fk.Row = &zero
	}
	return Use[To]()
}

func (fk NullForeignKey[To]) WidgetOptions() ([]forms.WidgetOption, error) {
	options := make([]forms.WidgetOption, 0)
	options = append(options, forms.WidgetOption{})
	weave := fk.Weave()

	err := fk.Query().All().ForEach(func(_ int, value *To) error {
		options = append(options, forms.WidgetOption{
			Label: fmt.Sprint(value),
			Value: weave.ToValuesMap(value)[weave.PrimaryColumn],
		})
		return nil
	}).Error

	return options, err
}

func (fk NullForeignKey[To]) Query() *QueryStream[To] {
	return Query[To]()
}
