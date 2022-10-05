package fmb920

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/filipkroca/teltonikaparser"
	"github.com/halacs/haltonika/config"
	metrics2 "github.com/halacs/haltonika/metrics"
	"github.com/sirupsen/logrus"
	"net"
	"strings"
	"time"
)

type TeltonikaMessage struct {
	Decoded       teltonikaparser.Decoded
	SourceAddress string
}

/*
PacketArrivedCallback function used to report new decoded Teltonika packet.
If it returns with false, network connection will be closed. This can be used, for example, to reject unknown devices.
*/
type PacketArrivedCallback func(ctx context.Context, message TeltonikaMessage)

type Server struct {
	host         string
	port         int
	allowedIMEIs []string
	callback     PacketArrivedCallback
	metrics      metrics2.TeltonikaMetricsInterface
	ctx          context.Context
	localCtx     context.Context
	stopFunc     context.CancelFunc

	// To check if we receive a packet more times
	processedPackets map[string]time.Time
}

func NewServer(ctx context.Context, host string, port int, allowedIMEIs []string, metrics metrics2.TeltonikaMetricsInterface, callback PacketArrivedCallback) *Server {
	server := &Server{
		host:             host,
		port:             port,
		callback:         callback,
		ctx:              ctx,
		metrics:          metrics,
		allowedIMEIs:     allowedIMEIs,
		processedPackets: make(map[string]time.Time),
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

	go func() {
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

				decoded, err := teltonikaparser.Decode(&buffer)
				if err != nil {
					if s.metrics != nil {
						s.metrics.AddMalformedPackages(1)
					}

					log.Errorf("Malformed packet received. Ignoring packet. (%v)", err)
					continue
				}

				if s.metrics != nil {
					s.metrics.AddReceivedPackages(1)
				}

				if !s.isAllowedIMEI(decoded.IMEI) {
					log.Warningf("Packet rejected. %s IMEI is not on the allow list.", decoded.IMEI)

					if s.metrics != nil {
						s.metrics.AddRejectedPackages(1)
					}

					continue
				}

				err = s.sendBytes(listen, decoded.Response, remote)
				if err != nil {
					// just log the error and let the connection alive
					log.Errorf("Failed to send response for a packet. %v Continue.", err)
				}

				if s.isResentPackage(&buffer) {
					log.Warningf("Doubled packet received: %v", buffer)
				}

				// Send notification about the new decoded packet
				s.callback(s.ctx, TeltonikaMessage{
					Decoded:       decoded,
					SourceAddress: remote.String(),
				})
			}
		}
	}()

	return nil
}

// WARNING! Depends on the amount of actual incoming traffic, this might be a very resource intensive function!
func (s *Server) isResentPackage(pkg *[]byte) bool {
	log := config.GetLogger(s.ctx)

	hexBytes := hex.EncodeToString(*pkg)
	ts, ok := s.processedPackets[hexBytes]
	if ok {
		logrus.Warningf("Doubled package found. Last received at %v. Package: %v", ts, hexBytes)
		s.processedPackets[hexBytes] = time.Now()
		if s.metrics != nil {
			s.metrics.AddResentPackages(1)
		}
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

func (s *Server) isAllowedIMEI(imei string) bool {
	for _, actualIMEI := range s.allowedIMEIs {
		if strings.EqualFold(imei, actualIMEI) {
			return true
		}
	}
	return false
}

func (s *Server) receiveBytes(listen *net.UDPConn) (int, []byte, *net.UDPAddr, error) {
	log := config.GetLogger(s.ctx)

	buffer := make([]byte, 10*1024) // TODO find out the right buffer size which is not too big neither too small
	size, remote, err := listen.ReadFromUDP(buffer)
	if err != nil {
		return 0, buffer, remote, err
	}

	log.Debugf("%d bytes received from %v", size, remote)

	if s.metrics != nil {
		s.metrics.AddReceivedBytes(uint64(size))
	}

	return size, buffer[:size], remote, nil
}

func (s *Server) sendBytes(listen *net.UDPConn, data []byte, remote *net.UDPAddr) error {
	log := config.GetLogger(s.ctx)

	log.Tracef("Sending %d bytes to %v: %s", len(data), remote, hex.EncodeToString(data))

	size, err := listen.WriteToUDP(data, remote)
	if err != nil {
		return err
	}

	if s.metrics != nil {
		s.metrics.AddSentBytes(uint64(size))
		s.metrics.AddSentPackages(1)
	}

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
