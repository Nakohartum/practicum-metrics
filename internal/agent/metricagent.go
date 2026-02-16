package agent

import (
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


func (mA *MetricsAgent) sendGaugeMetrics(path string){
	for k, v := range mA.gaugeMetrics{
		endpoint := fmt.Sprintf("%s/update", path)

		metric := models.Metrics{
			ID: k,
			MType: "gauge",
			Value: &v,
		}
		
		resp, err := mA.client.R().SetBody(metric).Post(endpoint)

		if err != nil {
			log.Println(err)
		}

		fmt.Print(resp)
	}
}

func (mA *MetricsAgent) sendCounterMetrics(path string){
	for k, v := range mA.counterMetrics{
		endpoint := fmt.Sprintf("%s/update", path)

		metric := models.Metrics{
			ID: k,
			MType: "counter",
			Delta: &v,
		}

		jsonData, err := json.Marshal(metric)

		if err != nil {
			panic(err)
		}
		
		
		resp, err := mA.client.R().SetBody(jsonData).Post(endpoint)

		if err != nil {
			panic(err)
		}

		fmt.Print(resp)
	}
}

func (mA *MetricsAgent) sendMetrics(path string) {
	mA.sendGaugeMetrics(path)
	mA.sendCounterMetrics(path)
}

func (mA *MetricsAgent) Run(host string) {
	elapsed := time.Duration(0)
	endpoint := host


	for {
		mA.setRuntimeGaugeMetrics()
		elapsed += mA.PollInterval

		if elapsed >= mA.ReportInterval {
			mA.setCounterMetrics()
			mA.sendMetrics(endpoint)
			elapsed = time.Duration(0)
		}
		time.Sleep(mA.PollInterval)
	}
}

