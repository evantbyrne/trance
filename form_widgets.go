package trance

import (
	"context"
	"errors"
	"io"
)

type FormWidget interface {
	Render(FormTemplateDataField, io.Writer) error
}

type FormWidgeter interface {
	FormWidgets(context.Context) map[string]FormWidget
}

type FormWidgetSelect struct {
	Options []FormWidgetSelectOption
}

func (widget FormWidgetSelect) Render(field FormTemplateDataField, writer io.Writer) error {
	tpl := defaultFormTpl.Lookup("trance.form.select")
	if tpl == nil {
		return errors.New("trance: template 'trance.form.select' not found on (FormWidgetSelect).Render()")
	}
	field.WidgetData = map[string]any{
		"Options": widget.Options,
	}
	return tpl.Execute(writer, field)
}

type FormWidgetSelectOption struct {
	Label string
	Value string
}
