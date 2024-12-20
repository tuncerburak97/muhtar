package metrics

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	dto "github.com/prometheus/client_model/go"
)

var (
	defaultCollector *MetricsCollector
	once             sync.Once
)

// GetMetricsCollector returns the singleton metrics collector instance
func GetMetricsCollector(namespace, appName string) *MetricsCollector {
	once.Do(func() {
		defaultCollector = NewMetricsCollector(namespace, appName)
	})
	return defaultCollector
}

type MetricsCollector struct {
	AppName         string
	RequestDuration *prometheus.HistogramVec
	RequestCounter  *prometheus.CounterVec
	ResponseSize    *prometheus.HistogramVec
	ErrorCounter    *prometheus.CounterVec
	ActiveRequests  prometheus.Gauge
	bufferChan      chan metricEvent
	done            chan struct{}
	QueueSize       *prometheus.GaugeVec
}

type metricEvent struct {
	metricType string
	labels     prometheus.Labels
	duration   time.Duration
	size       int64
	err        error
}

type MetricsResponse struct {
	AppName   string                 `json:"app_name"`
	Timestamp time.Time              `json:"timestamp"`
	Metrics   map[string]interface{} `json:"metrics"`
}

func NewMetricsCollector(namespace, appName string) *MetricsCollector {
	m := &MetricsCollector{
		AppName: appName,
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "request_duration_seconds",
				Help:      "Request duration in seconds",
				Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"app", "method", "path", "status"},
		),

		RequestCounter: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "requests_total",
				Help:      "Total number of requests",
			},
			[]string{"app", "method", "path", "status"},
		),

		ResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "response_size_bytes",
				Help:      "Response size in bytes",
				Buckets:   []float64{100, 1000, 10000, 100000, 1000000},
			},
			[]string{"app", "method", "path", "status"},
		),

		ErrorCounter: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "errors_total",
				Help:      "Total number of errors",
			},
			[]string{"app", "type", "error", "method"},
		),

		ActiveRequests: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "active_requests",
				Help:      "Number of active requests",
				ConstLabels: prometheus.Labels{
					"app": appName,
				},
			},
		),
		bufferChan: make(chan metricEvent, 100),
		done:       make(chan struct{}),
		QueueSize: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "queue_size",
				Help:      "Current size of the queue",
			},
			[]string{"app", "type", "queue"},
		),
	}

	m.startCollector()
	return m
}

func (m *MetricsCollector) startCollector() {
	go func() {
		batch := make([]metricEvent, 0, 100)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case event := <-m.bufferChan:
				batch = append(batch, event)
				if len(batch) >= 100 {
					m.processBatch(batch)
					batch = batch[:0]
				}
			case <-ticker.C:
				if len(batch) > 0 {
					m.processBatch(batch)
					batch = batch[:0]
				}
			case <-m.done:
				return
			}
		}
	}()
}

func (m *MetricsCollector) processBatch(batch []metricEvent) {
	for _, event := range batch {
		if event.err != nil {
			m.ErrorCounter.With(event.labels).Inc()
		}
		m.RequestDuration.With(event.labels).Observe(event.duration.Seconds())
		m.ResponseSize.With(event.labels).Observe(float64(event.size))
	}
}

func (m *MetricsCollector) ObserveRequest(method, path, status, target string, duration time.Duration, size int64, err error) {
	labels := prometheus.Labels{
		"app":    m.AppName,
		"method": method,
		"path":   path,
		"status": status,
	}

	m.bufferChan <- metricEvent{
		metricType: "request",
		labels:     labels,
		duration:   duration,
		size:       size,
		err:        err,
	}
}

func (m *MetricsCollector) IncActiveRequests() {
	m.ActiveRequests.Inc()
}

func (m *MetricsCollector) DecActiveRequests() {
	m.ActiveRequests.Dec()
}

func (m *MetricsCollector) LogError(errorType string, err error) {
	m.ErrorCounter.With(prometheus.Labels{
		"app":    m.AppName,
		"type":   errorType,
		"error":  err.Error(),
		"method": "unknown",
	}).Inc()
}

func (m *MetricsCollector) ObserveBatchSave(operation string, duration time.Duration, batchSize int) {
	labels := prometheus.Labels{
		"app":    m.AppName,
		"method": "batch",
		"path":   operation,
		"status": "200",
	}
	m.RequestDuration.With(labels).Observe(duration.Seconds())
	m.RequestCounter.With(labels).Add(float64(batchSize))
}

