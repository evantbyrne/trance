package trance

import (
	"fmt"
	"net/http"
)

type JsonRenderer[T Viewer] struct{}

func (renderer JsonRenderer[T]) RenderError(w http.ResponseWriter, r *http.Request, err error) {
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	responseJson(w, map[string]any{"error": message})
}

func (renderer JsonRenderer[T]) RenderFind(w http.ResponseWriter, r *http.Request, weave *Weave[T], view *View, record *T) {
	w.Header().Set("Content-Type", "application/json")

	// Guard and convert into serializable format.
	data, err := Guard(requestToGuardContext(r), record)
	if err != nil {
		renderer.RenderError(w, r, err)
		return
	}

	responseJson(w, map[string]any{"item": data})
}

func (renderer JsonRenderer[T]) RenderFormErrors(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Content-Type", "application/json")
	errorsMap := make(map[string]string, 0)
	if ev, ok := err.(FormErrors); ok {
		w.WriteHeader(ev.StatusCode)
		for key, err := range ev.Errors {
			errorsMap[key] = err.Error()
		}
	} else if ev, ok := err.(ErrorWithStatus); ok {
		w.WriteHeader(ev.Status())
		errorsMap["_global_"] = ev.Error()
	} else {
		w.WriteHeader(http.StatusBadRequest)
		errorsMap["_global_"] = err.Error()
	}
	responseJson(w, map[string]any{"errors": errorsMap})
}

func (renderer JsonRenderer[T]) RenderList(w http.ResponseWriter, r *http.Request, weave *Weave[T], view *View, records []*T) {
	w.Header().Set("Content-Type", "application/json")

	// Guard and convert into serializable format.
	data, err := GuardList(requestToGuardContext(r), records)
	if err != nil {
		renderer.RenderError(w, r, err)
		return
	}

	responseJson(w, map[string]any{"items": data})
}

func CreateJson[T Viewer, F any]() func(*Strand) error {
	return Create[T, F](Json[T]())
}

func DeleteJson[T Viewer, F any]() func(*Strand) error {
	return Delete[T, F](Json[T]())
}

func EditJson[T Viewer, F any]() func(*Strand) error {
	return Edit[T, F](Json[T]())
}

func FindJson[T Viewer]() func(*Strand) error {
	return Find[T](Json[T]())
}

func Json[T Viewer]() JsonRenderer[T] {
	return JsonRenderer[T]{}
}

func ListJson[T Viewer]() func(*Strand) error {
	return List[T](Json[T]())
}
