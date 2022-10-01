package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/halacs/haltonika/config"
	"github.com/halacs/haltonika/fmb920"
	influxdb2 "github.com/halacs/haltonika/influxdb"
	"github.com/halacs/haltonika/messaging"
	m "github.com/halacs/haltonika/metrics"
	mi "github.com/halacs/haltonika/metrics/impl"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"strings"
)

func parseConfig() *config.Config {
	// Initialize logger
	log := config.NewLogger()

	// Read configuration
	viper.SetConfigName("cfg")                                     // Name of cfg file (without extension)
	viper.SetConfigType("yaml")                                    // REQUIRED if the cfg file does not have the extension in the name
	viper.AddConfigPath(fmt.Sprintf("/etc/%s/", config.AppName))   // path to look for the cfg file in
	viper.AddConfigPath(fmt.Sprintf("$HOME/.%s/", config.AppName)) // call multiple times to add many search paths
	viper.AddConfigPath(".")                                       // Optionally look for cfg in the working directory
	viper.SetEnvPrefix(config.ViperEnvPrefix)
	viper.AutomaticEnv() // Use environment variables if defined

	err := viper.ReadInConfig() // Find and read the cfg file
	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		log.Infof("Config file was not found. Using defaults.")
	} else {
		log.Fatalf("Failed to parse cfg file. %v", err)
	}

	// General configs
	flag.Bool(config.Debug, config.DefaultDebug, "Set log level to debug")
	flag.Bool(config.Verbose, config.DefaultVerbose, "Set log level to verbose")
	flag.String(config.AllowedIMEIs, config.DefaultAllowedIMEIs, "IMEI identifiers needs to be processed. Separated by comma. Example: 123456789012345,123456789012345,123456789012345")
	// InfluxDB client configs
	flag.String(config.InfluxConfigUrl, config.DefaultInfluxDbUrl, "URL of InfluxDB server")
	flag.String(config.InfluxConfigUsername, config.DefaultInfluxDbUserName, "InfluxDB username")
	flag.String(config.InfluxConfigPassword, config.DefaultInfluxDbPassword, "InfluxDB password")
	flag.String(config.InfluxConfigDatabase, config.DefaultInfluxDbDatabaseName, "InfluxDB database name")
	flag.String(config.InfluxConfigMeasurement, config.DefaultInfluxDbMeasurementName, "Name of the Influxdb measurement")
	// Teltonika server configs
	flag.String(config.TeltonikaListeningIp, config.DefaultTeltonikaListeningIP, "Teltonika server listening IP address (IPv4 or IPv6)")
	flag.Int(config.TeltonikaListeningPort, config.DefaultTeltonikaListeningPort, "Teltonika server listening UDP port")
	// Metrics server configs
	flag.String(config.MetricsListeningIp, config.DefaultMetricsListeningIP, "Metrics server listening IP address (IPv4 or IPv6)")
	flag.Int(config.MetricsListeningPort, config.DefaultMetricsListeningPort, "Metrics server listening port")
	flag.String(config.MetricsTeltonikaMetricsFileName, config.DefaultMetricsTeltonikaMetricsFileName, "File where metrics are written")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	verbose := viper.GetBool(config.Verbose)
	debug := viper.GetBool(config.Debug)
	if verbose {
		log.SetLevel(logrus.TraceLevel)
		log.Warningf("Active log level: %s", log.GetLevel())
	} else if debug {
		log.SetLevel(logrus.DebugLevel)
		log.Warningf("Active log level: %s", log.GetLevel())
	}

	// Initialize cfg
	influxConfig := &config.InfluxConfig{
		Url:         viper.GetString(config.InfluxConfigUrl),
		Username:    viper.GetString(config.InfluxConfigUsername),
		Password:    viper.GetString(config.InfluxConfigPassword),
		Database:    viper.GetString(config.InfluxConfigDatabase),
		Measurement: viper.GetString(config.InfluxConfigMeasurement),
	}

	allowedIMEIs := strings.Split(viper.GetString(config.AllowedIMEIs), ",")

	teltonikaConfig := &config.TeltonikaConfig{
		Host:         viper.GetString(config.TeltonikaListeningIp),
		Port:         viper.GetInt(config.TeltonikaListeningPort),
		AllowedIMEIs: allowedIMEIs,
	}

	metricsConfig := &config.MetricsConfig{
		Host:                     viper.GetString(config.MetricsListeningIp),
		Port:                     viper.GetInt(config.MetricsListeningPort),
		TeltonikaMetricsFileName: viper.GetString(config.MetricsTeltonikaMetricsFileName),
	}

	cfg := config.NewConfig(log, influxConfig, teltonikaConfig, metricsConfig)
	return cfg
}

func main() {
	cfg := parseConfig()

	log := cfg.GetLogger()
	log.Tracef("Used InfluxDB client configuration: %+v", cfg.GetInfluxConfig())
	log.Tracef("Used Teltonika server configuration: %+v", cfg.GetTeltonikaConfig())
	log.Tracef("Used metrics configuration: %+v", cfg.GetMetricsConfig())

	// Initialize context
	ctxSignals, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx := context.WithValue(context.Background(), config.ContextConfigKey, cfg)

	// Initialize Influxdb connection
	influxdb := influxdb2.NewConnection(ctx, cfg.GetInfluxConfig())

	// Initialize messaging subsystem
	messenger := messaging.NewMessaging(ctx)
	// Subscribe function knows how to store data into influxdb
	messenger.Subscribe(func(data interface{}) error {
		message := data.(fmb920.TeltonikaMessage)
		// Insert new record into influxdb
		tags := map[string]string{
			influxdb2.SourceTag: message.SourceAddress,
		}
		err := influxdb.InsertMessage(message.Decoded, tags)
		if err != nil {
			return fmt.Errorf("failed to persist data. %v", err)
		}
		return nil
	})

	hostname, err := os.Hostname()
	if err != nil {
		log.Errorf("Failed to get hostname. %v", err)
	}

	// Initialize metrics collector
	metrics := mi.NewMetrics(ctx, cfg.GetMetricsConfig().TeltonikaMetricsFileName)
	defer func() {
		err := metrics.Close()
		if err != nil {
			log.Errorf("Failed to close metrics. %v", err)
		}
	}()

	tags := []string{
		fmt.Sprintf("host=%s", hostname),
	}

	metricsServer := m.NewServer(ctx, cfg.GetMetricsConfig(), tags, []m.MetricProvider{
		metrics,
	})

	go metricsServer.Start()

	// Connect to influxdb server
	err = influxdb.Connect()
	if err != nil {
		log.Fatalf("Failed to open influxdb connection. %v", err)
		os.Exit(1)
	}

	// Always close influxdb connection
	defer func() {
		err := influxdb.Close()
		if err != nil {
			log.Errorf("Failed to close influxdb connection. %v", err)
		}
	}()

	server := fmb920.NewServer(ctx, cfg.GetTeltonikaConfig().Host, cfg.GetTeltonikaConfig().Port, cfg.GetTeltonikaConfig().AllowedIMEIs, metrics, func(ctx context.Context, message fmb920.TeltonikaMessage) {
		log := cfg.GetLogger()

		log.Debugf("PACKET ARRIVED: %+v", message)

		// Forward data internally for further processing
		messenger.Publish(message)
	})

	// Start Teltonika server
	err = server.Start()
	if err != nil {
		log.Errorf("Failed to start Teltonika server. %v", err)
	}

	<-ctxSignals.Done()
	log.Infof("Exiting")
}
