package uds

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
)

type MultiServerInterface interface {
	Stop() error
	StartServer(deviceID string, toDevice, fromDevice chan string) (*Server, error)
	StopServer(deviceID string) error
	StopAllServers() error
	KeepAlive(deviceID string) (found bool, err error)
	GetServer(deviceID string) (*Server, error)
}

type MultiServer struct {
	ctx      context.Context
	servers  map[string]*Server
	log      logrus.Logger
	basePath string
}

func NewMultiServer(ctx context.Context, basePath string) (*MultiServer, error) {
	ms := &MultiServer{
		ctx:      ctx,
		servers:  make(map[string]*Server),
		basePath: basePath,
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

func (ms *MultiServer) KeepAlive(deviceID string) (found bool, err error) {
	return false, nil // TODO implement
}

/*
func (ms *MultiServer) keepAliveChecker() error {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ms.log.Warningf("Ticker fired - NOT IMPLEMENTED YET")
				// TODO we should check here if there is a socket should be closed because of time out
			case <-ms.ctx.Done():
				return
			}
		}
	}()

	return nil
}
*/

func (ms *MultiServer) setServerForDevice(deviceID string, server *Server) {
	ms.servers[deviceID] = server
}

/*
TODO implement udsServer cleanup based on timeout with keep alive calls
func (ms *MultiServer) removeServer(deviceID string) error {
	_, found := ms.servers[deviceID]
	if !found {
		return fmt.Errorf("no UDS server found for %s device ID", deviceID)
	}

	delete(ms.servers, deviceID)

	return nil
}
*/

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
