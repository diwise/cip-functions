package events

import "time"

type FunctionUpdated struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	SubType   string    `json:"subtype"`
	Location  *Location `json:"location,omitempty"`
	Tenant    string    `json:"tenant,omitempty"`
	Source    string    `json:"source,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`

	/*Counter      counters.Counter            `json:"counter,omitempty"`
	Level        levels.Level                `json:"level,omitempty"`
	Presence     presences.Presence          `json:"presence,omitempty"`
	Timer        timers.Timer                `json:"timer,omitempty"`
	WaterQuality waterqualities.WaterQuality `json:"waterquality,omitempty"`
	Building     buildings.Building          `json:"building,omitempty"`
	AirQuality   airquality.AirQuality       `json:"AirQuality,omitempty"`*/

	State struct {
		Timestamp string `json:"timestamp"`
		State_    bool   `json:"state"`
	} `json:"state,omitempty"`

	Stopwatch struct {
		Count          int32          `json:"count"`
		CumulativeTime time.Duration  `json:"cumulativeTime"`
		Duration       *time.Duration `json:"duration,omitempty"`
		StartTime      time.Time      `json:"startTime"`
		State          bool           `json:"state"`
		StopTime       *time.Time     `json:"stopTime,omitempty"`
	} `json:"stopwatch,omitempty"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
