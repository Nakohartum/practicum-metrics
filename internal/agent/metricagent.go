package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	cpu "github.com/shirou/gopsutil/v4/cpu"
	memory "github.com/shirou/gopsutil/v4/mem"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/Nakohartum/practicum-metrics/internal/cryptoutil"
	internalLogger "github.com/Nakohartum/practicum-metrics/internal/logger"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
	pb "github.com/Nakohartum/practicum-metrics/internal/proto"
)

const realIPMetadataKey = "x-real-ip"

// MetricsAgent periodically collects runtime metrics and reports them to a server.
type MetricsAgent struct {
	PollInterval   time.Duration
	ReportInterval time.Duration
	snapshots      chan snapshot
	key            string
	client         *resty.Client
	mu             sync.RWMutex
	lastSnapshot   snapshot
	pollCount      int
	rateLimit      int
	publicKey      *rsa.PublicKey
	agentIP        string
}

// NewAgentMetrics creates a MetricsAgent.
func NewAgentMetrics(pollInterval, reportInterval time.Duration, rateLimit int, key string, cryptoKey string) (*MetricsAgent, error) {
	var pubKey *rsa.PublicKey
	if cryptoKey != "" {
		var err error
		pubKey, err = cryptoutil.LoadPublicKey(cryptoKey)
		if err != nil {
			return nil, fmt.Errorf("load public key: %w", err)
		}
	}
	agentIP := getAgentIP()
	client := resty.New().SetHeader("Content-Type", "application/json")
	if agentIP != "" {
		client.SetHeader("X-Real-IP", agentIP)
	}

	var agent = MetricsAgent{
		PollInterval:   pollInterval,
		ReportInterval: reportInterval,
		key:            key,
		client:         client,
		pollCount:      0,
		rateLimit:      rateLimit,
		publicKey:      pubKey,
		agentIP:        agentIP,
	}
	internalLogger.AttachLoggingToRequest(agent.client)
	return &agent, nil
}

func getAgentIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}

		ip := ipNet.IP.To4()
		if ip == nil || ip.IsLoopback() {
			continue
		}

		return ip.String()
	}

	return ""
}

func (mA *MetricsAgent) collectSnapshot() snapshot {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	mA.pollCount++

	metrics := make([]models.Metrics, 0, 32)
	v, _ := memory.VirtualMemory()
	cpuTotal, err := cpu.Percent(time.Second, false)
	if err != nil {
		cpuTotal[0] = 0
	}
	metrics = append(metrics,
		newGaugeMetric("Alloc", float64(mem.Alloc)),
		newGaugeMetric("BuckHashSys", float64(mem.BuckHashSys)),
		newGaugeMetric("Frees", float64(mem.Frees)),
		newGaugeMetric("GCCPUFraction", mem.GCCPUFraction),
		newGaugeMetric("GCSys", float64(mem.GCSys)),
		newGaugeMetric("HeapAlloc", float64(mem.HeapAlloc)),
		newGaugeMetric("HeapInuse", float64(mem.HeapInuse)),
		newGaugeMetric("HeapIdle", float64(mem.HeapIdle)),
		newGaugeMetric("HeapObjects", float64(mem.HeapObjects)),
		newGaugeMetric("HeapReleased", float64(mem.HeapReleased)),
		newGaugeMetric("HeapSys", float64(mem.HeapSys)),
		newGaugeMetric("LastGC", float64(mem.LastGC)),
		newGaugeMetric("Lookups", float64(mem.Lookups)),
		newGaugeMetric("MCacheInuse", float64(mem.MCacheInuse)),
		newGaugeMetric("MCacheSys", float64(mem.MCacheSys)),
		newGaugeMetric("MSpanInuse", float64(mem.MSpanInuse)),
		newGaugeMetric("MSpanSys", float64(mem.MSpanSys)),
		newGaugeMetric("Mallocs", float64(mem.Mallocs)),
		newGaugeMetric("NextGC", float64(mem.NextGC)),
		newGaugeMetric("NumForcedGC", float64(mem.NumForcedGC)),
		newGaugeMetric("NumGC", float64(mem.NumGC)),
		newGaugeMetric("OtherSys", float64(mem.OtherSys)),
		newGaugeMetric("PauseTotalNs", float64(mem.PauseTotalNs)),
		newGaugeMetric("StackInuse", float64(mem.StackInuse)),
		newGaugeMetric("StackSys", float64(mem.StackSys)),
		newGaugeMetric("Sys", float64(mem.Sys)),
		newGaugeMetric("TotalAlloc", float64(mem.TotalAlloc)),
		newGaugeMetric("RandomValue", rand.Float64()*100),
		newGaugeMetric("TotalMemory", float64(v.Total)),
		newGaugeMetric("FreeMemory", float64(v.Free)),
		newGaugeMetric("CPUutilization1", float64(cpuTotal[0])),
		newCounterMetric("PollCount", int64(mA.pollCount)),
	)

	return snapshot{Metrics: metrics}
}

