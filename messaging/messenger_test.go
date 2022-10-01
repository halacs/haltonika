package messaging

import (
	"context"
	"github.com/halacs/haltonika/config"
	"github.com/sirupsen/logrus"
	"strings"
	"testing"
)

func TestMessaging(t *testing.T) {
	testCases := []struct {
		Name    string
		Send    []string
		Receive []string
	}{
		{
			Name: "PassCase1",
			Send: []string{
				"one",
				"two",
			},
			Receive: []string{
				"one",
				"two",
			},
		},
		{
			Name: "FailedCase1",
			Send: []string{
				"one",
			},
			Receive: []string{
				"one",
			},
		},
		{
			Name:    "FailedCase2",
			Send:    []string{},
			Receive: []string{},
		},
	}

	log := logrus.New()
	log.SetLevel(logrus.TraceLevel)
	cfg := config.NewConfig(log, nil, nil, nil) // only the logger is needed in this natsio

	// Run all natsio cases as a separated network connection
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(test *testing.T) {
			receivedMessages := make([]string, 0)

			ctx := context.WithValue(context.Background(), config.ContextConfigKey, cfg)
			messenger := NewMessaging(ctx)
			messenger.Subscribe(func(data interface{}) error {
				receivedMessages = append(receivedMessages, data.(string))
				return nil
			})

			for _, v := range testCase.Send {
				messenger.Publish(v)
			}

			if strings.Join(receivedMessages, "") != strings.Join(testCase.Receive, "") {
				test.Fail()
			}
		})
	}
}
