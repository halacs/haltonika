package metrics

import (
	"context"
	"fmt"
	"github.com/halacs/haltonika/config"
	"github.com/sirupsen/logrus"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type MetricProvider interface {
	MetricRendererHandler() (string, map[string]uint64)
}

/*
Provides HTTP endpoint for http input plugin of Telegraf
https://github.com/influxdata/telegraf/tree/master/plugins/inputs/http
*/
type Server struct {
	ctx       context.Context
	wg        *sync.WaitGroup
	stopFunc  context.CancelFunc
	host      string
	port      int
	renderers []MetricProvider
	tags      []string
}

func NewServer(ctx context.Context, wg *sync.WaitGroup, cfg *config.MetricsConfig, tags []string, renderers []MetricProvider) *Server {
	return &Server{
		wg:        wg,
		host:      cfg.Host,
		port:      cfg.Port,
		ctx:       ctx,
		renderers: renderers,
		tags:      tags,
	}
}

func (s *Server) metricsHandler(w http.ResponseWriter, req *http.Request) {
	//log := config.GetLogger(s.ctx)
	//log.Tracef("Serving metrics request")	// generates too much log

	for _, renderer := range s.renderers {
		metricName, fieldsMap := renderer.MetricRendererHandler()

		/*
			Influx line protocol example:
			citibike,station_id=4703 eightd_has_available_keys=false,is_installed=1,is_renting=1,is_returning=1,legacy_id="4703",num_bikes_available=6,num_bikes_disabled=2,num_docks_available=26,num_docks_disabled=0,num_ebikes_available=0,station_status="active" 1641505084000000000

			See more: https://docs.influxdata.com/influxdb/v1.8/write_protocols/line_protocol_tutorial/
		*/

		// Convert map to fields part of influx line protocol (only for humans but ensure same key orders each time by sorting)
		keys := make([]string, 0)
		for k := range fieldsMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var fieldsArray []string
		for _, k := range keys {
			fieldsArray = append(fieldsArray, fmt.Sprintf("%s=%d", k, fieldsMap[k]))
		}

		tags := strings.Join(s.tags, ",")
		fields := strings.Join(fieldsArray, ",")
		timestamp := time.Now().UnixMilli() * 1000000
		rawMetrics := fmt.Sprintf("%s,%s %s %d\n", metricName, tags, fields, timestamp)

		// Send line to the HTTP client
		fmt.Fprint(w, rawMetrics)
	}
}

func (s *Server) Start() {
	log := config.GetLogger(s.ctx)

	url := fmt.Sprintf("%s:%d", s.host, s.port)

	log.Infof("Start metrics server on %s", url)

	http.HandleFunc("/metrics", s.metricsHandler)

	httpServer := &http.Server{
		Addr:              url,
		Handler:           nil,
		ReadHeaderTimeout: 5 * time.Second, // Potential Slowloris Attack if not set
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		err := httpServer.ListenAndServe()
		if err != nil {
			if err == http.ErrServerClosed {
				return
			}

			logrus.Errorf("Error in metric server. %v", err)
			return
		}
	}()

	<-s.ctx.Done()
	err := httpServer.Shutdown(context.Background())
	if err != nil {
		log.Errorf("Failed to stop http server. %v", err)
	}
}

func (s *Server) Stop() error {
	if s.stopFunc == nil {
		return fmt.Errorf("server is not running")
	}

	s.stopFunc()
	s.stopFunc = nil
	return nil
}
