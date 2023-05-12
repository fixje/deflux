package sensor

// ZHAOpenClose represents the state of an open/close sensor
type ZHAOpenClose struct {
	State
	Open       bool
	Lowbattery bool
	Tampered   bool
}

// Fields implements the fielder interface and returns time series data for InfluxDB
func (z *ZHAOpenClose) Fields() map[string]interface{} {
	return mergeFields(z.State.Fields(),
		map[string]interface{}{
			"lowbattery": z.Lowbattery,
			"tampered":   z.Tampered,
			"open":       z.Open,
			"open_int":   boolToInt(z.Open),
		})
}

func boolToInt(b bool) int {
	if b {
		return 1
	} else {
		return 0
	}
}
