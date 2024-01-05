package sensor

// ZHAVibration represents the state of a vibration sensor
// TODO not sure if int or float: orientation, tiltangle, vibrationstrength
type ZHAVibration struct {
	State
	Vibration bool
	Tiltangle int16
	Vibrationstrength int16
	Orientation []int16
}

// Fields implements the fielder interface and returns time series data for InfluxDB
func (z *ZHAVibration) Fields() map[string]interface{} {
	return mergeFields(z.State.Fields(),
		map[string]interface{}{
			"vibration": z.Vibration,
			"tiltangle": z.Tiltangle,
			"vibrationstrength": z.Vibrationstrength,
			"orientation_x": z.Orientation[0],
			"orientation_y": z.Orientation[1],
			"orientation_z": z.Orientation[2],
		})
}
