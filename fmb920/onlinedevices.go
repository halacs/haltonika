package fmb920

import (
	"github.com/halacs/haltonika/config"
	"net"
	"time"
)

func (s *Server) markDeviceOnline(remote *net.UDPAddr, imei string) {
	s.devicesByIMEI.Store(remote.String(), &DevicesWithTimeout{
		Imei:      imei,
		Remote:    remote,
		Timestamp: time.Now(),
	})
}

func (s *Server) cleanupDevicesOnline() {
	log := config.GetLogger(s.ctx)

	s.devicesByIMEI.Range(func(k, value any) bool {
		item := value.(*DevicesWithTimeout)
		if !item.Timestamp.Before(time.Now().Add(s.devicesByImeitimeout)) {
			s.devicesByIMEI.Delete(k)
			log.Debugf("Device with %s IMEI has been removed from map of online devices. Item's timestamp: %v", item.Imei, item.Timestamp)
		}
		return true // continue map iteration
	})
}

// Periodically cleanup map of connected devices
func (s *Server) startPeriodicCleanupOnlineDevices() {
	go func() {
		ticker := time.NewTicker(s.devicesByImeitimeout)
		defer ticker.Stop()

		for {
			select {
			case <-s.localCtx.Done():
				return
			case <-ticker.C:
				s.cleanupDevicesOnline()
			}
		}
	}()
}

func (s *Server) getOnlineDeviceEndpoint(remote *net.UDPAddr) (*DevicesWithTimeout, bool) {
	value, ok := s.devicesByIMEI.Load(remote.String())
	if !ok {
		return nil, false
	}

	return value.(*DevicesWithTimeout), true
}
