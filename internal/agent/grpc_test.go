package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
	pb "github.com/Nakohartum/practicum-metrics/internal/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type recordingMetricsClient struct {
	request *pb.UpdateMetricsRequest
	md      metadata.MD
	err     error
}

func (c *recordingMetricsClient) UpdateMetrics(
	ctx context.Context,
	request *pb.UpdateMetricsRequest,
	_ ...grpc.CallOption,
) (*pb.UpdateMetricsResponse, error) {
	c.request = request
	c.md, _ = metadata.FromOutgoingContext(ctx)
	if c.err != nil {
		return nil, c.err
	}
	return &pb.UpdateMetricsResponse{}, nil
}

func TestMetricsAgentSendGRPCSnapshot(t *testing.T) {
	tests := []struct {
		name        string
		metrics     []models.Metrics
		clientErr   error
		wantErr     bool
		wantMetrics []*pb.Metric
	}{
		{
			name: "sends gauge and counter batch",
			metrics: []models.Metrics{
				newGaugeMetric("load", 1.5),
				newCounterMetric("hits", 3),
			},
			wantMetrics: []*pb.Metric{
				protoGaugeMetric("load", 1.5),
				protoCounterMetric("hits", 3),
			},
		},
		{
			name: "preserves zero values",
			metrics: []models.Metrics{
				newGaugeMetric("load", 0),
				newCounterMetric("hits", 0),
			},
			wantMetrics: []*pb.Metric{
				protoGaugeMetric("load", 0),
				protoCounterMetric("hits", 0),
			},
		},
		{
			name:        "sends empty batch",
			metrics:     []models.Metrics{},
			wantMetrics: []*pb.Metric{},
		},
		{
			name:    "rejects counter without delta",
			metrics: []models.Metrics{{ID: "hits", MType: models.Counter}},
			wantErr: true,
		},
		{
			name:    "rejects gauge without value",
			metrics: []models.Metrics{{ID: "load", MType: models.Gauge}},
			wantErr: true,
		},
		{
			name:    "rejects unsupported metric type",
			metrics: []models.Metrics{{ID: "metric", MType: "unknown"}},
			wantErr: true,
		},
		{
			name:        "returns gRPC client error",
			metrics:     []models.Metrics{newCounterMetric("hits", 1)},
			clientErr:   errors.New("connection failed"),
			wantErr:     true,
			wantMetrics: []*pb.Metric{protoCounterMetric("hits", 1)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metricAgent, err := NewAgentMetrics(time.Second, time.Second, 1, "", "")
			require.NoError(t, err)
			metricAgent.agentIP = "192.168.1.10"
			client := &recordingMetricsClient{err: tt.clientErr}

			err = metricAgent.sendGRPCSnapshot(context.Background(), client, snapshot{Metrics: tt.metrics})

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if tt.wantMetrics == nil {
				assert.Nil(t, client.request)
				return
			}

			require.NotNil(t, client.request)
			assert.Equal(t, tt.wantMetrics, client.request.GetMetrics())
			assert.Equal(t, []string{"192.168.1.10"}, client.md.Get(realIPMetadataKey))
		})
	}
}

func protoGaugeMetric(id string, value float64) *pb.Metric {
	return (&pb.Metric_builder{
		Id:    id,
		Type:  pb.Metric_GAUGE,
		Value: value,
	}).Build()
}

func protoCounterMetric(id string, delta int64) *pb.Metric {
	return (&pb.Metric_builder{
		Id:    id,
		Type:  pb.Metric_COUNTER,
		Delta: delta,
	}).Build()
}
