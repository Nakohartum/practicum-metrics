package grpcapi

import (
	"context"
	"errors"
	"testing"

	"github.com/Nakohartum/practicum-metrics/internal/mocks"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
	pb "github.com/Nakohartum/practicum-metrics/internal/proto"
	"github.com/Nakohartum/practicum-metrics/internal/service"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestMetricsServerUpdateMetrics(t *testing.T) {
	serviceFailure := errors.New("storage failed")
	tests := []struct {
		name        string
		request     *pb.UpdateMetricsRequest
		serviceErr  error
		wantMetrics []models.Metrics
		wantCode    codes.Code
		callService bool
	}{
		{
			name: "stores gauge and counter batch",
			request: protoUpdateRequest(
				protoGaugeMetric("load", 2.5),
				protoCounterMetric("hits", 4),
			),
			wantMetrics: []models.Metrics{
				gaugeMetric("load", 2.5),
				counterMetric("hits", 4),
			},
			wantCode:    codes.OK,
			callService: true,
		},
		{
			name: "stores zero values",
			request: protoUpdateRequest(
				protoGaugeMetric("load", 0),
				protoCounterMetric("hits", 0),
			),
			wantMetrics: []models.Metrics{
				gaugeMetric("load", 0),
				counterMetric("hits", 0),
			},
			wantCode:    codes.OK,
			callService: true,
		},
		{
			name:        "stores empty batch",
			request:     protoUpdateRequest(),
			wantMetrics: []models.Metrics{},
			wantCode:    codes.OK,
			callService: true,
		},
		{
			name:     "rejects metric without ID",
			request:  protoUpdateRequest(protoCounterMetric("", 1)),
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "rejects nil metric",
			request:  protoUpdateRequest(nil),
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "rejects unknown metric type",
			request:  protoUpdateRequest(protoMetric("metric", pb.Metric_MType(100), 0, 0)),
			wantCode: codes.InvalidArgument,
		},
		{
			name:        "maps validation service error",
			request:     protoUpdateRequest(protoCounterMetric("hits", 1)),
			serviceErr:  service.ErrMetricTypeNotSupported,
			wantMetrics: []models.Metrics{counterMetric("hits", 1)},
			wantCode:    codes.InvalidArgument,
			callService: true,
		},
		{
			name:        "maps storage error to internal",
			request:     protoUpdateRequest(protoCounterMetric("hits", 1)),
			serviceErr:  serviceFailure,
			wantMetrics: []models.Metrics{counterMetric("hits", 1)},
			wantCode:    codes.Internal,
			callService: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			metricService := mocks.NewMockService(ctrl)
			if tt.callService {
				metricService.EXPECT().
					SetDataUsingMetrics(gomock.Any(), tt.wantMetrics).
					Return(tt.serviceErr)
			}
			server := NewMetricsServer(metricService)

			response, err := server.UpdateMetrics(context.Background(), tt.request)

			assert.Equal(t, tt.wantCode, status.Code(err))
			if tt.wantCode == codes.OK {
				require.NoError(t, err)
				assert.NotNil(t, response)
			} else {
				require.Error(t, err)
				assert.Nil(t, response)
			}
		})
	}
}

func TestTrustedSubnetUnaryInterceptor(t *testing.T) {
	tests := []struct {
		name              string
		trustedSubnet     string
		ip                string
		wantConfigErr     bool
		wantCode          codes.Code
		wantHandlerCalled bool
	}{
		{
			name:              "allows trusted IPv4 address",
			trustedSubnet:     "192.168.1.0/24",
			ip:                "192.168.1.10",
			wantCode:          codes.OK,
			wantHandlerCalled: true,
		},
		{
			name:              "allows trusted IPv6 address",
			trustedSubnet:     "2001:db8::/32",
			ip:                "2001:db8::1",
			wantCode:          codes.OK,
			wantHandlerCalled: true,
		},
		{
			name:          "denies address outside subnet",
			trustedSubnet: "192.168.1.0/24",
			ip:            "10.0.0.1",
			wantCode:      codes.PermissionDenied,
		},
		{
			name:          "denies missing metadata",
			trustedSubnet: "192.168.1.0/24",
			wantCode:      codes.PermissionDenied,
		},
		{
			name:          "denies malformed IP",
			trustedSubnet: "192.168.1.0/24",
			ip:            "not-an-ip",
			wantCode:      codes.PermissionDenied,
		},
		{
			name:              "allows request when subnet is disabled",
			ip:                "not-an-ip",
			wantCode:          codes.OK,
			wantHandlerCalled: true,
		},
		{
			name:          "rejects invalid subnet configuration",
			trustedSubnet: "invalid",
			wantConfigErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor, err := TrustedSubnetUnaryInterceptor(tt.trustedSubnet)
			if tt.wantConfigErr {
				require.Error(t, err)
				assert.Nil(t, interceptor)
				return
			}
			require.NoError(t, err)

			ctx := context.Background()
			if tt.ip != "" {
				ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(realIPMetadataKey, tt.ip))
			}
			handlerCalled := false
			response, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, func(context.Context, any) (any, error) {
				handlerCalled = true
				return "ok", nil
			})

			assert.Equal(t, tt.wantCode, status.Code(err))
			assert.Equal(t, tt.wantHandlerCalled, handlerCalled)
			if tt.wantCode == codes.OK {
				assert.Equal(t, "ok", response)
			} else {
				assert.Nil(t, response)
			}
		})
	}
}

func gaugeMetric(id string, value float64) models.Metrics {
	return models.Metrics{ID: id, MType: models.Gauge, Value: &value}
}

func counterMetric(id string, delta int64) models.Metrics {
	return models.Metrics{ID: id, MType: models.Counter, Delta: &delta}
}

func protoUpdateRequest(metrics ...*pb.Metric) *pb.UpdateMetricsRequest {
	return (&pb.UpdateMetricsRequest_builder{Metrics: metrics}).Build()
}

func protoMetric(id string, metricType pb.Metric_MType, delta int64, value float64) *pb.Metric {
	return (&pb.Metric_builder{
		Id:    id,
		Type:  metricType,
		Delta: delta,
		Value: value,
	}).Build()
}

func protoGaugeMetric(id string, value float64) *pb.Metric {
	return protoMetric(id, pb.Metric_GAUGE, 0, value)
}

func protoCounterMetric(id string, delta int64) *pb.Metric {
	return protoMetric(id, pb.Metric_COUNTER, delta, 0)
}
