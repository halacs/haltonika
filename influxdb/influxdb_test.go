package influxdb

import (
	"context"
	"encoding/hex"
	"github.com/filipkroca/teltonikaparser"
	cfg "github.com/halacs/haltonika/config"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestConnect(t *testing.T) {
	testCases := []struct {
		Name    string
		Request string
	}{
		{
			Name:    "StorePacket1",
			Request: "01e4cafe0128000f333532303934303839333937343634080400000163c803eb02010a2524c01d4a377d00d3012f130032421b0a4503f00150051503ef01510052005900be00c1000ab50008b60006426fd8cd3d1ece605a5400005500007300005a0000c0000007c70000000df1000059d910002d33c65300000000570000000064000000f7bf000000000000000163c803e6e8010a2530781d4a316f00d40131130031421b0a4503f00150051503ef01510052005900be00c1000ab50008b60005426fcbcd3d1ece605a5400005500007300005a0000c0000007c70000000ef1000059d910002d33b95300000000570000000064000000f7bf000000000000000163c803df18010a2536961d4a2e4f00d50134130033421b0a4503f00150051503ef01510052005900be00c1000ab50008b6000542702bcd3d1ece605a5400005500007300005a0000c0000007c70000001ef1000059d910002d33aa5300000000570000000064000000f7bf000000000000000163c8039ce2010a25d8d41d49f42c00dc0123120058421b0a4503f00150051503ef01510052005900be00c1000ab50009b60005427031cd79d8ce605a5400005500007300005a0000c0000007c700000019f1000059d910002d32505300000000570000000064000000f7bf000000000004",
		},
	}

	log := logrus.New()
	influxConfig := &cfg.InfluxConfig{
		Url:         cfg.DefaultInfluxDbUrl,
		Username:    cfg.DefaultInfluxDbUserName,
		Password:    cfg.DefaultInfluxDbPassword,
		Database:    cfg.DefaultInfluxDbDatabaseName,
		Measurement: cfg.DefaultInfluxDbMeasurementName,
	}
	config := cfg.NewConfig(log, influxConfig, nil, nil) // only the logger is needed in this natsio
	ctx := context.WithValue(context.Background(), cfg.ContextConfigKey, config)

	// Run all natsio cases as a separated network connection
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(test *testing.T) {
			client := NewConnection(ctx, influxConfig)
			err := client.Connect()
			if err != nil {
				logrus.Errorf("Failed to connect to influxdb. %v", err)
			}

			byteRequest, err := hex.DecodeString(testCase.Request)
			if err != nil {
				test.Logf("Incorrect test case input! Failed do convert hex string to byte array.")
				test.Fail()
			}

			decoded, err := teltonikaparser.Decode(&byteRequest)
			if err != nil {
				test.Logf("Incorrect test case input! Failed to decode haltonika packet.")
				test.Fail()
			}

			extraTags := map[string]string{
				SourceTag: "127.0.0.1:1234",
			}

			err = client.InsertMessage(decoded, extraTags)
			if err != nil {
				test.Logf("Error: %v", err)
				test.Fail()
			}
		})
	}
}
