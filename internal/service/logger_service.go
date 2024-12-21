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

func NewLoggerService(repo repository.LogRepository, metrics *metrics.MetricsCollector, workerCount, bufferSize int) *LoggerService {
	s := &LoggerService{
		repo:         repo,
		requestChan:  make(chan *model.RequestLog, bufferSize),
		responseChan: make(chan *model.ResponseLog, bufferSize),
		workerCount:  workerCount,
		done:         make(chan struct{}),
		bufferSize:   bufferSize,
		metrics:      metrics,
		logger:       zap.NewExample(),
	}

	s.startWorkers()
	return s
}

func (s *LoggerService) startWorkers() {
	// Request workers
	go s.monitorBuffers()
}

func (s *LoggerService) LogRequest(log *model.Log) error {
	return s.repo.SaveLog(context.Background(), log)
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
