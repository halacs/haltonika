package uds

import (
	"bytes"
	"context"
	"fmt"
	"github.com/halacs/haltonika/config"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
)

const (
	protocol = "unix"
)

type Server struct {
	ctx               context.Context
	deviceConnections []net.Conn
	quit              chan interface{}
	wg                sync.WaitGroup
	listener          net.Listener
	fromDeviceChannel chan string
	toDeviceChannel   chan string
	log               *logrus.Entry
	basePath          string
	deviceID          string
}

func NewUdsServer(ctx context.Context, deviceID string, basePath string) *Server {
	log := config.GetLogger(ctx).WithField("deviceID", deviceID)

	return &Server{
		ctx:               ctx,
		quit:              make(chan interface{}),
		wg:                sync.WaitGroup{},
		log:               log,
		basePath:          basePath,
		deviceID:          deviceID,
		deviceConnections: make([]net.Conn, 0),
	}
}

func (us *Server) GetDeviceID() string {
	return us.deviceID
}

func (us *Server) forwardMessageToUser(message string) {
	us.log.Infof("Device to user: %s", message)

	connections := us.getDeviceConnections()

	for _, c := range connections {
		_, err := c.Write([]byte(message + "\n"))
		if err != nil {
			us.log.Errorf("Failed to send message to UDS connectiuion. %v", err)
		}
	}
}

func (us *Server) forwardMessageToDevice(message string) error {
	us.log.Infof("User to device: %s", message)

	c, err := us.getToDeviceChannel()
	if err != nil {
		return err
	}

	c <- message
	return nil
}

func (us *Server) getUdsName() (string, error) {
	if us.deviceID == "" {
		return "", fmt.Errorf("deviceID must be specified")
	}

	path := filepath.Join(us.basePath, fmt.Sprintf("haltonika-%s", us.deviceID))
	return path, nil
}

func (us *Server) getDeviceConnections() []net.Conn {
	return us.deviceConnections
}

func (us *Server) addDeviceConnection(conn net.Conn) {
	// Check if connection is already there
	for _, c := range us.deviceConnections {
		if c == conn {
			return // found, nothing to do
		}
	}

	us.deviceConnections = append(us.deviceConnections, conn)
}

func (us *Server) removeDeviceConnections(conn net.Conn) error {
	for i, c := range us.deviceConnections {
		if c == conn {
			us.deviceConnections[i] = us.deviceConnections[len(us.deviceConnections)-1]
			us.deviceConnections = us.deviceConnections[:len(us.deviceConnections)-1]
			return nil
		}
	}

	return fmt.Errorf("connection not found")
}

func (us *Server) getFromDeviceChannel() (chan string, error) {
	if us.fromDeviceChannel == nil {
		return nil, fmt.Errorf("reqested channel not found")
	}

	return us.fromDeviceChannel, nil
}

func (us *Server) SetFromDeviceChannel(c chan string) {
	us.fromDeviceChannel = c
	us.log.Debugf("Device FROM channel has been set")
}

func (us *Server) getToDeviceChannel() (chan string, error) {
	if us.toDeviceChannel == nil {
		return nil, fmt.Errorf("reqested channel not found")
	}

	return us.toDeviceChannel, nil
}

func (us *Server) SetToDeviceChannel(c chan string) {
	us.toDeviceChannel = c
	us.log.Debugf("Device TO channel has been set")
}

func (us *Server) Stop() error {
	close(us.quit)

	err := us.listener.Close()
	if err != nil {
		us.log.Errorf("Failed to close listener. %v", err)
	}

	err = us.removeUdsSocket()
	if err != nil {
		us.log.Errorf("Failed to remove socket file. %v", err)
	}

	us.wg.Wait()

	return err
}

func (us *Server) removeUdsSocket() error {
	sockAddr, err := us.getUdsName()
	if err != nil {
		return err
	}

	_, err = os.Stat(sockAddr)
	if err == nil {
		if err := os.RemoveAll(sockAddr); err != nil {
			return err
		}
	}

	return nil
}

func (us *Server) Start() error {
	sockAddr, err := us.getUdsName()
	if err != nil {
		return err
	}

	// Remove UDS if exists in the file system
	err = us.removeUdsSocket()
	if err != nil {
		us.log.Errorf("Failed to remove socket file. %v", err)
	}

	// Open UDS
	us.log.Infof("Opening socket: %s://%s", protocol, sockAddr)
	us.listener, err = net.Listen(protocol, sockAddr)
	if err != nil {
		return fmt.Errorf("failed to open socket. %v", err)
	}

	us.wg.Add(1)
	go us.acceptConnections()

	return nil
}

func (us *Server) acceptConnections() {
	defer us.wg.Done()

	// Device to sockets (one to many)
	us.wg.Add(1)
	go func() {
		us.handleChannelToSocketDirection()
		us.wg.Done()
	}()

	for {
		conn, err := us.listener.Accept()
		if err != nil {
			select {
			case <-us.quit:
				return
			default:
				us.log.Errorf("failed to accept UDS connection. %v", err)
			}
		} else {
			us.wg.Add(1)
			go func() { // sockets to device (many to one)
				us.addDeviceConnection(conn)
				us.handleSocketToChannelDirection(conn)
				err := us.removeDeviceConnections(conn)
				if err != nil {
					us.log.Errorf("%v", err)
				}
				us.wg.Done()
			}()
		}

		us.log.Infof("New UDS connection accepted")
	}
}

func (us *Server) handleSocketToChannelDirection(conn net.Conn) {
	var message bytes.Buffer

	for {
		buffer := make([]byte, 1)
		_, err := conn.Read(buffer)
		if err != nil {
			if err == io.EOF {
				us.log.Infof("UDS socket terminated.")
				return // connection has been closed
			}
			if err != nil {
				us.log.Errorf("Failed to read. %s", err)
			}
			return
		}

		if buffer[0] == '\n' {
			msg := message.String()
			err := us.forwardMessageToDevice(msg)
			if err != nil {
				us.log.Errorf("Failed to forward message to device. %v Message: %s", err, msg)
			}
			message.Reset()
		} else {
			_, err = message.Write(buffer)
			if err != nil {
				us.log.Errorf("Failed to write character into 'message' buffer. %v", err)
			}
		}
	}
}

func (us *Server) handleChannelToSocketDirection() {
	for {
		ch, err := us.getFromDeviceChannel()
		if err != nil {
			us.log.Debugf("No channel for %s device!", us.GetDeviceID())
		} else {
			select {
			case <-us.quit:
				return
			case message := <-ch:
				us.forwardMessageToUser(message)
			}
		}
	}
}
