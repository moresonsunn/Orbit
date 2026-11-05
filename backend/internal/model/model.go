// Package model contains the JSON data transfer objects shared between collectors,
// the REST API and the WebSocket hub.
//
// Every optional metric carries an explicit availability status so that the UI can
// distinguish "0 %" from "not available", "no permission", "unsupported", "source
// unreachable" and "no samples yet".
package model

type Availability string

const (
	OK           Availability = "ok"
	Unavailable  Availability = "unavailable"
	NoPermission Availability = "no_permission"
	Unsupported  Availability = "unsupported"
	SourceDown   Availability = "source_down"
	Pending      Availability = "pending"
	Disabled     Availability = "disabled"
)

// Gauge is an optional numeric value with an availability status.
type Gauge struct {
	Value  *float64     `json:"value"`
	Status Availability `json:"status"`
	Reason string       `json:"reason,omitempty"`
}

func Val(v float64) Gauge { return Gauge{Value: &v, Status: OK} }

func NA(status Availability, reason string) Gauge { return Gauge{Status: status, Reason: reason} }

// Section describes the availability of a whole block of metrics.
type Section struct {
	Status Availability `json:"status"`
	Reason string       `json:"reason,omitempty"`
}

func SectionOK() Section { return Section{Status: OK} }

