package sensor

// ZHALightLevel represents the state of a light level sensor
type ZHALightLevel struct {
	State
	Dark       bool
	Daylight   bool
	LightLevel int32
	Lux        int32
}

// Fields implements the fielder interface and returns time series data for InfluxDB
func (z *ZHALightLevel) Fields() map[string]interface{} {
	return mergeFields(z.State.Fields(),
		map[string]interface{}{
			"daylight":   z.Daylight,
			"dark":       z.Dark,
			"lightlevel": z.LightLevel,
			"lux":        z.Lux,
		})
}
