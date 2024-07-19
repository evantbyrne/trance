package trance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Strand struct {
	Context  context.Context
	Error    error
	Response http.ResponseWriter
}

func (strand *Strand) Redirect(url string) error {
	return strand.RedirectWithStatus(url, http.StatusFound)
}

func (strand *Strand) RedirectWithStatus(url string, status int) error {
	strand.Response.Header().Set("Location", url)
	strand.Response.WriteHeader(status)
	return nil
}

func (strand *Strand) Request() *http.Request {
	return strand.Context.Value("Request").(*http.Request)
}

func (strand *Strand) WriteHtml(body any) error {
	strand.Response.Header().Set("Content-Type", "text/html")
	switch bv := body.(type) {
	case []byte:
		_, err := strand.Response.Write(bv)
		return err
	}
	_, err := fmt.Fprint(strand.Response, body)
	return err
}

func (strand *Strand) WriteJson(body any) error {
	strand.Response.Header().Set("Content-Type", "application/json")
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	_, err = strand.Response.Write(encoded)
	return err
}
