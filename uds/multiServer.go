package uds

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

const (
	deleteIfOlderThen  = 1 * time.Hour
	checkLastSeenEvery = 10 * time.Second
)

type MultiServerInterface interface {
	Stop() error
	StartServer(deviceID string, toDevice, fromDevice chan string) (*Server, error)
	StopServer(deviceID string) error
	StopAllServers() error
	KeepAlive(deviceID string)
	GetServer(deviceID string) (*Server, error)
}

type MultiServer struct {
	ctx      context.Context
	servers  map[string]*Server
	log      logrus.Logger
	basePath string
	lastSeen sync.Map
	wg       *sync.WaitGroup
}

func NewMultiServer(ctx context.Context, basePath string, wg *sync.WaitGroup) (*MultiServer, error) {
	ms := &MultiServer{
		ctx:      ctx,
		servers:  make(map[string]*Server),
		basePath: basePath,
		wg:       wg,
	}

	return ms, nil
}

func (ms *MultiServer) StartServer(deviceID string, toDevice, fromDevice chan string) (*Server, error) {
	udsServer := NewUdsServer(ms.ctx, deviceID, ms.basePath)

	err := udsServer.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start UDS server. %v", err)
	}

	// https://wiki.teltonika-gps.com/view/FMB920_SMS/GPRS_Commands
	udsServer.SetFromDeviceChannel(fromDevice)
	udsServer.SetToDeviceChannel(toDevice)

	ms.setServerForDevice(deviceID, udsServer)

	ms.keepAliveChecker()

	return udsServer, nil
}

func (ms *MultiServer) StopServer(deviceID string) error {
	server, err := ms.GetServer(deviceID)
	if err != nil {
		return err
	}

	err = server.Stop()
	if err != nil {
		return err
	}

	return nil
}

func (ms *MultiServer) Stop() error {
	return ms.StopAllServers()
}

func (ms *MultiServer) StopAllServers() error {
	ok := true

	for _, server := range ms.getAllServers() {
		err := server.Stop()
		if err != nil {
			ms.log.Errorf("failed to stop UDS server. %v", err)
			ok = false
		}
	}

	if !ok {
		return fmt.Errorf("at least one UDS server failed to stop")
	}

	return nil
}

func (ms *MultiServer) KeepAlive(deviceID string) {
	key := deviceID
	value := time.Now()
	ms.lastSeen.Store(key, value)
}

func (ms *MultiServer) keepAliveChecker() {
	ms.wg.Add(1)
	go func() {
		defer ms.wg.Done()

		ticker := time.NewTicker(checkLastSeenEvery)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ms.lastSeen.Range(func(key, value any) bool {
					deviceID := key.(string)
					lastSeenTimestamp := value.(time.Time)

					server, err := ms.GetServer(deviceID)
					if err != nil {
						ms.log.Errorf("Failed to close expired Unix Domain Socket. %v", err)
					}

					if server != nil && server.IsActive() {
						err2 := ms.removeServer(deviceID)
						if err2 != nil {
							ms.log.Errorf("Failed to close expired Unix Domain Socket. %v", err)
						}
					}

					if time.Now().Add(-1 * deleteIfOlderThen).Before(lastSeenTimestamp) {
						ms.lastSeen.Delete(deviceID)
						ms.log.Debugf("Device expired: %s", deviceID)
					}

					return true // continue
				})
			case <-ms.ctx.Done():
				return
			}
		}
	}()
}

func (ms *MultiServer) removeServer(deviceID string) error {
	server, found := ms.servers[deviceID]
	if !found {
		return fmt.Errorf("no UDS server found for %s device ID", deviceID)
	}

	if server.IsActive() {
		err := server.Stop()
		if err != nil {
			return fmt.Errorf("failed to stop server. %v", err)
		}
	}

	delete(ms.servers, deviceID)

	return nil
}

func (ms *MultiServer) setServerForDevice(deviceID string, server *Server) {
	ms.servers[deviceID] = server
}

func (ms *MultiServer) GetServer(deviceID string) (*Server, error) {
	server, found := ms.servers[deviceID]
	if !found {
		return nil, fmt.Errorf("no UDS server found for %s device ID", deviceID)
	}

	return server, nil
}

func (ms *MultiServer) getAllServers() map[string]*Server {
	return ms.servers
}
