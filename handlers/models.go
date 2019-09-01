package handlers

import "time"

const PRECISION = 10000000.0

type Location struct {
	Type        string
	Coordinates []float64
}

type Record struct {
	Imei      string                 `json:"imei,omitempty"`
	Location  Location               `json:"location,omitempty"`
	Time      time.Time              `json:"time,omitempty"`
	Angle     int16                  `json:"angle,omitempty"`
	Speed     int16                  `json:"speed,omitempty"`
	Altitude  int16                  `json:"altitude,omitempty"`
	Satellite int8                   `json:"satellite,omitempty"`
	EventID   uint8                  `json:"event_id,omitempty"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
}
