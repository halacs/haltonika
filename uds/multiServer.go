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
	log      *logrus.Logger
	basePath string
	lastSeen sync.Map
	wg       *sync.WaitGroup
}

func NewMultiServer(ctx context.Context, basePath string, log *logrus.Logger) (*MultiServer, error) {
	var wg sync.WaitGroup

	ms := &MultiServer{
		ctx:      ctx,
		servers:  make(map[string]*Server),
		basePath: basePath,
		wg:       &wg,
		log:      log,
	}

	ms.keepAliveChecker()

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
	ms.log.Tracef("multiServer keep alive: %s", deviceID)
}

func (ms *MultiServer) keepAliveChecker() {
	ms.wg.Add(1)
	go func() {
		defer ms.wg.Done()

		ms.log.Debugf("UDSServer: keep alive checker started")

		ticker := time.NewTicker(checkLastSeenEvery)
		defer func() {
			ticker.Stop()
		}()

		for {
			select {
			case <-ticker.C:
				ms.log.Tracef("UDSServer: checking keep alive timestamp")

				ms.lastSeen.Range(func(key, value any) bool {
					deviceID := key.(string)
					lastSeenTimestamp := value.(time.Time)

					now := time.Now()
					toCompareDate := lastSeenTimestamp.Add(deleteIfOlderThen)
					if toCompareDate.Before(now) { // Do last keep alive too old?
						ms.log.Infof("%v (%d) is too old (max %v allowed - %v == %d). UDS of %s device is going to be deleted. Now: %v (%d)", lastSeenTimestamp, lastSeenTimestamp.Unix(), deleteIfOlderThen, toCompareDate, toCompareDate.Unix(), deviceID, now, now.Unix())

						// Stop UDS server for the given device
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

						// Drop keep alive data
						ms.lastSeen.Delete(deviceID)
						ms.log.Debugf("Device expired: %s", deviceID)
					}

					return true // continue with rest of the keep alive timestamps
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
