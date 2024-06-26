package main

import (
	"github.com/WPGe/go-yandex-advanced/internal/model"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/WPGe/go-yandex-advanced/internal/handler"
	"github.com/WPGe/go-yandex-advanced/internal/storage"
)

func float64Ptr(f float64) *float64 {
	return &f
}

func int64Ptr(i int64) *int64 {
	return &i
}

func TestMetricUpdateHandler(t *testing.T) {
	type want struct {
		code            int
		request         string
		response        string
		expectedStorage *storage.MemStorage
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	logger.Sync()

	testCases := []struct {
		name    string
		storage *storage.MemStorage
		want    want
	}{
		{
			name:    "empty type",
			storage: storage.NewMemStorageWithMetrics(map[string]map[string]model.Metric{}, logger),
			want: want{
				code:     http.StatusBadRequest,
				request:  "/update/asdasd/asdasd/232",
				response: "Incorrect metric type\n",
			},
		},
		{
			name:    "empty name",
			storage: storage.NewMemStorageWithMetrics(map[string]map[string]model.Metric{}, logger),
			want: want{
				code:     http.StatusNotFound,
				request:  "/update/gauge/",
				response: "404 page not found\n",
			},
		},
		{
			name:    "incorrect value",
			storage: storage.NewMemStorageWithMetrics(map[string]map[string]model.Metric{}, logger),
			want: want{
				code:     http.StatusBadRequest,
				request:  "/update/gauge/test1/asdasdasd",
				response: "Incorrect value\n",
			},
		},
		{
			name: "add exist gauge metric",
			storage: storage.NewMemStorageWithMetrics(map[string]map[string]model.Metric{
				model.Gauge: {"test1": {
					MType: model.Gauge,
					ID:    "test1",
					Value: float64Ptr(2.5),
				}},
			}, logger),
			want: want{
				code:     http.StatusOK,
				request:  "/update/gauge/test1/2",
				response: "",
				expectedStorage: storage.NewMemStorageWithMetrics(map[string]map[string]model.Metric{
					model.Gauge: {"test1": {
						MType: model.Gauge,
						ID:    "test1",
						Value: float64Ptr(2.0),
					}},
				}, logger),
			},
		},
		{
			name: "add not exist gauge metric",
			storage: storage.NewMemStorageWithMetrics(map[string]map[string]model.Metric{
				model.Gauge: {"test1": {
					MType: model.Gauge,
					ID:    "test1",
					Value: float64Ptr(2.5),
				}},
			}, logger),
			want: want{
				code:     http.StatusOK,
				request:  "/update/gauge/test2/2",
				response: "",
				expectedStorage: storage.NewMemStorageWithMetrics(map[string]map[string]model.Metric{
					model.Gauge: {
						"test1": {
							MType: model.Gauge,
							ID:    "test1",
							Value: float64Ptr(2.5),
						},
						"test2": {
							MType: model.Gauge,
							ID:    "test2",
							Value: float64Ptr(2.0),
						},
					},
				}, logger),
			},
		},
		{
			name: "add not exist counter metric",
			storage: storage.NewMemStorageWithMetrics(map[string]map[string]model.Metric{
				model.Counter: {"test1": {
					MType: model.Counter,
					ID:    "test1",
					Delta: int64Ptr(2),
				}},
			}, logger),
			want: want{
				code:     http.StatusOK,
				request:  "/update/counter/test2/3",
				response: "",
				expectedStorage: storage.NewMemStorageWithMetrics(map[string]map[string]model.Metric{
					model.Counter: {
						"test1": {
							MType: model.Counter,
							ID:    "test1",
							Delta: int64Ptr(2),
						},
						"test2": {
							MType: model.Counter,
							ID:    "test2",
							Delta: int64Ptr(3),
						},
					},
				}, logger),
			},
		},
		{
			name: "add exist counter metric",
			storage: storage.NewMemStorageWithMetrics(map[string]map[string]model.Metric{
				model.Counter: {"test1": {
					MType: model.Counter,
					ID:    "test1",
					Delta: int64Ptr(2),
				}},
			}, logger),
			want: want{
				code:     http.StatusOK,
				request:  "/update/counter/test1/3",
				response: "",
				expectedStorage: storage.NewMemStorageWithMetrics(map[string]map[string]model.Metric{
					model.Counter: {"test1": {
						MType: model.Counter,
						ID:    "test1",
						Delta: int64Ptr(5),
					}},
				}, logger),
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/update/{type}/{name}/{value}", handler.MetricUpdateHandler(testCase.storage, logger))
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
