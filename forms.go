package trance

import (
	_ "embed"
	"html/template"
	"maps"
	"net/http"
	"reflect"
	"slices"

	"github.com/a-h/templ"
	"github.com/evantbyrne/trance/templates/forms"
)

type Form[T any] struct {
	Action   string
	Error    error
	Fields   []string
	Method   string
	Template *template.Template
	Value    *T
	Weave    *Weave[T]
	Widgets  map[string]forms.Widgeter
}

func (form *Form[T]) Component(additionalData ...map[string]any) templ.Component {
	if form.Weave == nil {
		form.Weave = Use[T]()
	}
	errorsMap := make(map[string]error, 0)
	if form.Error != nil {
		if ev, ok := form.Error.(FormErrors); ok {
			errorsMap = ev.Errors
		} else {
			errorsMap["_global_"] = form.Error
		}
	}
	if form.Method == "" {
		form.Method = "POST"
	}
	if form.Value == nil {
		form.Value = new(T)
	}

	// Order fields.
	if form.Fields == nil {
		form.Fields = make([]string, 0)
		for column := range form.Weave.Fields {
			form.Fields = append(form.Fields, column)
		}
	}

	data := forms.FormTemplateData{
		Action:     form.Action,
		Data:       make(map[string]any),
		Errors:     errorsMap,
		Fields:     form.Fields,
		FieldTypes: form.Weave.Fields,
		Method:     form.Method,
		Values:     form.Weave.ToJsonMap(form.Value),
		ValuesMap:  form.Weave.ToValuesMap(form.Value),
		Widgets:    form.Widgets,
	}
	for _, n := range additionalData {
		maps.Copy(data.Data, n)
	}

	return forms.Form(data)
}

func (form *Form[T]) Validate(request *http.Request) bool {
	if form.Weave == nil {
		form.Weave = Use[T]()
	}

	// Make sure we don't mutate cached weaves.
	temp := form.Weave.Clone()

	// Only validate included fields.
	if len(form.Fields) > 0 {
		for column := range temp.Fields {
			if !slices.Contains(form.Fields, column) {
				delete(temp.Fields, column)
			}
		}
	}

	var value *T
	value, form.Error = temp.Validate(request)
	if form.Value == nil {
		form.Value = value
	} else if len(form.Fields) > 0 {
		// Preserve unmodified fields.
		fv := reflect.ValueOf(form.Value).Elem()
		vv := reflect.ValueOf(value).Elem()

		for _, column := range form.Fields {
			field := temp.Fields[column]
			newValue := vv.FieldByName(field.Name)
			fv.FieldByName(field.Name).Set(newValue)
		}
	}

	return form.Error == nil
}
