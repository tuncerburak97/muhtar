package proxy

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type fiberResponseWriter struct {
	*fiber.Response
	statusCode int
	body       []byte
	header     http.Header
}

func newFiberResponseWriter(res *fiber.Response) *fiberResponseWriter {
	return &fiberResponseWriter{
		Response:   res,
		statusCode: 200, // Varsayılan durum kodu
	}
}

func (w *fiberResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
		w.header.Set("Content-Type", "application/json")
	}
	return w.header
}

func (w *fiberResponseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	w.Response.AppendBody(b)
	return len(b), nil
}

func (w *fiberResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.Response.SetStatusCode(statusCode)
}

func (w *fiberResponseWriter) StatusCode() int {
	return w.statusCode
}

func (w *fiberResponseWriter) Body() []byte {
	return w.body
}

func NewFiberResponseWriter(res *fiber.Response) *fiberResponseWriter {
	return &fiberResponseWriter{
		Response:   res,
		statusCode: 200,
	}
}
