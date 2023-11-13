package config

import (
	"context"
	"github.com/sirupsen/logrus"
)

type Config struct {
	log             *logrus.Logger
	influxConfig    *InfluxConfig
	teltonikaConfig *TeltonikaConfig
	metricsConfig   *MetricsConfig
	udsServerConfig *UdsServerConfig
}

func NewConfig(log *logrus.Logger, influxConfig *InfluxConfig, teltonikaConfig *TeltonikaConfig, metricsConfig *MetricsConfig, udsServerConfig *UdsServerConfig) *Config {
	return &Config{
		log:             log,
		influxConfig:    influxConfig,
		teltonikaConfig: teltonikaConfig,
		metricsConfig:   metricsConfig,
		udsServerConfig: udsServerConfig,
	}
}

func (c *Config) GetInfluxConfig() *InfluxConfig {
	return c.influxConfig
}

func (c *Config) GetTeltonikaConfig() *TeltonikaConfig {
	return c.teltonikaConfig
}

func (c *Config) GetMetricsConfig() *MetricsConfig {
	return c.metricsConfig
}

func (c *Config) GetUdsServerConfig() *UdsServerConfig {
	return c.udsServerConfig
}

func (c *Config) GetLogger() *logrus.Logger {
	return c.log
}

func GetLogger(ctx context.Context) *logrus.Logger {
	config := ctx.Value(ContextConfigKey).(*Config)
	return config.GetLogger()
}