func (m *MetricsCollector) ObserveQueueSize(queueType string, size float64) {
	m.QueueSize.With(prometheus.Labels{
		"app":   m.AppName,
		"type":  "queue_size",
		"queue": queueType,
	}).Set(size)
}

// GetMetricsJSON returns metrics in JSON format
func (m *MetricsCollector) GetMetricsJSON() ([]byte, error) {
	metrics := MetricsResponse{
		AppName:   m.AppName,
		Timestamp: time.Now(),
		Metrics: map[string]interface{}{
			"request_duration": m.getHistogramMetrics(m.RequestDuration),
			"requests_total":   m.getCounterMetrics(m.RequestCounter),
			"response_size":    m.getHistogramMetrics(m.ResponseSize),
			"errors_total":     m.getCounterMetrics(m.ErrorCounter),
			"active_requests":  m.getGaugeValue(m.ActiveRequests),
			"queue_size":       m.getGaugeVecMetrics(m.QueueSize),
		},
	}

	return json.Marshal(metrics)
}

func (m *MetricsCollector) getHistogramMetrics(vec *prometheus.HistogramVec) map[string]float64 {
	metrics := make(map[string]float64)
	ch := make(chan prometheus.Metric, 1000)
	vec.Collect(ch)
	close(ch)

	for metric := range ch {
		dtoMetric := &dto.Metric{}
		metric.Write(dtoMetric)
		hist := dtoMetric.GetHistogram()

		for _, bucket := range hist.GetBucket() {
			metrics[fmt.Sprintf("bucket_%.2f", bucket.GetUpperBound())] = float64(bucket.GetCumulativeCount())
		}
		metrics["sum"] = hist.GetSampleSum()
		metrics["count"] = float64(hist.GetSampleCount())
	}

	return metrics
}

func (m *MetricsCollector) getCounterMetrics(vec *prometheus.CounterVec) map[string]float64 {
	metrics := make(map[string]float64)
	ch := make(chan prometheus.Metric, 1000)
	vec.Collect(ch)
	close(ch)

	for metric := range ch {
		dtoMetric := &dto.Metric{}
		metric.Write(dtoMetric)
		counter := dtoMetric.GetCounter()
		metrics[getMetricName(metric)] = counter.GetValue()
	}

	return metrics
}

func (m *MetricsCollector) getGaugeValue(gauge prometheus.Gauge) float64 {
	ch := make(chan prometheus.Metric, 1)
	gauge.Collect(ch)
	close(ch)

	metric := <-ch
	dtoMetric := &dto.Metric{}
	metric.Write(dtoMetric)
	return dtoMetric.GetGauge().GetValue()
}

func (m *MetricsCollector) getGaugeVecMetrics(vec *prometheus.GaugeVec) map[string]float64 {
	metrics := make(map[string]float64)
	ch := make(chan prometheus.Metric, 1000)
	vec.Collect(ch)
	close(ch)

	for metric := range ch {
		dtoMetric := &dto.Metric{}
		metric.Write(dtoMetric)
		gauge := dtoMetric.GetGauge()
		metrics[getMetricName(metric)] = gauge.GetValue()
	}

	return metrics
}

func getMetricName(metric prometheus.Metric) string {
	dtoMetric := &dto.Metric{}
	metric.Write(dtoMetric)

	var labels []string
	for _, label := range dtoMetric.GetLabel() {
		labels = append(labels, fmt.Sprintf("%s=%s", label.GetName(), label.GetValue()))
	}

	return strings.Join(labels, ",")
}

// IncRequestCounter increments the request counter with given labels
func (m *MetricsCollector) IncRequestCounter(method, path, status string) {
	m.RequestCounter.With(prometheus.Labels{
		"app":    m.AppName,
		"method": method,
		"path":   path,
		"status": status,
	}).Inc()
}

// ObserveRequestDuration observes the request duration
func (m *MetricsCollector) ObserveRequestDuration(method, path, status string, duration time.Duration) {
	m.RequestDuration.With(prometheus.Labels{
		"app":    m.AppName,
		"method": method,
		"path":   path,
		"status": status,
	}).Observe(duration.Seconds())
}
