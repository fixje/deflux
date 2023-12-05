package sensor

// ZHAThermostat represents the state of a thermostat
type ZHAThermostat struct {
	State
	Temperature int
	Valve int
}

// Fields implements the fielder interface and returns time series data for InfluxDB
func (z *ZHAThermostat) Fields() map[string]interface{} {
	return mergeFields(z.State.Fields(),
		map[string]interface{}{
			"temperature": float64(z.Temperature) / 100,
			"valve": z.Valve,
		})
}
