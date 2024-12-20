package service

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tuncerburak97/muhtar/internal/metrics"
	"github.com/tuncerburak97/muhtar/internal/model"
	"github.com/tuncerburak97/muhtar/internal/repository"
	"go.uber.org/zap"
)

type LoggerService struct {
	repo         repository.LogRepository
	requestChan  chan *model.RequestLog
	responseChan chan *model.ResponseLog
	workerCount  int
	wg           sync.WaitGroup
	done         chan struct{}
	bufferSize   int
	mu           sync.RWMutex
	metrics      *metrics.MetricsCollector
	logger       *zap.Logger
}

func NewLoggerService(repo repository.LogRepository, workerCount, bufferSize int) *LoggerService {
	s := &LoggerService{
		repo:         repo,
		requestChan:  make(chan *model.RequestLog, bufferSize),
		responseChan: make(chan *model.ResponseLog, bufferSize),
		workerCount:  workerCount,
		done:         make(chan struct{}),
		bufferSize:   bufferSize,
		metrics:      metrics.GetMetricsCollector("muhtar", "muhtar_proxy"),
		logger:       zap.NewExample(),
	}

	s.startWorkers()
	return s
}

func (s *LoggerService) startWorkers() {
	// Request workers
	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.processRequestLogs(i)
	}

	// Response workers
	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.processResponseLogs(i)
	}

	// Start buffer monitor
	go s.monitorBuffers()
}

func (s *LoggerService) processRequestLogs(workerID int) {
	defer s.wg.Done()

	ctx := context.Background()
	batch := make([]*model.RequestLog, 0, 100)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case log := <-s.requestChan:
			batch = append(batch, log)
			if len(batch) >= 100 {
				s.saveBatchRequestLogs(ctx, batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				s.saveBatchRequestLogs(ctx, batch)
				batch = batch[:0]
			}
		}
	}
}

func (s *LoggerService) saveBatchRequestLogs(ctx context.Context, batch []*model.RequestLog) {
	start := time.Now()
	logs := make([]*model.Log, len(batch))
	for i, reqLog := range batch {
		logs[i] = &model.Log{
			ID:          reqLog.ID,
			TraceID:     reqLog.TraceID,
			ProcessType: model.ProcessTypeRequest,
			Timestamp:   reqLog.Timestamp,
			Method:      reqLog.Method,
			URL:         reqLog.URL,
			Path:        reqLog.Path,
			PathParams:  reqLog.PathParams,
			QueryParams: reqLog.QueryParams,
			Headers:     reqLog.Headers,
			Body:        reqLog.RequestBody,
			ClientIP:    reqLog.ClientIP,
			UserAgent:   reqLog.UserAgent,
		}
	}
	if err := s.repo.SaveLogs(ctx, logs); err != nil {
		s.metrics.LogError("batch_request_save", err)
	}
	s.metrics.ObserveBatchSave("request", time.Since(start), len(batch))
}

func (s *LoggerService) processResponseLogs(workerID int) {
	defer s.wg.Done()
	ctx := context.Background()
	batch := make([]*model.ResponseLog, 0, 100)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case log := <-s.responseChan:
			batch = append(batch, log)
			if len(batch) >= 100 {
				s.saveBatchResponseLogs(ctx, batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				s.saveBatchResponseLogs(ctx, batch)
				batch = batch[:0]
			}
		}
	}
}

func (s *LoggerService) saveBatchResponseLogs(ctx context.Context, batch []*model.ResponseLog) {
	start := time.Now()
	logs := make([]*model.Log, len(batch))
	for i, respLog := range batch {
		logs[i] = &model.Log{
			ID:            respLog.ID,
			TraceID:       respLog.TraceID,
			ProcessType:   model.ProcessTypeResponse,
			Timestamp:     respLog.Timestamp,
			StatusCode:    respLog.StatusCode,
			Headers:       respLog.Headers,
			Body:          respLog.ResponseBody,
			ResponseTime:  respLog.ResponseTime,
			ContentLength: respLog.ContentLength,
			Error:         respLog.Error,
		}
	}
	if err := s.repo.SaveLogs(ctx, logs); err != nil {
		s.metrics.LogError("batch_response_save", err)
		s.logger.Error("Failed to save response logs batch",
			zap.Error(err),
			zap.Int("batch_size", len(batch)),
			zap.String("operation", "save_response_logs"),
		)
	}
	s.metrics.ObserveBatchSave("response", time.Since(start), len(batch))
}

func (s *LoggerService) LogRequest(log *model.RequestLog) {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	if log.TraceID == "" {
		log.TraceID = uuid.New().String()
	}
	s.requestChan <- log
}

func (s *LoggerService) LogResponse(log *model.ResponseLog) {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	s.responseChan <- log
}

func (s *LoggerService) Shutdown() {
	close(s.done)
	close(s.requestChan)
	close(s.responseChan)
	s.wg.Wait()
	s.repo.Close()
}

func (s *LoggerService) monitorBuffers() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.metrics.ObserveQueueSize("request", float64(len(s.requestChan)))
			s.metrics.ObserveQueueSize("response", float64(len(s.responseChan)))
		}
	}
}
