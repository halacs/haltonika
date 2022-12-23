package impl

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/halacs/haltonika/config"
	"github.com/sirupsen/logrus"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type Metrics struct {
	ctx      context.Context
	wg       *sync.WaitGroup
	values   *persistentMetrics
	fileName string
}

type persistentMetrics struct {
	SentBytes         uint64
	ReceivedBytes     uint64
	SentPackages      uint64
	ReceivedPackages  uint64
	MalformedPackages uint64
	RejectedPackages  uint64
	ResentPackages    uint64
}

func NewMetrics(ctx context.Context, wg *sync.WaitGroup, fileName string) *Metrics {
	metrics := &Metrics{
		ctx:      ctx,
		wg:       wg,
		fileName: fileName,
		values: &persistentMetrics{
			SentBytes:         0,
			ReceivedBytes:     0,
			SentPackages:      0,
			ReceivedPackages:  0,
			MalformedPackages: 0,
			RejectedPackages:  0,
			ResentPackages:    0,
		},
	}

	ticker := time.NewTicker(60 * time.Second)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
			case <-ticker.C:
				err := metrics.save()
				if err != nil {
					logrus.Errorf("Failed to save metrics. %v", err)
				}
			}
		}
	}()

	err := metrics.load()
	if err != nil {
		logrus.Errorf("Warn to load previously saved metrics. %v", err)
	}

	return metrics
}

func (m *Metrics) Close() error {
	err := m.save()
	if err != nil {
		return fmt.Errorf("failed to save metrics data. %v", err)
	}

	return nil
}

func (m *Metrics) AddSentBytes(count uint64) {
	atomic.AddUint64(&m.values.SentBytes, count)
}

func (m *Metrics) GetSentBytes() uint64 {
	return atomic.AddUint64(&m.values.SentBytes, 0)
}

func (m *Metrics) AddReceivedBytes(count uint64) {
	atomic.AddUint64(&m.values.ReceivedBytes, count)
}

func (m *Metrics) GetReceivedBytes() uint64 {
	return atomic.AddUint64(&m.values.ReceivedBytes, 0)
}

func (m *Metrics) AddSentPackages(count uint64) {
	atomic.AddUint64(&m.values.SentPackages, count)
}

func (m *Metrics) GetSentPackages() uint64 {
	return atomic.AddUint64(&m.values.SentPackages, 0)
}

func (m *Metrics) AddReceivedPackages(count uint64) {
	atomic.AddUint64(&m.values.ReceivedPackages, count)
}

func (m *Metrics) GetReceivedPackages() uint64 {
	return atomic.AddUint64(&m.values.ReceivedPackages, 0)
}

func (m *Metrics) AddMalformedPackages(count uint64) {
	atomic.AddUint64(&m.values.MalformedPackages, count)
}

func (m *Metrics) GetMalformedPackages() uint64 {
	return atomic.AddUint64(&m.values.MalformedPackages, 0)
}

func (m *Metrics) AddRejectedPackages(count uint64) {
	atomic.AddUint64(&m.values.RejectedPackages, count)
}

func (m *Metrics) GetRejectedPackages() uint64 {
	return atomic.AddUint64(&m.values.RejectedPackages, 0)
}

func (m *Metrics) AddResentPackages(count uint64) {
	atomic.AddUint64(&m.values.ResentPackages, count)
}

func (m *Metrics) GetResentPackages() uint64 {
	return atomic.AddUint64(&m.values.ResentPackages, 0)
}

/*
Provides metrics in InfluxDB linie protocol format
*/
func (m *Metrics) MetricRendererHandler() (string, map[string]uint64) {
	log := config.GetLogger(m.ctx)

	err := m.save()
	if err != nil {
		log.Errorf("Failed to persist metric counters! %v", err)
	}

	metricName := "haltonika"
	metrics := map[string]uint64{
		"SentBytes":         m.GetSentBytes(),
		"SentPackages":      m.GetSentPackages(),
		"ReceivedBytes":     m.GetReceivedBytes(),
		"ReceivedPackages":  m.GetReceivedPackages(),
		"RejectedPackages":  m.GetRejectedPackages(),
		"MalformedPackages": m.GetMalformedPackages(),
		"ResentPackages":    m.GetResentPackages(),
	}

	return metricName, metrics
}

func (m *Metrics) save() error {
	if m.fileName == "" {
		return fmt.Errorf("filename must not be empty")
	}

	jsonData, err := json.MarshalIndent(m.values, "", " ")
	if err != nil {
		return fmt.Errorf("failed to serialize metric data into json format. %v", err)
	}

	err = os.WriteFile(m.fileName, jsonData, 0600)
	if err != nil {
		return fmt.Errorf("failed to write metric data into file. %v", err)
	}

	return nil
}

func (m *Metrics) load() error {
	if m.fileName == "" {
		return fmt.Errorf("filename must not be empty")
	}

	jsonData, err := os.ReadFile(m.fileName)
	if err != nil {
		return fmt.Errorf("failed to read metric data file. %v", err)
	}

	err = json.Unmarshal(jsonData, m.values)
	if err != nil {
		return fmt.Errorf("failed unmoarshal metric jason. %v", err)
	}

	return nil
}
