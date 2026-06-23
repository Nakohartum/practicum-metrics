package grpcapi

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/Nakohartum/practicum-metrics/internal/audit"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
	pb "github.com/Nakohartum/practicum-metrics/internal/proto"
	"github.com/Nakohartum/practicum-metrics/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const realIPMetadataKey = "x-real-ip"

// MetricsServer implements the gRPC Metrics service.
type MetricsServer struct {
	pb.UnimplementedMetricsServer
	service service.Service
	auditor *audit.Auditor
}

// NewMetricsServer creates a gRPC metrics server backed by the application service.
func NewMetricsServer(service service.Service, auditors ...*audit.Auditor) *MetricsServer {
	var auditor *audit.Auditor
	if len(auditors) > 0 {
		auditor = auditors[0]
	}
	return &MetricsServer{service: service, auditor: auditor}
}

// UpdateMetrics validates and stores a batch of metrics.
func (s *MetricsServer) UpdateMetrics(ctx context.Context, request *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	metrics, err := metricsFromProto(request.GetMetrics())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := s.service.SetDataUsingMetrics(ctx, metrics); err != nil {
		if errors.Is(err, service.ErrMetricNameRequired) ||
			errors.Is(err, service.ErrMetricTypeNotSupported) ||
			errors.Is(err, service.ErrCounterDeltaRequired) ||
			errors.Is(err, service.ErrGaugeValueRequired) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	s.notifyAudit(ctx, metrics)
	return &pb.UpdateMetricsResponse{}, nil
}

func metricsFromProto(metrics []*pb.Metric) ([]models.Metrics, error) {
	result := make([]models.Metrics, 0, len(metrics))
	for _, metric := range metrics {
		if metric == nil || metric.GetId() == "" {
			return nil, service.ErrMetricNameRequired
		}
		switch metric.GetType() {
		case pb.Metric_COUNTER:
			delta := metric.GetDelta()
			result = append(result, models.Metrics{
				ID:    metric.GetId(),
				MType: models.Counter,
				Delta: &delta,
			})
		case pb.Metric_GAUGE:
			value := metric.GetValue()
			result = append(result, models.Metrics{
				ID:    metric.GetId(),
				MType: models.Gauge,
				Value: &value,
			})
		default:
			return nil, service.ErrMetricTypeNotSupported
		}
	}
	return result, nil
}

func (s *MetricsServer) notifyAudit(ctx context.Context, metrics []models.Metrics) {
	if s.auditor == nil || !s.auditor.Enabled() || len(metrics) == 0 {
		return
	}
	names := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		names = append(names, metric.ID)
	}
	s.auditor.Notify(ctx, audit.Event{
		Ts:        time.Now().Unix(),
		Metrics:   names,
		IPAddress: realIPFromMetadata(ctx),
	})
}

// TrustedSubnetUnaryInterceptor checks x-real-ip against the configured subnet.
func TrustedSubnetUnaryInterceptor(trustedSubnet string) (grpc.UnaryServerInterceptor, error) {
	if trustedSubnet == "" {
		return func(
			ctx context.Context,
			req any,
			info *grpc.UnaryServerInfo,
			handler grpc.UnaryHandler,
		) (any, error) {
			return handler(ctx, req)
		}, nil
	}

	_, subnet, err := net.ParseCIDR(trustedSubnet)
	if err != nil {
		return nil, err
	}

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		ip := net.ParseIP(realIPFromMetadata(ctx))
		if ip == nil || !subnet.Contains(ip) {
			return nil, status.Error(codes.PermissionDenied, "agent IP is outside trusted subnet")
		}
		return handler(ctx, req)
	}, nil
}

func realIPFromMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(realIPMetadataKey)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