func newGaugeMetric(id string, value float64) models.Metrics {
	return models.Metrics{
		ID:    id,
		MType: "gauge",
		Value: &value,
	}
}

func newCounterMetric(id string, delta int64) models.Metrics {
	return models.Metrics{
		ID:    id,
		MType: "counter",
		Delta: &delta,
	}
}

type metricsBytes = []byte

func (mA *MetricsAgent) collectLoop(ctx context.Context) {

	ticker := time.NewTicker(mA.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			snap := mA.collectSnapshot()

			mA.mu.Lock()
			mA.lastSnapshot = snap
			mA.mu.Unlock()
		}
	}
}

func (mA *MetricsAgent) sendWorker(ctx context.Context, host string) error {
	return mA.sendWorkerWith(ctx, func(ctx context.Context, snap snapshot) error {
		return mA.sendSnapshot(ctx, snap, host)
	})
}

func (mA *MetricsAgent) sendWorkerWith(ctx context.Context, send func(context.Context, snapshot) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case snap, ok := <-mA.snapshots:
			if !ok {
				return nil
			}
			if err := send(ctx, snap); err != nil {
				return err
			}
		}
	}
}

func (mA *MetricsAgent) reportLoop(ctx context.Context) {
	ticker := time.NewTicker(mA.ReportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mA.mu.RLock()
			snap := mA.lastSnapshot
			mA.mu.RUnlock()

			if len(snap.Metrics) == 0 {
				continue
			}

			select {
			case <-ctx.Done():
				return
			case mA.snapshots <- snap:
			}
		}
	}
}

func (mA *MetricsAgent) sendDataWithDeadline(ctx context.Context, v metricsBytes, endpoint string, timeoutDuration time.Duration) error {
	compressedData, err := compressData(v)
	if err != nil {
		log.Println(err)
	}
	body := compressedData
	if mA.publicKey != nil {
		body, err = cryptoutil.Encrypt(body, mA.publicKey)
		if err != nil {
			return err
		}
	}
	hash := makeHash(body, mA.key)
	req := mA.client.
		SetTimeout(timeoutDuration).
		R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(body)
	if hash != "" {
		req.SetHeader("HashSHA256", hash)
	}
	resp, err := req.Post(endpoint)
	if err != nil || resp.StatusCode() >= http.StatusBadRequest {
		body := v
		if mA.publicKey != nil {
			body, err = cryptoutil.Encrypt(body, mA.publicKey)
			if err != nil {
				return err
			}
		}
		hash = makeHash(body, mA.key)
		req := mA.client.
			SetTimeout(timeoutDuration).
			R().
			SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetBody(body)

		if hash != "" {
			req.SetHeader("HashSHA256", hash)
		}

		_, err = req.Post(endpoint)

		return err
	}
	return nil
}

