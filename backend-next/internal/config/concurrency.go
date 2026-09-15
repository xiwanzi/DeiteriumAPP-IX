package config

import "errors"

// Zero values keep existing configuration files compatible. Changes take effect
// on service restart; these are capacity budgets, not user or throughput limits.
type Concurrency struct {
	HTTPRequests        int `json:"httpRequests,omitempty"`
	RealtimeConnections int `json:"realtimeConnections,omitempty"`
	AIStreams           int `json:"aiStreams,omitempty"`
	ControlRequests     int `json:"controlRequests,omitempty"`
}

func (c Concurrency) WithDefaults() Concurrency {
	if c.HTTPRequests == 0 {
		c.HTTPRequests = 256
	}
	if c.RealtimeConnections == 0 {
		c.RealtimeConnections = 1024
	}
	if c.AIStreams == 0 {
		c.AIStreams = 64
	}
	if c.ControlRequests == 0 {
		c.ControlRequests = 64
	}
	return c
}

func (c Concurrency) Validate() error {
	for _, limit := range []int{c.HTTPRequests, c.RealtimeConnections, c.AIStreams, c.ControlRequests} {
		if limit < 0 || limit > 16384 {
			return errors.New("concurrency limits must be between 0 and 16384")
		}
	}
	return nil
}
