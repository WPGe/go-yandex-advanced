package main

import (
	"github.com/WPGe/go-yandex-advanced/internal/entity"
	"github.com/WPGe/go-yandex-advanced/internal/handler"
	"github.com/WPGe/go-yandex-advanced/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetricUpdateHandler(t *testing.T) {
	type want struct {
		code            int
		request         string
		response        string
		expectedStorage *storage.MemStorage
	}

	testCases := []struct {
		name    string
		storage *storage.MemStorage
		want    want
	}{
		{
			name:    "empty type",
			storage: storage.NewMemStorageWithMetrics(map[string]entity.Metric{}),
			want: want{
				code:     http.StatusBadRequest,
				request:  "/update/asdasd/asdasd/232",
				response: "Incorrect metric type\n",
			},
		},
		{
			name:    "empty name",
			storage: storage.NewMemStorageWithMetrics(map[string]entity.Metric{}),
			want: want{
				code:     http.StatusNotFound,
				request:  "/update/gauge/",
				response: "404 page not found\n",
			},
		},
		{
			name:    "incorrect value",
			storage: storage.NewMemStorageWithMetrics(map[string]entity.Metric{}),
			want: want{
				code:     http.StatusBadRequest,
				request:  "/update/gauge/test1/asdasdasd",
				response: "Incorrect value\n",
			},
		},
		{
			name: "add exist gauge metric",
			storage: storage.NewMemStorageWithMetrics(map[string]entity.Metric{
				"test1": {
					Type:  entity.Gauge,
					Name:  "test1",
					Value: 2.5,
				},
			}),
			want: want{
				code:     http.StatusOK,
				request:  "/update/gauge/test1/2",
				response: "",
				expectedStorage: storage.NewMemStorageWithMetrics(map[string]entity.Metric{
					"test1": {
						Type:  entity.Gauge,
						Name:  "test1",
						Value: 2.0,
					},
				}),
			},
		},
		{
			name: "add not exist gauge metric",
			storage: storage.NewMemStorageWithMetrics(map[string]entity.Metric{
				"test1": {
					Type:  entity.Gauge,
					Name:  "test1",
					Value: 2.5,
				},
			}),
			want: want{
				code:     http.StatusOK,
				request:  "/update/gauge/test2/2",
				response: "",
				expectedStorage: storage.NewMemStorageWithMetrics(map[string]entity.Metric{
					"test1": {
						Type:  entity.Gauge,
						Name:  "test1",
						Value: 2.5,
					},
					"test2": {
						Type:  entity.Gauge,
						Name:  "test2",
						Value: 2.0,
					},
				}),
			},
		},
		{
			name: "add not exist counter metric",
			storage: storage.NewMemStorageWithMetrics(map[string]entity.Metric{
				"test1": {
					Type:  entity.Counter,
					Name:  "test1",
					Value: int64(2),
				},
			}),
			want: want{
				code:     http.StatusOK,
				request:  "/update/counter/test2/3",
				response: "",
				expectedStorage: storage.NewMemStorageWithMetrics(map[string]entity.Metric{
					"test1": {
						Type:  entity.Counter,
						Name:  "test1",
						Value: int64(2),
					},
					"test2": {
						Type:  entity.Counter,
						Name:  "test2",
						Value: int64(3),
					},
				}),
			},
		},
		{
			name: "add exist counter metric",
			storage: storage.NewMemStorageWithMetrics(map[string]entity.Metric{
				"test1": {
					Type:  entity.Counter,
					Name:  "test1",
					Value: int64(2),
				},
			}),
			want: want{
				code:     http.StatusOK,
				request:  "/update/counter/test1/3",
				response: "",
				expectedStorage: storage.NewMemStorageWithMetrics(map[string]entity.Metric{
					"test1": {
						Type:  entity.Counter,
						Name:  "test1",
						Value: int64(5),
					},
				}),
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/update/{type}/{name}/{value}", handler.MetricUpdateHandler(testCase.storage))
			srv := httptest.NewServer(r)
			defer srv.Close()

			req := resty.New().R()
			req.Method = http.MethodPost
			req.URL = srv.URL + testCase.want.request

			resp, err := req.Send()
			assert.NoError(t, err, "error making HTTP request")

			assert.Equal(t, testCase.want.code, resp.StatusCode())

			if testCase.want.response != "" {
				require.Equal(t, testCase.want.response, string(resp.Body()))
			}

			if testCase.want.expectedStorage != nil {
				require.Equal(t, testCase.want.expectedStorage, testCase.storage)
			}
		})
	}
}
