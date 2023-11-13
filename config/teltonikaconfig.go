package config

type TeltonikaConfig struct {
	Host         string
	Port         int
	AllowedIMEIs []string
}

type MetricsConfig struct {
	Host                     string
	Port                     int
	TeltonikaMetricsFileName string
}

type UdsServerConfig struct {
	BasePath string
}
