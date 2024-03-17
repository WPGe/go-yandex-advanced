package utils

import "net/http"

type (
	ResponseData struct {
		status int
		size   int
		body   string
	}
	LoggingResponseWriter struct {
		http.ResponseWriter
		ResponseData *ResponseData
	}
)

func (r *LoggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.ResponseData.size += size
	r.ResponseData.body = string(b) // Сохраняем тело ответа
	return size, err
}

func (r *LoggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.ResponseData.status = statusCode
}

/// ОТМЕТКА О ТОМ, ЧТО НУЖНО ОБРАТИТЬ ВНИМАНИЕ ПРИ РЕВЬЮ
