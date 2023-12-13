package deconz

import (
	"encoding/json"
	"fmt"
	"github.com/fixje/deflux/pkg/deconz/sensor"
	"strconv"
)

// Event is a common interface for different types of websocket events
type Event interface {
	EventName() string
	Resource() string
	ResourceID() int
	State() interface{}
}

// Timeserieser returns time series data
type Timeserieser interface {
	Timeseries() (map[string]string, map[string]interface{}, error)
}

// SensorEvent is an Event triggered by a Sensor
type SensorEvent struct {
	*sensor.Sensor
	Event
}

// Timeseries returns tags and fields for use in InfluxDB
func (s *SensorEvent) Timeseries() (map[string]string, map[string]interface{}, error) {
	if s.Event == nil || s.Event.State() == nil {
		return nil, nil, fmt.Errorf("event is empty: %v", s)
	}

	f, ok := s.Event.State().(sensor.Fielder)
	if !ok {
		return nil, nil, fmt.Errorf("this event (%T:%s) has no time series data", s.State, s.Name)
	}

	fields := f.Fields()

	if _, ok := fields["battery"]; !ok {
		fields["battery"] = int(s.Sensor.Config.Battery)
	}

	// special cases
        switch s.Type {

                case "ZHAThermostat":
                        fields["heatsetpoint"] = float64(*s.Config.HeatSetpoint) / 100
                        fields["mode"] = *s.Config.Mode
                        fields["offset"] = float64(*s.Config.Offset) / 100


        }


	return map[string]string{
			"name":   s.Name,
			"type":   s.Sensor.Type,
			"id":     strconv.Itoa(s.Event.ResourceID()),
			"source": "websocket"},
		fields,
		nil
}

// WsEvent is a message received over the deCONZ websocket
// We are only interested in e = 'change' events of resource type r = 'sensor'.
// Thus we don't implement all fields.
// See https://dresden-elektronik.github.io/deconz-rest-doc/endpoints/websocket/#message-fields
type WsEvent struct {
	// type should always be 'event'
	Type string `json:"t"`

	Event        string `json:"e"`
	ResourceName string `json:"r"`
	ID           int    `json:"id,string"`

	// TODO intermediate type
	// only for e = 'changed'
	RawState json.RawMessage `json:"state"`

	// only for e = 'changed'
	StateDef interface{}
}

// EventName return the name of an event, e.g. "change"
func (e WsEvent) EventName() string {
	return e.Event
}

// Resource returns the resource type affected by the event, e.g. "sensor"
func (e WsEvent) Resource() string {
	return e.ResourceName
}

// ResourceID returns the unique id of the resource which was affected by the event
func (e WsEvent) ResourceID() int {
	return e.ID
}

// State returns the current state of the resource
func (e WsEvent) State() interface{} {
	return e.StateDef
}

// DecodeEvent parses events from bytes
func DecodeEvent(sp sensor.Provider, b []byte) (Event, error) {
	var e WsEvent
	err := json.Unmarshal(b, &e)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal json: %s", err)
	}

	// We don't decode anything other than sensor events
	// If there is no state, dont try to parse it
	if e.Resource() != "sensors" || len(e.RawState) == 0 {
		e.StateDef = &sensor.EmptyState{}
		return e, nil
	}

	s, err := sp.Sensor(e.ID)
	if err != nil {
		return nil, fmt.Errorf("unable to get sensor with id %d: %s", e.ID, err)
	}

	state, err := sensor.DecodeSensorState(e.RawState, s.Type)
	if err != nil {
		return nil, fmt.Errorf("unable to decode state: %s", err)
	}
	e.StateDef = state

	return SensorEvent{Sensor: s, Event: e}, nil
}
