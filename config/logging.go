package config

import (
	"github.com/sirupsen/logrus"
	"os"
)

func NewLogger() *logrus.Logger {
	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		//ForceColors:   true,
		DisableColors: false,
		FullTimestamp: true,
	})
	log.SetReportCaller(false)
	log.SetLevel(logrus.InfoLevel)
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	return log
}
