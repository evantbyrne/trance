package trance

import (
	"encoding/json"
	"reflect"
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

func (fk *ForeignKey[To]) Weave() *Weave[To] {
	if fk.Row == nil {
		var zero To
		fk.Row = &zero
	}
	return Use[To]()
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

func (fk *NullForeignKey[To]) Weave() *Weave[To] {
	if fk.Row == nil {
		var zero To
		fk.Row = &zero
	}
	return Use[To]()
}

func (fk NullForeignKey[To]) Query() *QueryStream[To] {
	return Query[To]()
}
