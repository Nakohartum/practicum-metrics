package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"time"

	internalLogger "github.com/Nakohartum/practicum-metrics/internal/logger"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/go-resty/resty/v2"
)

type MetricsAgent struct {
	PollInterval   time.Duration
	ReportInterval time.Duration
	gaugeMetrics   map[string]float64
	counterMetrics map[string]int64
	client *resty.Client
	sleep func(time.Duration)
}

func NewAgentMetrics(pollInterval, reportInterval int) *MetricsAgent {
	var agent = MetricsAgent{
		PollInterval:   time.Duration(pollInterval * int(time.Second)),
		ReportInterval: time.Duration(reportInterval * int(time.Second)),
		gaugeMetrics:   make(map[string]float64),
		counterMetrics: make(map[string]int64),
		client: resty.New().SetHeader("Content-Type", "application/json"),
	}
	internalLogger.AttachLoggingToRequest(agent.client)
	return &agent
}

func (mA *MetricsAgent) setRuntimeGaugeMetrics() {
	var m runtime.MemStats

	runtime.ReadMemStats(&m)

	mA.setGaugeMetric("Alloc",float64(m.Alloc))
	mA.setGaugeMetric("BuckHashSys", float64(m.BuckHashSys))
	mA.setGaugeMetric("Frees", float64(m.Frees))
	mA.setGaugeMetric("GCCPUFraction", m.GCCPUFraction)
	mA.setGaugeMetric("GCSys", float64(m.GCSys))
	mA.setGaugeMetric("HeapAlloc", float64(m.HeapAlloc))
	mA.setGaugeMetric("HeapInuse", float64(m.HeapInuse))
	mA.setGaugeMetric("HeapIdle", float64(m.HeapIdle))
	mA.setGaugeMetric("HeapObjects", float64(m.HeapObjects))
	mA.setGaugeMetric("HeapReleased", float64(m.HeapReleased))
	mA.setGaugeMetric("HeapSys", float64(m.HeapSys))
	mA.setGaugeMetric("LastGC", float64(m.LastGC))
	mA.setGaugeMetric("Lookups", float64(m.Lookups))
	mA.setGaugeMetric("MCacheInuse", float64(m.MCacheInuse))
	mA.setGaugeMetric("MCacheSys", float64(m.MCacheSys))
	mA.setGaugeMetric("MSpanInuse", float64(m.MSpanInuse))
	mA.setGaugeMetric("MSpanSys", float64(m.MSpanSys))
	mA.setGaugeMetric("Mallocs", float64(m.Mallocs))
	mA.setGaugeMetric("NextGC", float64(m.NextGC))
	mA.setGaugeMetric("NumForcedGC", float64(m.NumForcedGC))
	mA.setGaugeMetric("NumGC", float64(m.NumGC))
	mA.setGaugeMetric("OtherSys", float64(m.OtherSys))
	mA.setGaugeMetric("PauseTotalNs", float64(m.PauseTotalNs))
	mA.setGaugeMetric("StackInuse", float64(m.StackInuse))
	mA.setGaugeMetric("StackSys", float64(m.StackSys))
	mA.setGaugeMetric("Sys", float64(m.Sys))
	mA.setGaugeMetric("TotalAlloc", float64(m.TotalAlloc))
	mA.setGaugeMetric("RandomValue", rand.Float64() * 100)
}

func (mA *MetricsAgent) setGaugeMetric(metricName string, value float64) {
	mA.gaugeMetrics[metricName] = value
}

func (mA *MetricsAgent) setCounterMetrics() {
	mA.counterMetrics["PollCount"]++
}

type metricsBytes = []byte

func (mA *MetricsAgent) sendGaugeMetrics() []metricsBytes{
	metrics := make([]metricsBytes, 0)
	
	for k, v := range mA.gaugeMetrics{

		metric := models.Metrics{
			ID: k,
			MType: "gauge",
			Value: &v,
		}

		jsonData, err := json.Marshal(metric)

		if err != nil {
			log.Println(err)
		}

		compressedData, err := compressData(jsonData)

		if err != nil {
			log.Println(err)
		}
		metrics = append(metrics, compressedData)
	}
	return metrics
}

func (mA *MetricsAgent) sendCounterMetrics() []metricsBytes{
	metrics := make([]metricsBytes, 0)
	for k, v := range mA.counterMetrics{

		metric := models.Metrics{
			ID: k,
			MType: "counter",
			Delta: &v,
		}

		jsonData, err := json.Marshal(metric)

		if err != nil {
			log.Println(err)
		}
		
		compressedData, err := compressData(jsonData)

		if err != nil {
			log.Println(err)
		}

		metrics = append(metrics, compressedData)
	}
	return metrics
}

func (mA *MetricsAgent) sendMetrics(path string) {
	metricsToSend := make([]metricsBytes, 0)
	metricsToSend = append(metricsToSend, mA.sendGaugeMetrics()...)
	metricsToSend = append(metricsToSend, mA.sendCounterMetrics()...)
	endpoint := fmt.Sprintf("%s/update", path)
	for start := 0; start < len(metricsToSend); start += 10 {
		end := start + 10
		if end > len(metricsToSend) {
			end = len(metricsToSend)
		}

		for _, v := range metricsToSend[start:end] {
			resp, err := mA.client.R().
				SetHeader("Content-Type", "application/json").
				SetHeader("Content-Encoding", "gzip").
				SetBody(v).Post(endpoint)
			if err != nil {
				log.Fatal(err.Error())
				return
			}
			log.Print(resp)
		}
	}
}

func (mA *MetricsAgent) Run(ctx context.Context, host string) {
	if mA.sleep == nil {
		mA.sleep = time.Sleep
	}
	elapsed := time.Duration(0)
	endpoint := host


	for {
		select{
		case <- ctx.Done():
			return
		default:
		}
		mA.setRuntimeGaugeMetrics()
		elapsed += mA.PollInterval

		if elapsed >= mA.ReportInterval {
			mA.setCounterMetrics()
			mA.sendMetrics(endpoint)
			elapsed = 0
		}
		mA.sleep(mA.PollInterval)
	}
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
