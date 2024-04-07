package utils

import (
	"log"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

func WithGzip(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ow := w

		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		acceptType := r.Header.Get("Accept")
		supportType := strings.Contains(acceptType, "application/json") || strings.Contains(acceptType, "html/text") || strings.Contains(acceptType, "text/html")
		if supportsGzip && supportType {
			w.Header().Set("Content-Encoding", "gzip")
			cw := newCompressWriter(w)
			ow = cw
			defer func(cw *compressWriter) {
				err := cw.Close()
				if err != nil {
					log.Fatal(err)
				}
			}(cw)
		}

		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer func(cr *compressReader) {
				err := cr.Close()
				if err != nil {
					log.Fatal(err)
				}
			}(cr)
		}

		h.ServeHTTP(ow, r)
	}
}

func WithLogging(h http.HandlerFunc, sugar zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		responseData := &ResponseData{
			status: 0,
			size:   0,
			body:   "",
		}
		lw := LoggingResponseWriter{
			ResponseWriter: w,
			ResponseData:   responseData,
		}

		h.ServeHTTP(&lw, r)

		duration := time.Since(start)

		sugar.Infoln(
			"uri", r.RequestURI,
			"method", r.Method,
			"duration", duration,
			"status", responseData.status,
			"size", responseData.size,
			"body", responseData.body,
		)
	}
}
