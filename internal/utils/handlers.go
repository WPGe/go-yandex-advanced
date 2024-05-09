package utils

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/WPGe/go-yandex-advanced/internal/model"
)

func WithGzip() func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		})
	}
}

func WithLogging(sugar zap.SugaredLogger) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		})
	}
}

func WithHash(key string) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hashSHA256 := r.Header.Get("HashSHA256")
			if hashSHA256 == "" {
				h.ServeHTTP(w, r)
				return
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				writeResponse(w, http.StatusBadRequest, model.Error{Error: "Bad Request"})
				return
			}
			hmacHash := hmac.New(sha256.New, []byte(key))
			if _, err := hmacHash.Write(body); err != nil {
				writeResponse(w, http.StatusInternalServerError, model.Error{Error: "Internal Server Error"})
				return
			}
			decodedHmacHash := hmacHash.Sum(nil)
			decodedHashSHA256, err := hex.DecodeString(hashSHA256)
			if err != nil {
				writeResponse(w, http.StatusInternalServerError, model.Error{Error: "Internal Server Error"})
				return
			}
			if !hmac.Equal(decodedHmacHash, decodedHashSHA256) {
				writeResponse(w, http.StatusBadRequest, model.Error{Error: "Bad Request"})
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			h.ServeHTTP(w, r)
		})
	}
}

func writeResponse(w http.ResponseWriter, code int, v any) {
	w.Header().Add("Content-Type", "application.json")
	b, err := json.Marshal(v)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error"}`))
		return
	}
	w.WriteHeader(code)
	w.Write(b)
}
