package trance

import (
	"fmt"
	"html/template"
	"net/http"
)

type HtmlRenderer[T Viewer] struct {
	Template *template.Template
}

func (renderer HtmlRenderer[T]) RenderError(w http.ResponseWriter, r *http.Request, err error) {
	message := err.Error()
	status := http.StatusInternalServerError

	if ev, ok := err.(ErrorWithStatus); ok {
		message = ev.Error()
		status = ev.Status()
	}

	if status == http.StatusInternalServerError {
		fmt.Println(err)
		message = "Internal server error"
	}

	w.WriteHeader(status)
	w.Write([]byte(message))
}

func (renderer HtmlRenderer[T]) RenderFind(w http.ResponseWriter, r *http.Request, weave *Weave[T], view *View, record *T) {
	// Guard Select.
	data, err := Guard(requestToGuardContext(r), record)
	if err != nil {
		renderer.RenderError(w, r, err)
		return
	}

	if err := renderer.Template.Execute(w, data); err != nil {
		renderer.RenderError(w, r, ErrorInternalServer{Message: "trance: " + err.Error()})
	}
}

func (renderer HtmlRenderer[T]) RenderList(w http.ResponseWriter, r *http.Request, weave *Weave[T], view *View, records []*T) {
	// Guard and convert into serializable format.
	data, err := GuardList(requestToGuardContext(r), records)
	if err != nil {
		renderer.RenderError(w, r, err)
		return
	}

	if err := renderer.Template.Execute(w, data); err != nil {
		renderer.RenderError(w, r, ErrorInternalServer{Message: "trance: " + err.Error()})
	}
}

func FindHtml[T Viewer](templatePath string) func(*Strand) error {
	return Find[T](Html[T](templatePath))
}

func Html[T Viewer](templatePath string) HtmlRenderer[T] {
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		panic("trance: " + err.Error())
	}
	return HtmlRenderer[T]{Template: tmpl}
}

func ListHtml[T Viewer](templatePath string) func(*Strand) error {
	return List[T](Html[T](templatePath))
}
