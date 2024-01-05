package sensor

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

// Sensors is a map of sensors indexed by their id
type Sensors map[int]Sensor

// Provider provides information about sensors
type Provider interface {
	// Sensors provides info about all known sensors
	Sensors() (*Sensors, error)

	// Sensor gets a sensor by id
	Sensor(int) (*Sensor, error)
}

// Fielder is an interface that provides fields for InfluxDB
type Fielder interface {
	Fields() map[string]interface{}
}

// TimeSeries provides tags and fields for the time series database
type TimeSeries interface {
	Timeseries() (map[string]string, map[string]interface{}, error)
}

// Sensor is a deCONZ sensor
// We only implement required fields for event decoding
type Sensor struct {
	Type     string    `json:"type"`
	Name     string    `json:"name"`
	LastSeen time.Time `json:"lastseen"`
	StateDef interface{}
	Config   Config
	ID       int
}

// Config represents the sensor configuration as retrieved from the API
// Currently, it holds only the battery state.
type Config struct {
	// Battery state in percent; not present for all sensors
	Battery			uint32	`json:"battery"`
	HeatSetpoint		*int	`json:"heatsetpoint"`
	Mode			*string	`json:"mode"`
	Offset			*int	`json:"offset"`
	ExternalSensorTemp	*int	`json:"externalsensortemp"`
}

// State contains properties that are provided by all sensors
// It is embedded in specific sensors' State
type State struct {
	Lastupdated string
}

// EmptyState is an empty struct used to indicate no state was parsed
type EmptyState struct{}

// UnmarshalJSON converts a JSON representation of a Sensor into the Sensor struct
// The auxiliary approach is inspired by https://github.com/golang/go/issues/21990
func (s *Sensor) UnmarshalJSON(b []byte) error {
	var aux struct {
		Type     string          `json:"type"`
		Name     string          `json:"name"`
		LastSeen string          `json:"lastseen"`
		State    json.RawMessage `json:"state"`
		Config   Config
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return err
	}

	if aux.LastSeen != "" {
		t, err := time.Parse("2006-01-02T15:04Z", aux.LastSeen)
		if err != nil {
			return err
		}

		s.LastSeen = t
	}

	s.Type = aux.Type
	s.Name = aux.Name
	s.Config = aux.Config

	state, err := DecodeSensorState(aux.State, aux.Type)
	if err == nil {
		s.StateDef = state
	} else {
		slog.Warn("unable to decode state: %s", err)
		s.StateDef = EmptyState{}
	}

	return nil
}

// Timeseries returns tags and fields for use in InfluxDB
func (s *Sensor) Timeseries() (map[string]string, map[string]interface{}, error) {
	f, ok := s.StateDef.(Fielder)
	if !ok {
		return nil, nil, fmt.Errorf("this sensor (%T:%s) has no time series data", s.StateDef, s.Name)
	}

	fields := f.Fields()

	if _, ok := fields["battery"]; !ok {
		fields["battery"] = int(s.Config.Battery)
	}

        // special cases
        switch s.Type {

                case "ZHAThermostat":
			fields["heatsetpoint"] = float64(*s.Config.HeatSetpoint) / 100
			fields["mode"] = *s.Config.Mode
			fields["offset"] = float64(*s.Config.Offset) / 100
			fields["externalsensortemp"] = float64(*s.Config.ExternalSensorTemp) / 100

        }

	return map[string]string{
			"name":   s.Name,
			"type":   s.Type,
			"id":     strconv.Itoa(s.ID),
			"source": "rest"},
		fields,
		nil
}

// DecodeSensorState tries to unmarshal the appropriate state based
// on the given sensor type
func DecodeSensorState(rawState json.RawMessage, sensorType string) (interface{}, error) {

	var err error

	switch sensorType {
	case "CLIPPresence":
		var s CLIPPresence
		err = json.Unmarshal(rawState, &s)
		return &s, err

	case "Daylight":
		var s Daylight
		err = json.Unmarshal(rawState, &s)
		return &s, err

	case "ZHAAirQuality":
		var s ZHAAirQuality
		err = json.Unmarshal(rawState, &s)
		return &s, err

	case "ZHABattery":
		var s ZHABattery
		err = json.Unmarshal(rawState, &s)
		return &s, err

	case "ZHACarbonMonoxide":
		var s ZHACarbonMonoxide
		err = json.Unmarshal(rawState, &s)
		return &s, err

	case "ZHAConsumption":
		var s ZHAConsumption
		err = json.Unmarshal(rawState, &s)
		return &s, err

	case "ZHAFire":
		var s ZHAFire
		err = json.Unmarshal(rawState, &s)
		return &s, err

	case "ZHAHumidity":
		var s ZHAHumidity
		err = json.Unmarshal(rawState, &s)
		return &s, err

	case "ZHALightLevel":
		var s ZHALightLevel
		err = json.Unmarshal(rawState, &s)
		return &s, err

	case "ZHAOpenClose":
		var s ZHAOpenClose
		err = json.Unmarshal(rawState, &s)
		return &s, err

	case "ZHAPower":
		var s ZHAPower
		err = json.Unmarshal(rawState, &s)
		return &s, err

	case "ZHAPresence":
		var s ZHAPresence
		err = json.Unmarshal(rawState, &s)
		return &s, err

	case "ZHAPressure":
		var s ZHAPressure
		err = json.Unmarshal(rawState, &s)
		return &s, err

	case "ZHASwitch":
		var s ZHASwitch
		err = json.Unmarshal(rawState, &s)
		return &s, err

	case "ZHATemperature":
		var s ZHATemperature
		err = json.Unmarshal(rawState, &s)
		return &s, err

	case "ZHAVibration":
		var s ZHAVibration
		err = json.Unmarshal(rawState, &s)
		return &s, err

	case "ZHAWater":
		var s ZHAWater
		err = json.Unmarshal(rawState, &s)
		return &s, err

	case "ZHAThermostat":
		var s ZHAThermostat
		err = json.Unmarshal(rawState, &s)
		return &s, err

	}

	return nil, fmt.Errorf("%s is not a known sensor type", sensorType)
}

// Fields returns the data age of the state (time.Now() - state.Lastupdated) in seconds
func (s *State) Fields() map[string]interface{} {
	if s.Lastupdated != "" {
		t, err := time.Parse("2006-01-02T15:04:05.999", s.Lastupdated)

		if err != nil {
			slog.Warn("Failed to unmarshal `lastupdated`: %s", err)
		} else {
			return map[string]interface{}{
				"age_secs": int64(time.Now().Sub(t).Seconds()),
			}
		}
	}

	return map[string]interface{}{}
}

// mergeFields returns a merged map that contains all entries from p and s.
// If both map contain a certain key, the value of p takes precedence.
func mergeFields(p, s map[string]interface{}) map[string]interface{} {
	for k, e := range s {
		if _, ok := p[k]; !ok {
			p[k] = e
		}
	}
	return p
}
