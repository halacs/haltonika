package fmb920

import (
	"context"
	"encoding/hex"
	"github.com/halacs/haltonika/config"
	"github.com/halacs/haltonika/metrics"
	metrics2 "github.com/halacs/haltonika/metrics/impl"
	"github.com/sirupsen/logrus"
	"net"
	"testing"
)

var (
	allowedIMEIs = []string{
		"352094089397464",
	}
)

const (
	metricsFilename = "/tmp/haltonika.met"
)

func send(ctx context.Context, conn *net.UDPConn, data []byte) {
	log := config.GetLogger(ctx)

	_, err := conn.Write(data)
	if err != nil {
		log.Errorf("Write to server failed. %v\n", err.Error())
	}
}

func recv(ctx context.Context, conn *net.UDPConn) (int, []byte) {
	log := config.GetLogger(ctx)

	buffer := make([]byte, 1024)

	size, err := conn.Read(buffer)
	if err != nil {
		log.Errorf("Write to server failed. %v\n", err.Error())
	}

	return size, buffer
}

func startServer(ctx context.Context, metrics metrics.TeltonikaMetricsInterface, callback PacketArrivedCallback) {
	go func() {
		log := config.GetLogger(ctx)

		server := NewServer(ctx, "127.0.0.1", 9001, allowedIMEIs, metrics, callback)

		err := server.Start()
		if err != nil {
			log.Errorf("Failed to start Teltonika server. %v", err)
		}
	}()
}

func TestConnect(t *testing.T) {
	testCases := []struct {
		Name             string
		Request          string
		ExpectedResponse string
	}{
		{
			Name:             "ReceivingMessage1",
			Request:          "01e4cafe0128000f333532303934303839333937343634080400000163c803eb02010a2524c01d4a377d00d3012f130032421b0a4503f00150051503ef01510052005900be00c1000ab50008b60006426fd8cd3d1ece605a5400005500007300005a0000c0000007c70000000df1000059d910002d33c65300000000570000000064000000f7bf000000000000000163c803e6e8010a2530781d4a316f00d40131130031421b0a4503f00150051503ef01510052005900be00c1000ab50008b60005426fcbcd3d1ece605a5400005500007300005a0000c0000007c70000000ef1000059d910002d33b95300000000570000000064000000f7bf000000000000000163c803df18010a2536961d4a2e4f00d50134130033421b0a4503f00150051503ef01510052005900be00c1000ab50008b6000542702bcd3d1ece605a5400005500007300005a0000c0000007c70000001ef1000059d910002d33aa5300000000570000000064000000f7bf000000000000000163c8039ce2010a25d8d41d49f42c00dc0123120058421b0a4503f00150051503ef01510052005900be00c1000ab50009b60005427031cd79d8ce605a5400005500007300005a0000c0000007c700000019f1000059d910002d32505300000000570000000064000000f7bf000000000004",
			ExpectedResponse: "0005cafe010104",
		},
	}

	log := logrus.New()
	log.SetLevel(logrus.TraceLevel)
	cfg := config.NewConfig(log, nil, nil, nil) // only the logger is needed in this natsio
	ctx := context.WithValue(context.Background(), config.ContextConfigKey, cfg)

	// Initialize metrics collector
	metrics := metrics2.NewMetrics(ctx, metricsFilename)
	// Create callback function for decoded packets
	callbackFunc := func(ctx context.Context, message TeltonikaMessage) {
		log2 := config.GetLogger(ctx)
		log2.Infof("New decoded packet: %+v", message)
	}
	// Start server to be tested"cof
	startServer(ctx, metrics, callbackFunc)

	// Run all natsio cases as a separated network connection
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(test *testing.T) {
			data, err := hex.DecodeString(testCase.Request)
			if err != nil {
				t.Errorf("Incorrect natsio request data. %v", err)
			}

			udpAddr, err := net.ResolveUDPAddr("udp", "localhost:9001")
			if err != nil {
				t.Errorf("ResolveTCPAddr failed. %v", err)
			}

			clientConnection, err := net.DialUDP("udp", nil, udpAddr)
			if err != nil {
				t.Errorf("Dial failed. %v", err)
			}

			// Ensure network connection will be always closed
			defer func() {
				err := clientConnection.Close()
				if err != nil {
					t.Errorf("Failed to close network clientConnection. %v", err)
				}
			}()

			// Send request
			send(ctx, clientConnection, data)

			// Receive response
			size, buffer := recv(ctx, clientConnection)
			actualRespStr := hex.EncodeToString(buffer[:size])
			if actualRespStr != testCase.ExpectedResponse {
				test.Errorf("Wrong reponse! Expected: %v Actual: %v", testCase.ExpectedResponse, actualRespStr)
			}
		})
	}
}
