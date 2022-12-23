package fmb920

import (
	"encoding/hex"
	"github.com/halacs/haltonika/config"
	"github.com/sirupsen/logrus"
	"time"
)

func (s *Server) addReceivedBytes(count uint64) {
	if s.metrics != nil {
		s.metrics.AddReceivedBytes(count)
	}
}

func (s *Server) addReceivedPackages(count uint64) {
	if s.metrics != nil {
		s.metrics.AddReceivedPackages(count)
	}
}

func (s *Server) addSentBytes(count uint64) {
	if s.metrics != nil {
		s.metrics.AddSentBytes(count)
	}
}

func (s *Server) addSentPackages(count uint64) {
	if s.metrics != nil {
		s.metrics.AddSentPackages(count)
	}
}

func (s *Server) addRejectedPackages(count uint64) {
	if s.metrics != nil {
		s.metrics.AddRejectedPackages(1)
	}
}

func (s *Server) addMalformedPackages(count uint64) {
	if s.metrics != nil {
		s.metrics.AddMalformedPackages(count)
	}
}

func (s *Server) addResentPackages(count uint64) {
	if s.metrics != nil {
		s.metrics.AddResentPackages(count)
	}
}

// WARNING! Depends on the amount of actual incoming traffic, this might be a very resource intensive function!
func (s *Server) isResentPackage(pkg *[]byte) bool {
	log := config.GetLogger(s.ctx)

	hexBytes := hex.EncodeToString(*pkg)
	ts, ok := s.processedPackets[hexBytes]
	if ok {
		logrus.Warningf("Doubled package found. Last received at %v. Package: %v", ts, hexBytes)
		s.processedPackets[hexBytes] = time.Now()
		s.addResentPackages(1)
		return true
	}

	for p, ts := range s.processedPackets {
		if !ts.Before(time.Now().Add(-1 * time.Hour)) {
			delete(s.processedPackets, p)
			log.Tracef("Packet removed from processed packet.")
		}
	}

	s.processedPackets[hexBytes] = time.Now()

	return false
}
