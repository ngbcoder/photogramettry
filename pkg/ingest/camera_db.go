package ingest

import "strings"

// sensorWidths maps camera model substrings to sensor width in mm.
var sensorWidths = map[string]float64{
	"iphone":  4.8,
	"canon":   36.0,
	"nikon":   35.9,
	"sony":    35.6,
	"gopro":   6.17,
	"pixel":   5.6,
	"galaxy":  7.0,
	"default": 36.0,
}

// SensorWidth returns an approximate sensor width in mm for the given make/model.
func SensorWidth(makeStr, model string) float64 {
	low := strings.ToLower(model + " " + makeStr)
	for k, v := range sensorWidths {
		if k != "default" && strings.Contains(low, k) {
			return v
		}
	}
	return sensorWidths["default"]
}
