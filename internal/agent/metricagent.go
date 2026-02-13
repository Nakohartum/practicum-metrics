package agent

import (
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"time"

	internalLogger "github.com/Nakohartum/practicum-metrics/internal/logger"
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
		client: resty.New().SetHeader("Content-Type", "text/plain"),
	}
	internalLogger.AttachLoggingToRequest(agent.client)
	return &agent
}

func (mA *MetricsAgent) setRuntimeGaugeMetrics() {
	var m runtime.MemStats

	runtime.ReadMemStats(&m)

	mA.setGaugeMetric("alloc",float64(m.Alloc))
	mA.setGaugeMetric("buckHashSys", float64(m.BuckHashSys))
	mA.setGaugeMetric("frees", float64(m.Frees))
	mA.setGaugeMetric("gCCPUFraction", m.GCCPUFraction)
	mA.setGaugeMetric("gCSys", float64(m.GCSys))
	mA.setGaugeMetric("heapAlloc", float64(m.HeapAlloc))
	mA.setGaugeMetric("heapInuse", float64(m.HeapInuse))
	mA.setGaugeMetric("heapIdle", float64(m.HeapIdle))
	mA.setGaugeMetric("heapObjects", float64(m.HeapObjects))
	mA.setGaugeMetric("heapReleased", float64(m.HeapReleased))
	mA.setGaugeMetric("heapSys", float64(m.HeapSys))
	mA.setGaugeMetric("lastGC", float64(m.LastGC))
	mA.setGaugeMetric("lookups", float64(m.Lookups))
	mA.setGaugeMetric("mCacheInuse", float64(m.MCacheInuse))
	mA.setGaugeMetric("mCacheSys", float64(m.MCacheSys))
	mA.setGaugeMetric("mSpanInuse", float64(m.MSpanInuse))
	mA.setGaugeMetric("mSpanSys", float64(m.MSpanSys))
	mA.setGaugeMetric("mallocs", float64(m.Mallocs))
	mA.setGaugeMetric("nextGC", float64(m.NextGC))
	mA.setGaugeMetric("numForcedGC", float64(m.NumForcedGC))
	mA.setGaugeMetric("numGC", float64(m.NumGC))
	mA.setGaugeMetric("otherSys", float64(m.OtherSys))
	mA.setGaugeMetric("pauseTotalNs", float64(m.PauseTotalNs))
	mA.setGaugeMetric("stackInuse", float64(m.StackInuse))
	mA.setGaugeMetric("stackSys", float64(m.StackSys))
	mA.setGaugeMetric("sys", float64(m.Sys))
	mA.setGaugeMetric("totalAlloc", float64(m.TotalAlloc))
	mA.setGaugeMetric("randomValue", rand.Float64() * 100)
}

func (mA *MetricsAgent) setGaugeMetric(metricName string, value float64) {
	mA.gaugeMetrics[metricName] = value
}

func (mA *MetricsAgent) setCounterMetrics() {
	mA.counterMetrics["pollCount"] = 1
}


func (mA *MetricsAgent) sendGaugeMetrics(path string){
	for k, v := range mA.gaugeMetrics{
		endpoint := fmt.Sprintf("%s/update/gauge/%s/%s", path, k, strconv.FormatFloat(v, 'f', -1, 64))
		
		resp, err := mA.client.R().Post(endpoint)

		if err != nil {
			panic(err)
		}

		fmt.Print(resp)
	}
}

func (mA *MetricsAgent) sendCounterMetrics(path string){
	for k, v := range mA.counterMetrics{
		endpoint := fmt.Sprintf("%s/update/counter/%s/%s", path, k, strconv.FormatInt(v, 10))
		
		resp, err := mA.client.R().Post(endpoint)

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

