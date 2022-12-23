package fmb920

import (
	"fmt"
	"github.com/filipkroca/teltonikaparser"
	"github.com/halacs/haltonika/config"
	"time"
)

func (s *Server) SendCommand(imei, command string, timeout time.Duration) (string, error) {
	log := config.GetLogger(s.ctx)

	request, err := teltonikaparser.EncodeCommandRequest(command)
	if err != nil {
		return "", fmt.Errorf("failed to encode command request. %v", err)
	}

	log.Debugf("IMEI: %s, Command: %s, Timeout: %v, Request: %v", imei, command, timeout, request)

	/*
		remote, ok := s.devicesByIMEI.Load(imei)
		if !ok {
			return "", fmt.Errorf("device with %s IMEI looks offline", imei)
		}
	*/

	// TODO send command
	//s.sendBytes(remote, request)

	// Waiting for response with timeout
	// TODO waiting for response with timeout somehow

	response := ""

	return response, nil
}
