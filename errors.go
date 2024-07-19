package trance

import (
	"encoding/json"
	"net/http"
	"strings"
)

type ErrorBadRequest struct {
	Message string
}

func (err ErrorBadRequest) Error() string {
	if err.Message != "" {
		return err.Message
	}
	return "Bad request"
}

func (err ErrorBadRequest) Status() int {
	return http.StatusBadRequest
}

type ErrorInternalServer struct {
	Message string
}

func (err ErrorInternalServer) Error() string {
	if err.Message != "" {
		return err.Message
	}
	return "Internal server error"
}

func (err ErrorInternalServer) Status() int {
	return http.StatusInternalServerError
}

type ErrorMethodNotAllowed struct {
	AllowedMethod string
}

func (err ErrorMethodNotAllowed) Error() string {
	if err.AllowedMethod != "" {
		return "Method not allowed. Must be " + err.AllowedMethod
	}
	return "Method not allowed"
}

func (err ErrorMethodNotAllowed) Status() int {
	return http.StatusMethodNotAllowed
}

type ErrorNotFound struct{}

func (err ErrorNotFound) Error() string {
	return "Not found"
}

func (err ErrorNotFound) Status() int {
	return http.StatusNotFound
}

type ErrorUnauthorized struct{}

func (err ErrorUnauthorized) Error() string {
	return "Unauthorized"
}

func (err ErrorUnauthorized) Status() int {
	return http.StatusUnauthorized
}

type ErrorWithStatus interface {
	error
	Status() int
}

type FormErrors struct {
	Errors     map[string]error
	StatusCode int
}

func (form FormErrors) Error() string {
	s := strings.Builder{}
	i := 0
	for column, err := range form.Errors {
		if i > 0 {
			s.WriteString("\n")
		}
		s.WriteString(column)
		s.WriteString(": ")
		s.WriteString(err.Error())
		i++
	}
	return s.String()
}

func (form FormErrors) MarshalJSON() ([]byte, error) {
	errorsMap := make(map[string]string, 0)
	for key, err := range form.Errors {
		errorsMap[key] = err.Error()
	}
	return json.Marshal(map[string]map[string]string{"errors": errorsMap})
}

func (form FormErrors) Status() int {
	return form.StatusCode
}

type StatusOK struct{}

func (err StatusOK) Error() string {
	return ""
}

func (err StatusOK) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]bool{"ok": true})
}

func (err StatusOK) Status() int {
	return http.StatusOK
}
