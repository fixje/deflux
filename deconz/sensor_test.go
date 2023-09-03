//go:build integration_test
// +build integration_test

package deconz

import (
	"fmt"
	"github.com/fixje/deflux/deconz/sensor"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"log/slog"
	"testing"
	"time"
)

// FIXME config via environment variables
func TestSensorEvent(t *testing.T) {

	s := &sensor.Sensor{
		Type: "type",
		Name: "name",
	}

	event := &WsEvent{
		Type:         "event",
		Event:        "changed",
		ResourceName: "sensor",
		ID:           123,
		RawState:     nil,
		StateDef:     nil,
	}

	se := SensorEvent{
		Sensor: s,
		Event:  event,
	}

	slog.Info("sensor event",
		"id", se.ResourceID(), "res", se.Resource(), "state", se.State(), "sensor", se.Sensor)
}

func TestWriteInfluxDB(t *testing.T) {
	//
	influxClient := influxdb2.NewClientWithOptions(
		"http://localhost:8086",
		"fixje:asdf1234",
		influxdb2.DefaultOptions().SetBatchSize(20))

	// Get non-blocking write client
	writeAPI := influxClient.WriteAPI("", "sensors")
	// Get errors channel
	errorsCh := writeAPI.Errors()

	// read and log errors in a separate go routine
	go func() {
		for err := range errorsCh {
			fmt.Printf("write error: %s\n", err.Error())
		}
	}()

	sensorEvent := SensorEvent{
		Sensor: &sensor.Sensor{
			Type: "ZHAAirQuality",
			Name: "test",
		},
		Event: WsEvent{
			Type:         "event",
			Event:        "changed",
			ResourceName: "sensor",
			ID:           15,
			StateDef: &sensor.ZHAAirQuality{
				State:         sensor.State{Lastupdated: ""},
				Airquality:    "good",
				AirqualityPPB: 77,
			},
			RawState: nil,
		},
	}

	tags, fields, err := sensorEvent.Timeseries()
	if err != nil {
		slog.Warn("not adding event to influx: %s", err)
	}

	writeAPI.WritePoint(influxdb2.NewPoint(
		fmt.Sprintf("deflux_%s", sensorEvent.Sensor.Type),
		tags,
		fields,
		time.Now(), // TODO: we should use the time associated with the event...
	))

	writeAPI.Flush()
	influxClient.Close()
}
