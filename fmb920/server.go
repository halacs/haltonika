package fmb920

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/filipkroca/teltonikaparser"
	"github.com/halacs/haltonika/config"
	metrics2 "github.com/halacs/haltonika/metrics"
	"github.com/halacs/haltonika/uds"
	"net"
	"slices"
	"strings"
	"sync"
	"time"
)

func NewServer(ctx context.Context, wg *sync.WaitGroup, host string, port int, allowedIMEIs []string, udsServer uds.MultiServerInterface, metrics metrics2.TeltonikaMetricsInterface, callback PacketArrivedCallback) *Server {
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
		//commandResponses:     make(chan string),
		//commandRequests:      make(chan string, 1),
		udsServer: udsServer,
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
		defer func() {
			s.wg.Done()
		}()

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
				if size <= 0 {
					continue // context might be cancelled
				}
				if err != nil {
					log.Errorf("failed to read from connection. %v", err)
					return
				}

				log.Tracef("%d bytes long packet received: %s", size, hex.EncodeToString(buffer))

				// is it a heartbeat package?
				if size == 1 && strings.ToLower(hex.EncodeToString(buffer)) == "ff" {
					value, ok := s.getOnlineDeviceEndpoint(remote)
					if !ok {
						log.Debugf("Device not found in the map by %v remote. Ignoring FF package.", remote)
						continue
					}

					log.Debugf("Device with %s IMEI sent FF package.", value.Imei)

					commandRequests, _, err := s.GetCommandRequestChannel(value.Imei)
					if err != nil {
						log.Errorf("Failed to send command response to channel. %v", err)
					} else {
						// Check if there is a command to be sent and if yes send it to the device who send FF just now
						select {
						case commandStr := <-commandRequests:
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
						defer func() {
							s.wg.Done()
						}()

						commandResponses, _, err := s.GetCommandResponseChannel(value.Imei)
						if err != nil {
							log.Errorf("Failed to send command response to channel. %v", err)
						} else {
							commandResponses <- string(commandResponse.Response)
						}
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

				server, _ := s.udsServer.GetServer(decodedAvl.IMEI) // UdsServer is already started
				if server == nil {
					socketPath, err := s.startNewUdsServer(decodedAvl.IMEI)
					if err != nil {
						log.Errorf("Failed to start new UDS server. %v", err)
					} else {
						log.Infof("New UDS server has been started for %s device at %s", decodedAvl.IMEI, socketPath)
					}
				} else {
					socketPath, err := server.GetSocketPath()
					if err != nil {
						log.Errorf("%v", err)
					}
					log.Tracef("UdsServer for %s device is running at %s. %v", decodedAvl.IMEI, socketPath, server)
				}

				err = s.markDeviceOnline(remote, decodedAvl.IMEI)
				if err != nil {
					log.Errorf("Failed to mark device online. %v", err)
				}

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
					defer func() {
						s.wg.Done()
					}()

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

func (s *Server) startNewUdsServer(imei string) (string, error) {
	log := config.GetLogger(s.ctx)

	requestChannel, _, err := s.GetCommandRequestChannel(imei)
	if err != nil {
		return "", fmt.Errorf("failed to start new UDS server for %s device: response channel error. %v", imei, err)
	}

	responseChannel, _, err := s.GetCommandResponseChannel(imei)
	if err != nil {
		return "", fmt.Errorf("failed to start new UDS server for %s device: request channel error. %v", imei, err)
	}

	server, err := s.udsServer.StartServer(imei, requestChannel, responseChannel)
	if err != nil {
		return "", fmt.Errorf("failed to start new UDS server for %s device. %v", imei, err)
	}

	socketPath, err := server.GetSocketPath()
	if err != nil {
		log.Errorf("Failed to get socket path. %v", err)
	}

	return socketPath, nil
}

func (s *Server) receiveBytes(listen *net.UDPConn) (int, []byte, *net.UDPAddr, error) {
	log := config.GetLogger(s.ctx)

	for {
		select {
		case <-s.ctx.Done():
			log.Tracef("receiveBytes returns because of context cancelled")
			return 0, nil, nil, fmt.Errorf("context cancelled")
		default:
			buffer := make([]byte, 10*1024)                                 // TODO find out the right buffer size which is not too big neither too small
			err := listen.SetReadDeadline(time.Now().Add(10 * time.Second)) // needed because of context cancellation
			if err != nil {
				return 0, buffer, nil, fmt.Errorf("failed to set context deadline. %v", err)
			}
			size, remote, err := listen.ReadFromUDP(buffer)
			if err != nil {
				//return 0, buffer, remote, err
				continue
			}

			log.Debugf("%d bytes received from %v", size, remote)

			s.addReceivedBytes(uint64(size)) // #nosec G115

			return size, buffer[:size], remote, nil
		}
	}
}

func (s *Server) sendBytes(listen *net.UDPConn, data []byte, remote *net.UDPAddr) error {
	log := config.GetLogger(s.ctx)

	log.Tracef("Sending %d bytes to %v: %s", len(data), remote, hex.EncodeToString(data))

	size, err := listen.WriteToUDP(data, remote)
	if err != nil {
		return err
	}

	s.addSentBytes(uint64(size)) // #nosec G115
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

func (s *Server) GetCommandResponseChannel(imei string) (ch chan string, created bool, err error) {
	log := config.GetLogger(s.ctx)

	created = false

	if !slices.Contains(s.allowedIMEIs, imei) {
		return nil, created, fmt.Errorf("%s device ID is not on the allowed list", imei)
	}

	newChan := make(chan string)
	c, loaded := s.responseCommandChannelsByIMEI.LoadOrStore(imei, newChan) // TODO do we need to implement a cleanup?
	if !loaded {
		log.Debugf("New command response channel was made for %s device.", imei)
		created = true
	}

	return c.(chan string), created, nil
}

func (s *Server) GetCommandRequestChannel(imei string) (ch chan string, created bool, err error) {
	log := config.GetLogger(s.ctx)

	created = false

	if !slices.Contains(s.allowedIMEIs, imei) {
		return nil, created, fmt.Errorf("%s device ID is not on the allowed list", imei)
	}

	newChan := make(chan string)
	c, loaded := s.requestCommandChannelsByIMEI.LoadOrStore(imei, newChan) // TODO do we need to implement a cleanup?
	if !loaded {
		log.Debugf("New command request channel was made for %s device.", imei)
		created = true
	}

	return c.(chan string), created, nil
}
