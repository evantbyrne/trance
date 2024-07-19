package trance

import (
	"fmt"
	"net/http"
)

type ComponentRenderer[T Viewer, C any] struct{}

func (renderer ComponentRenderer[T, C]) RenderError(w http.ResponseWriter, r *http.Request, err error) {
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

func (renderer ComponentRenderer[T, C]) RenderFind(w http.ResponseWriter, r *http.Request, weave *Weave[T], view *View, record *T) {
	// Guard Select.
	data, err := Guard(requestToGuardContext(r), record)
	if err != nil {
		renderer.RenderError(w, r, err)
		return
	}

	var componentTemp C
	if component, ok := any(componentTemp).(Renderer); ok {
		if err := component.Render(w, r, data); err != nil {
			renderer.RenderError(w, r, ErrorInternalServer{Message: "trance: " + err.Error()})
		}
	} else {
		renderer.RenderError(w, r, ErrorInternalServer{Message: "trance: Component does not implement trance.Renderer[T]"})
	}
}

func (renderer ComponentRenderer[T, C]) RenderList(w http.ResponseWriter, r *http.Request, weave *Weave[T], view *View, records []*T) {
	// Guard and convert into serializable format.
	data, err := GuardList(requestToGuardContext(r), records)
	if err != nil {
		renderer.RenderError(w, r, err)
		return
	}

	var componentTemp C
	if component, ok := any(componentTemp).(RenderLister); ok {
		if err := component.RenderList(w, r, data); err != nil {
			renderer.RenderError(w, r, ErrorInternalServer{Message: "trance: " + err.Error()})
		}
	} else {
		renderer.RenderError(w, r, ErrorInternalServer{Message: "trance: Component does not implement trance.RenderLister[T]"})
	}
}

type RenderLister interface {
	RenderList(http.ResponseWriter, *http.Request, []map[string]any) error
}

type Renderer interface {
	Render(http.ResponseWriter, *http.Request, map[string]any) error
}

func Component[T Viewer, C any]() ComponentRenderer[T, C] {
	return ComponentRenderer[T, C]{}
}

func FindComponent[T Viewer, R Renderer]() func(*Strand) error {
	return Find[T](Component[T, R]())
}

func ListComponent[T Viewer, R RenderLister]() func(*Strand) error {
	return List[T](Component[T, R]())
}
