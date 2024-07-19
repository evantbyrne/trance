package trance

import (
	"bytes"
	"context"
	_ "embed"
	"html/template"
	"io"
	"maps"
	"net/http"
	"reflect"
	"slices"
	"strings"
)

//go:embed templates/forms/foreign_key.gohtml
var FormForeignKeyHTML string

//go:embed templates/forms/hidden.gohtml
var FormHiddenHTML string

//go:embed templates/forms/select.gohtml
var FormSelectHTML string

//go:embed templates/forms/text.gohtml
var FormTextHTML string

//go:embed templates/forms/form.gohtml
var DefaultFormHTML string

var defaultFormTpl = template.Must(
	template.New("trance.form.render").
		Funcs(template.FuncMap{"hasPrefix": strings.HasPrefix}).
		Parse(DefaultFormHTML))

func init() {
	template.Must(defaultFormTpl.New("trance.form.foreign-key").Parse(FormForeignKeyHTML))
	template.Must(defaultFormTpl.New("trance.form.hidden").Parse(FormHiddenHTML))
	template.Must(defaultFormTpl.New("trance.form.select").Parse(FormSelectHTML))
	template.Must(defaultFormTpl.New("trance.form.text").Parse(FormTextHTML))
}

type Form[T any] struct {
	Action   string
	Error    error
	Fields   []string
	Method   string
	Template *template.Template
	Value    *T
	Weave    *Weave[T]
}

func (form *Form[T]) Render(writer io.Writer, additionalData ...map[string]any) error {
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

	data := FormTemplateData{
		Action:     form.Action,
		Data:       make(map[string]any),
		Errors:     errorsMap,
		Fields:     form.Fields,
		FieldTypes: form.Weave.Fields,
		Method:     form.Method,
		Values:     form.Weave.ToJsonMap(form.Value),
		ValuesMap:  form.Weave.ToValuesMap(form.Value),
	}
	for _, n := range additionalData {
		maps.Copy(data.Data, n)
	}

	if widgeter, ok := any(form.Value).(FormWidgeter); ok {
		// TODO: Context
		data.Widgets = widgeter.FormWidgets(context.Background())
	} else {
		data.Widgets = make(map[string]FormWidget, 0)
	}

	if form.Template == nil {
		form.Template = defaultFormTpl
	}
	return form.Template.Execute(writer, data)
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

type FormTemplateData struct {
	Action     string
	Data       map[string]any
	Errors     map[string]error
	Fields     []string
	FieldTypes map[string]reflect.StructField
	Method     string
	Values     map[string]any
	ValuesMap  map[string]any
	Widgets    map[string]FormWidget
}

func (ftd FormTemplateData) GetField(key string) FormTemplateDataField {
	return FormTemplateDataField{
		FormData: ftd,
		Key:      key,
	}
}

type FormTemplateDataField struct {
	FormData   FormTemplateData
	Key        string
	WidgetData map[string]any
}

func (field FormTemplateDataField) RenderWidget(widget FormWidget) template.HTML {
	writer := new(bytes.Buffer)
	if err := widget.Render(field, writer); err != nil {
		panic(err)
	}
	return template.HTML(writer.String())
}
