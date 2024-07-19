package trance

import (
	"context"
	"encoding/json"
	"net/http"
)

type App struct {
	ErrorHandler func(*Strand)
	Middleware   []func(*Strand) error
}

func (app *App) defaultErrorHandler(strand *Strand) {
	if strand.Error == nil {
		return
	}
	status := http.StatusInternalServerError
	if errWithStatus, ok := strand.Error.(ErrorWithStatus); ok {
		status = errWithStatus.Status()
	}

	if strand.Response.Header().Get("Content-Type") == "application/json" || strand.Request().Header.Get("Content-Type") == "application/json" {
		strand.Response.Header().Set("Content-Type", "application/json")
		strand.Response.WriteHeader(status)
		if jsonError, ok := strand.Error.(json.Marshaler); ok {
			encoded, _ := json.Marshal(jsonError)
			strand.Response.Write(encoded)
		} else {
			encoded, _ := json.Marshal(map[string]string{"error": strand.Error.Error()})
			strand.Response.Write(encoded)
		}
	} else {
		http.Error(strand.Response, strand.Error.Error(), status)
	}
}

func (app *App) Get(path string, handler func(*Strand) error) {
	app.routeRequireMethod("GET", path, handler)
}

func (app *App) Options(path string, handler func(*Strand) error) {
	app.routeRequireMethod("OPTIONS", path, handler)
}

func (app *App) Post(path string, handler func(*Strand) error) {
	app.routeRequireMethod("POST", path, handler)
}

func (app *App) Route(path string, handler func(*Strand) error) {
	if app.ErrorHandler == nil {
		app.ErrorHandler = app.defaultErrorHandler
	}
	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		strand := &Strand{
			Context:  context.WithValue(r.Context(), "Request", r),
			Response: w,
		}
		for _, middleware := range app.Middleware {
			if strand.Error = middleware(strand); strand.Error != nil {
				app.ErrorHandler(strand)
				return
			}
		}
		if strand.Error = handler(strand); strand.Error != nil {
			app.ErrorHandler(strand)
		}
	})
}

func (app *App) routeRequireMethod(method string, path string, handler func(*Strand) error) {
	app.Route(path, func(strand *Strand) error {
		if strand.Request().Method != method {
			return ErrorMethodNotAllowed{AllowedMethod: method}
		}
		return handler(strand)
	})
}

func (app *App) UseMiddleware(middleware ...func(*Strand) error) {
	app.Middleware = append(app.Middleware, middleware...)
}