func (mA *MetricsAgent) sendSnapshot(ctx context.Context, s snapshot, path string) error {
	endpoint := path + "/updates"
	attempts := 3
	timeoutDuration := 1

	jsonData, err := json.Marshal(s.Metrics)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}

	for attempt := 0; attempt < attempts; attempt++ {
		err := mA.sendDataWithDeadline(
			ctx,
			jsonData,
			endpoint,
			time.Duration(timeoutDuration)*time.Second,
		)

		var opError *net.OpError
		if err != nil && (errors.Is(err, context.DeadlineExceeded) || errors.As(err, &opError)) {
			timer := time.NewTimer(time.Duration(timeoutDuration) * time.Second)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				return ctx.Err()
			case <-timer.C:
			}
			timeoutDuration += 2
			continue
		}

		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

// Run starts collection and reporting loops until the context is canceled.
func (mA *MetricsAgent) Run(ctx context.Context, host string) error {
	return mA.runWithSender(ctx, func(ctx context.Context, snap snapshot) error {
		return mA.sendSnapshot(ctx, snap, host)
	})
}

// RunGRPC starts collection and sends metric batches to a gRPC server.
func (mA *MetricsAgent) RunGRPC(ctx context.Context, address string) error {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("create gRPC client: %w", err)
	}
	defer conn.Close()

	client := pb.NewMetricsClient(conn)
	return mA.runWithSender(ctx, func(ctx context.Context, snap snapshot) error {
		return mA.sendGRPCSnapshot(ctx, client, snap)
	})
}

func (mA *MetricsAgent) sendGRPCSnapshot(ctx context.Context, client pb.MetricsClient, snap snapshot) error {
	request := &pb.UpdateMetricsRequest{
		Metrics: make([]*pb.Metric, 0, len(snap.Metrics)),
	}
	for _, metric := range snap.Metrics {
		protoMetric := &pb.Metric{Id: metric.ID}
		switch metric.MType {
		case models.Counter:
			if metric.Delta == nil {
				return fmt.Errorf("metric %q: counter delta is required", metric.ID)
			}
			protoMetric.Type = pb.Metric_COUNTER
			protoMetric.Delta = *metric.Delta
		case models.Gauge:
			if metric.Value == nil {
				return fmt.Errorf("metric %q: gauge value is required", metric.ID)
			}
			protoMetric.Type = pb.Metric_GAUGE
			protoMetric.Value = *metric.Value
		default:
			return fmt.Errorf("metric %q: unsupported type %q", metric.ID, metric.MType)
		}
		request.Metrics = append(request.Metrics, protoMetric)
	}

	ctx = metadata.AppendToOutgoingContext(ctx, realIPMetadataKey, mA.agentIP)
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if _, err := client.UpdateMetrics(callCtx, request); err != nil {
		return fmt.Errorf("update metrics over gRPC: %w", err)
	}
	return nil
}

func (mA *MetricsAgent) runWithSender(ctx context.Context, send func(context.Context, snapshot) error) error {
	workerCount := mA.rateLimit
	if workerCount < 1 {
		workerCount = 1
	}
	mA.snapshots = make(chan snapshot, workerCount)

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		mA.collectLoop(groupCtx)
		return nil
	})
	group.Go(func() error {
		mA.reportLoop(groupCtx)
		return nil
	})

	for i := 0; i < workerCount; i++ {
		group.Go(func() error {
			return mA.sendWorkerWith(groupCtx, send)
		})
	}

	<-groupCtx.Done()
	groupErr := group.Wait()

	mA.mu.RLock()
	snap := mA.lastSnapshot
	mA.mu.RUnlock()
	if len(snap.Metrics) != 0 {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := send(shutdownCtx, snap); err != nil {
			return fmt.Errorf("send final snapshot: %w", err)
		}
	}

	close(mA.snapshots)
	if groupErr != nil && !errors.Is(groupErr, context.Canceled) {
		return groupErr
	}
	return nil
}

func compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	gz := gzip.NewWriter(&buf)

	if _, err := gz.Write(data); err != nil {
		return nil, err
	}

	if err := gz.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func makeHash(body []byte, key string) string {
	if key == "" {
		return ""
	}
	data := make([]byte, 0, len(body)+len(key))
	data = append(data, body...)
	data = append(data, key...)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
