package fmb920

import (
	"github.com/halacs/haltonika/config"
	"net"
	"time"
)

func (s *Server) markDeviceOnline(remote *net.UDPAddr, listener *net.UDPConn, imei string) error {
	s.devices.Store(remote.String(), &DevicesWithTimeout{
		Imei:      imei,
		Remote:    remote,
		Listener:  listener,
		Timestamp: time.Now(),
	})

	s.udsServer.KeepAlive(imei)

	return nil
}

func (s *Server) cleanupDevicesOnline() {
	log := config.GetLogger(s.ctx)

	s.devices.Range(func(k, value any) bool {
		item := value.(*DevicesWithTimeout)
		if !item.Timestamp.Before(time.Now().Add(s.devicesByImeitimeout)) {
			s.devices.Delete(k)
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

func (s *Server) getOnlineDevice(imei string) (*DevicesWithTimeout, bool) {
	var device *DevicesWithTimeout

	// TODO: consider to store devices by IMEI to be more efficient but I have not much motivation to do so right now :)
	s.devices.Range(func(k, value any) bool {
		item := value.(*DevicesWithTimeout)
		if item.Imei == imei {
			device = item
			return true // do NOT continue map iteration
		}
		return true // continue map iteration
	})

	return device, device != nil
}

func (s *Server) getOnlineDeviceEndpoint(remote *net.UDPAddr) (*DevicesWithTimeout, bool) {
	value, ok := s.devices.Load(remote.String())
	if !ok {
		return nil, false
	}

	return value.(*DevicesWithTimeout), true
}
