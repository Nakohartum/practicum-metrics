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
	"log"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	cpu "github.com/shirou/gopsutil/v4/cpu"
	memory "github.com/shirou/gopsutil/v4/mem"

	"github.com/Nakohartum/practicum-metrics/internal/cryptoutil"
	internalLogger "github.com/Nakohartum/practicum-metrics/internal/logger"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
)

// MetricsAgent periodically collects runtime metrics and reports them to a server.
type MetricsAgent struct {
	PollInterval   time.Duration
	ReportInterval time.Duration
	snapshots      chan snapshot
	key            string
	client         *resty.Client
	mu             sync.RWMutex
	lastSnapshot   snapshot
	sleep          func(time.Duration)
	pollCount      int
	rateLimit      int
	publicKey      *rsa.PublicKey
}

// NewAgentMetrics creates a MetricsAgent with intervals in seconds.
func NewAgentMetrics(pollInterval, reportInterval, rateLimit int, key string, cryptoKey string) *MetricsAgent {
	var pubKey *rsa.PublicKey
	if cryptoKey != "" {
		var err error
		pubKey, err = cryptoutil.LoadPublicKey(cryptoKey)
		if err != nil {
			slog.Error("failed to load public key", "error", err)
		}
	}
	var agent = MetricsAgent{
		PollInterval:   time.Duration(pollInterval * int(time.Second)),
		ReportInterval: time.Duration(reportInterval * int(time.Second)),
		key:            key,
		client:         resty.New().SetHeader("Content-Type", "application/json"),
		pollCount:      0,
		rateLimit:      rateLimit,
		publicKey:      pubKey,
	}
	internalLogger.AttachLoggingToRequest(agent.client)
	return &agent
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

func (mA *MetricsAgent) sendWorker(ctx context.Context, host string) {
	for {
		select {
		case <-ctx.Done():
			return
		case snap, ok := <-mA.snapshots:
			if !ok {
				return
			}
			mA.sendSnapshot(snap, host)
		}
	}
}

func (mA *MetricsAgent) reportLoop(ctx context.Context) {
	ticker := time.NewTicker(mA.ReportInterval)
	defer ticker.Stop()
	defer close(mA.snapshots)

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

func (mA *MetricsAgent) sendDataWithDeadline(v metricsBytes, endpoint string, timeoutDuration time.Duration) error {
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

func (mA *MetricsAgent) sendSnapshot(s snapshot, path string) {
	endpoint := path + "/updates"
	attempts := 3
	timeoutDuration := 1

	jsonData, err := json.Marshal(s.Metrics)
	if err != nil {
		log.Println(err)
		return
	}

	for attempt := 0; attempt < attempts; attempt++ {
		err := mA.sendDataWithDeadline(
			jsonData,
			endpoint,
			time.Duration(timeoutDuration)*time.Second,
		)

		var opError *net.OpError
		if err != nil && (errors.Is(err, context.DeadlineExceeded) || errors.As(err, &opError)) {
			time.Sleep(time.Duration(timeoutDuration) * time.Second)
			timeoutDuration += 2
			continue
		}

		if err != nil {
			log.Println(err)
		}
		break
	}
}

// Run starts collection and reporting loops until the context is canceled.
func (mA *MetricsAgent) Run(ctx context.Context, host string) {
	var wg sync.WaitGroup
	mA.snapshots = make(chan snapshot, mA.rateLimit)

	if mA.sleep == nil {
		mA.sleep = time.Sleep
	}
	wg.Add(1)

	go func() {
		defer wg.Done()
		mA.collectLoop(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		mA.reportLoop(ctx)
	}()

	for i := 0; i < mA.rateLimit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mA.sendWorker(ctx, host)
		}()
	}

	wg.Wait()
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
