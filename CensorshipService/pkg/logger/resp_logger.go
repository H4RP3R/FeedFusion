package logger

import "net/http"

type ResponseLogger struct {
	w      http.ResponseWriter
	status int
}

func New(w http.ResponseWriter) *ResponseLogger {
	return &ResponseLogger{w, http.StatusOK}
}

func (l *ResponseLogger) WriteHeader(code int) {
	l.status = code
	l.w.WriteHeader(code)
}

func (l *ResponseLogger) Write(b []byte) (int, error) {
	return l.w.Write(b)
}

func (l *ResponseLogger) Header() http.Header {
	return l.w.Header()
}

func (l *ResponseLogger) Status() int {
	return l.status
}
