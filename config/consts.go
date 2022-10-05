package config

type MyKey struct {
	KeyName string
}

var (
	ContextConfigKey = MyKey{
		KeyName: "config",
	}
)

const (
	AppName                                = "haltonika"
	ViperEnvPrefix                         = AppName
	Verbose                                = "verbose"
	Debug                                  = "debug"
	AllowedIMEIs                           = "imeilist"
	InfluxConfigUrl                        = "url"
	InfluxConfigUsername                   = "username"
	InfluxConfigPassword                   = "password"
	InfluxConfigDatabase                   = "database"
	InfluxConfigMeasurement                = "measurement"
	TeltonikaListeningIp                   = "listenip"
	TeltonikaListeningPort                 = "listenport"
	MetricsListeningIp                     = "metricsip"
	MetricsListeningPort                   = "metricsport"
	MetricsTeltonikaMetricsFileName        = "mp"
	DefaultDebug                           = false
	DefaultVerbose                         = false
	DefaultInfluxDbUrl                     = "http://localhost:8086"
	DefaultInfluxDbDatabaseName            = AppName
	DefaultInfluxDbMeasurementName         = "gps"
	DefaultInfluxDbUserName                = AppName
	DefaultInfluxDbPassword                = "123"
	DefaultAllowedIMEIs                    = "350424063817363" // list, separated by comma
	DefaultTeltonikaListeningIP            = "0.0.0.0"
	DefaultTeltonikaListeningPort          = 9160
	DefaultMetricsListeningIP              = "0.0.0.0"
	DefaultMetricsListeningPort            = 9161
	DefaultMetricsTeltonikaMetricsFileName = AppName + ".met"
)
