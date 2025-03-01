package forms

import "github.com/a-h/templ"

type WidgetOption struct {
	Label    string
	Selected bool
	Value    any
}

type WidgetOptioner interface {
	WidgetOptions() ([]WidgetOption, error)
}

type Widgeter func(fieldData FormTemplateDataField) templ.Component
