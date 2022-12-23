package fmb920

import (
	"context"
	"github.com/filipkroca/teltonikaparser"
	metrics2 "github.com/halacs/haltonika/metrics"
	"net"
	"sync"
	"time"
)

type TeltonikaMessage struct {
	Decoded       teltonikaparser.Decoded
	SourceAddress string
}

type DevicesWithTimeout struct {
	Imei      string
	Remote    *net.UDPAddr
	Timestamp time.Time
}

/*
PacketArrivedCallback function used to report new decoded Teltonika packet.
If it returns with false, network connection will be closed. This can be used, for example, to reject unknown devices.
*/
type PacketArrivedCallback func(ctx context.Context, message TeltonikaMessage)

type Server struct {
	wg           *sync.WaitGroup
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

	// Online devices by IMEI
	devicesByIMEI        sync.Map
	devicesByImeitimeout time.Duration

	commandResponses chan string
	commandRequests  chan string
}
