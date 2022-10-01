package impl

import (
	"context"
	"testing"
)

const (
	metricsFilename = "/tmp/haltonika.met"
)

func TestPersistency(t *testing.T) {
	// Save

	m := Metrics{
		ctx:      context.Background(),
		fileName: metricsFilename,
		values: &persistentMetrics{
			SentBytes:         0,
			ReceivedBytes:     1,
			SentPackages:      2,
			ReceivedPackages:  3,
			MalformedPackages: 4,
			RejectedPackages:  5,
			ResentPackages:    7,
		},
	}

	err := m.save()
	if err != nil {
		t.Logf("Failed to save. %v", err)
		t.Fail()
	}

	// Load

	m2 := Metrics{
		ctx:      context.Background(),
		fileName: metricsFilename,
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
	m2.load()

	// Compare

	if m.GetMalformedPackages() != m2.GetMalformedPackages() ||
		m.GetReceivedBytes() != m2.GetReceivedBytes() ||
		m.GetReceivedPackages() != m2.GetReceivedPackages() ||
		m.GetSentBytes() != m2.GetSentBytes() ||
		m.GetSentPackages() != m2.GetSentPackages() ||
		m.GetRejectedPackages() != m2.GetRejectedPackages() ||
		m.GetResentPackages() != m2.GetResentPackages() {
		t.Logf("Excepted values: %+v, Actual values: %+v", m.values, m.values)
		t.Fail()
	}
}
