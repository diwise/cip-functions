package events

type Stopwatch struct {
	StartTime string `json:"startTime"`
	StopTime  string `json:"stopTime"`
	State     bool   `json:"state"`
}
