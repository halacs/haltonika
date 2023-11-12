package fmb920

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/filipkroca/teltonikaparser"
	"github.com/halacs/haltonika/config"
	metrics2 "github.com/halacs/haltonika/metrics"
	"net"
	"strings"
	"sync"
	"time"
)

func NewServer(ctx context.Context, wg *sync.WaitGroup, host string, port int, allowedIMEIs []string, metrics metrics2.TeltonikaMetricsInterface, callback PacketArrivedCallback) *Server {
	server := &Server{
		wg:                   wg,
		host:                 host,
		port:                 port,
		callback:             callback,
		ctx:                  ctx,
		metrics:              metrics,
		allowedIMEIs:         allowedIMEIs,
		processedPackets:     make(map[string]time.Time),
		devicesByIMEI:        sync.Map{},
		devicesByImeitimeout: 5 * time.Minute,
		commandResponses:     make(chan string),
		commandRequests:      make(chan string, 1),
	}

	return server
}

func (s *Server) Start() error {
	log := config.GetLogger(s.ctx)

	log.Infof("Start Teltonika server on %s:%d", s.host, s.port)

	s.localCtx, s.stopFunc = context.WithCancel(s.ctx)

	// NOTE: There are different protocols for TCP and UDP!
	// TLS on the of UDP is not possible.
	listen, err := net.ListenUDP("udp", &net.UDPAddr{
		Port: s.port,
		IP:   net.ParseIP(s.host),
	})
	if err != nil {
		return fmt.Errorf("failed to open listening socket. %v", err)
	}

	s.startPeriodicCleanupOnlineDevices()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		// close listener
		defer func() {
			err := listen.Close()
			if err != nil {
				log.Errorf("failed to close listening socket. %v", err)
			}
		}()

		// Reading incoming packets
		for {
			select {
			case <-s.localCtx.Done():
				return
			default:
				size, buffer, remote, err := s.receiveBytes(listen)
				if err != nil {
					log.Errorf("failed to read from connection. %v", err)
					return
				}

				log.Tracef("%d bytes long packet received: %s", size, hex.EncodeToString(buffer))

				// is it a heartbeat package?
				if size == 1 && strings.ToLower(hex.EncodeToString(buffer)) == "ff" {
					value, ok := s.getOnlineDeviceEndpoint(remote)
					if !ok {
						log.Warningf("Device not found in the map by %v remote. Ignoring FF package.", remote)
						continue
					}

					log.Debugf("Device with %s IMEI send FF package.", value.Imei)

					// Check if there is a command to be sent and if yes send it to the device who send FF just now
					select {
					case commandStr := <-s.commandRequests:
						log.Infof("Command to be sent: %s", commandStr)
						command, err := teltonikaparser.EncodeCommandRequest(commandStr)
						if err != nil {
							log.Errorf("Failed to encode command. %v", err)
							continue
						}

						err = s.sendBytes(listen, command, remote)
						if err != nil {
							log.Errorf("Failed to send commands's bytes out. %v", err)
						}
					default:
						log.Tracef("No command to be sent for this remote endpoint, for this device.")
					}

					continue
				}

				// Is it an AVL data package? Most of the packages should be AVL Data Package.
				decodedAvl, errAvl := teltonikaparser.Decode(&buffer)
				if errAvl != nil {
					// Is it a command response package?
					commandResponse, errCmd := teltonikaparser.DecodeCommandResponse(&buffer)
					if errCmd != nil {
						// Neither AVL Data Package nor Command Response Package
						log.Errorf("Malformed packet received. Neither AVL Data Packer nor Command Response packet. Ignoring packet. AVL parser: %v. Command response parser: %v", errAvl, errCmd)
						s.addMalformedPackages(1)
						continue
					}

					log.Tracef("Remote endpoint: %+v, Command response: %+v", remote, commandResponse)

					// Find out which device sent command response
					value, ok := s.getOnlineDeviceEndpoint(remote)
					if !ok {
						log.Errorf("No IMEI for %v remote endpoint. Drop package.", remote)
						s.addRejectedPackages(1)
						continue
					}

					log.Debugf("Get command response from device with %s IMEI. Remote endpoint: %v Repsonse: %v", value.Imei, remote, string(commandResponse.Response))
					s.addReceivedPackages(1) // Command Response Package !

					// Forward command response for further processing
					s.wg.Add(1)
					go func() { // TODO is this the right way to send it back? Not sure...
						defer s.wg.Done()

						s.commandResponses <- string(commandResponse.Response)
					}()

					continue
				}

				// Got an AVL Data Package!

				s.addReceivedPackages(1)

				if !s.isAllowedIMEI(decodedAvl.IMEI) {
					log.Warningf("Packet rejected. %s IMEI is not on the allow list.", decodedAvl.IMEI)

					s.addRejectedPackages(1)

					continue
				}

				s.markDeviceOnline(remote, decodedAvl.IMEI)

				// Send response for an AVL
				err = s.sendBytes(listen, decodedAvl.Response, remote)
				if err != nil {
					// just log the error and let the connection alive
					log.Errorf("Failed to send response for a packet. %v Continue.", err)
				}

				// Process received packet on a separated thread
				// TODO consider if all after receiving the packet can be done in a separated thread even the response sending
				s.wg.Add(1)
				go func() {
					defer s.wg.Done()

					if s.isResentPackage(&buffer) {
						log.Warningf("Doubled packet received: %v", buffer)
					}

					// Send notification about the new decodedAvl packet
					s.callback(s.ctx, TeltonikaMessage{
						Decoded:       decodedAvl,
						SourceAddress: remote.String(),
					})
				}()
			}
		}
	}()
	return nil
}

func (s *Server) receiveBytes(listen *net.UDPConn) (int, []byte, *net.UDPAddr, error) {
	log := config.GetLogger(s.ctx)

	buffer := make([]byte, 10*1024) // TODO find out the right buffer size which is not too big neither too small
	size, remote, err := listen.ReadFromUDP(buffer)
	if err != nil {
		return 0, buffer, remote, err
	}

	log.Debugf("%d bytes received from %v", size, remote)

	s.addReceivedBytes(uint64(size))

	return size, buffer[:size], remote, nil
}

func (s *Server) sendBytes(listen *net.UDPConn, data []byte, remote *net.UDPAddr) error {
	log := config.GetLogger(s.ctx)

	log.Tracef("Sending %d bytes to %v: %s", len(data), remote, hex.EncodeToString(data))

	size, err := listen.WriteToUDP(data, remote)
	if err != nil {
		return err
	}

	s.addSentBytes(uint64(size))
	s.addSentPackages(1)

	return nil
}

func (s *Server) Stop() error {
	if s.stopFunc == nil {
		return fmt.Errorf("server is not running")
	}

	s.stopFunc()
	s.stopFunc = nil
	return nil
}

func (s *Server) GetCommandResponseChannel() chan string {
	return s.commandResponses
}

func (s *Server) GetCommandRequestChannel() chan string {
	return s.commandRequests
}
