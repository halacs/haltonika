package influxdb

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/filipkroca/teltonikaparser"
	"github.com/halacs/haltonika/config"
	_ "github.com/influxdata/influxdb1-client" // this is important because of the bug in go mod
	client "github.com/influxdata/influxdb1-client/v2"
	"time"
)

type Connection struct {
	ctx                context.Context
	url                string
	username           string
	password           string
	insecureSkipVerify bool
	measurement        string
	database           string

	client client.Client
}

func NewConnection(ctx context.Context, cfg *config.InfluxConfig) *Connection {
	return &Connection{
		ctx:                ctx,
		url:                cfg.Url,
		username:           cfg.Username,
		password:           cfg.Password,
		insecureSkipVerify: false,
		measurement:        cfg.Measurement,
		database:           cfg.Database,
	}
}

func (c *Connection) getInsecureSkipVerify() bool {
	return c.insecureSkipVerify
}

func (c *Connection) setInsecureSkipVerify(insecureSkipVerify bool) {
	c.insecureSkipVerify = insecureSkipVerify
}

func (c *Connection) Connect() error {
	var err error

	c.client, err = client.NewHTTPClient(client.HTTPConfig{
		Addr:               c.url,
		Username:           c.username,
		Password:           c.password,
		InsecureSkipVerify: c.insecureSkipVerify,
	})

	if err != nil {
		return fmt.Errorf("error creating InfluxDB Client. %v", err)
	}

	return nil
}

func (c *Connection) Close() error {
	err := c.client.Close()
	if err != nil {
		return fmt.Errorf("failed to close influxdb connection. %v", err)
	}
	return nil
}

func (c *Connection) renderTags(record teltonikaparser.Decoded) map[string]string {
	return map[string]string{
		"IMEI":    record.IMEI,
		"CodecID": fmt.Sprintf("%X", record.CodecID), // convert byte to hex number string
	}
}

func (c *Connection) renderFields(avlData teltonikaparser.AvlData) map[string]interface{} {
	log := config.GetLogger(c.ctx)

	fields := map[string]interface{}{
		"latitude":          float64(avlData.Lat) / 10000000.0,
		"longitude":         float64(avlData.Lng) / 10000000.0,
		"altitude":          avlData.Altitude,
		"visiblesatellites": avlData.VisSat,
		"angle":             avlData.Angle,
		"speed":             avlData.Speed,
		"priority":          avlData.Priority,
		"eventID":           avlData.EventID,
		"serverTime":        time.Now().UTC().Unix(),
		//"originalTime":      int64(avlData.Utime), //c.renderTimesamp(avlData),
	}

	for _, element := range avlData.Elements {
		IOID := element.IOID
		if element.Length > 8 {
			log.Warnf("IOID%d value long! Got %d bytes long IOID value! It will be save as hex string. Value: %s", IOID, element.Length, hex.EncodeToString(element.Value[:element.Length]))
			fields[fmt.Sprintf("IOID%d", IOID)] = hex.EncodeToString(element.Value)
		} else {
			// raw must have 8 bytes to parse it as uint64 so add zero bytes with BigEndian if needed
			addon := make([]byte, 8-element.Length)
			raw := append(addon, element.Value...)

			// parse byte array as integer
			value := binary.BigEndian.Uint64(raw)
			fields[fmt.Sprintf("IOID%d", IOID)] = int(value) // TODO if value is not casted to int, influx drops parse error. Why?
		}
	}

	return fields
}

func (c *Connection) renderTimesamp(avlData teltonikaparser.AvlData) time.Time {
	return time.UnixMilli(int64(avlData.UtimeMs))
}

func (c *Connection) insert(extraTags map[string]string, record teltonikaparser.Decoded) error {
	log := config.GetLogger(c.ctx)

	tags := c.renderTags(record)
	for k, v := range extraTags {
		_, ok := tags[k]
		if ok {
			log.Warningf("'%s' key already exist in record related tags. Ignore it in extra tags list.", k)
			continue
		}

		tags[k] = v
	}

	batchPointsConfig := client.BatchPointsConfig{
		Database: c.database,
	}

	bps, err := client.NewBatchPoints(batchPointsConfig)
	if err != nil {
		return fmt.Errorf("failed to create new batch point config. %v", err)
	}

	log.Debugf("Processing %d AVL data record.", len(record.Data))
	for _, data := range record.Data {
		fields := c.renderFields(data)
		timestamp := c.renderTimesamp(data)

		point, err := client.NewPoint(c.measurement, tags, fields, timestamp)
		//point, err := client.NewPoint(c.measurement, tags, fields)
		if err != nil {
			return fmt.Errorf("failed to create new point. %v", err)
		}
		bps.AddPoint(point)
	}
	log.Debugf("%d InfluxDB points are created.", len(bps.Points()))

	if c.client == nil {
		return fmt.Errorf("influxDB client must not be nil. Please check your influxdb connection")
	}

	err = c.client.Write(bps)
	if err != nil {
		return fmt.Errorf("failed to create write batch points into influxdb. %v", err)
	}

	return nil

}

func (c *Connection) InsertMessage(record teltonikaparser.Decoded, extraTags map[string]string) error {
	err := c.insert(extraTags, record)
	if err != nil {
		return fmt.Errorf("influxdb insert was failed. %v", err)
	}

	return nil
}
