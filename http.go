package healthcheck

import "golang.org/x/net/context"

// Key represents the context value for the health module.
const Key = "health"

// FromContext returns the Monitor instance from net.Context.
func FromContext(c context.Context) *Monitor {
	return c.Value(Key).(*Monitor)
}

// HealthStatusResponse is returned by the health status endpoint.
type HealthStatusResponse struct {
	Status map[string]string `json:"status"`
}

// MarshalHealthStatusResponse converts a health.Status model to a
// HealthStatusResponse object.
func MarshalHealthStatusResponse(status Status) HealthStatusResponse {
	r := HealthStatusResponse{Status: make(map[string]string, len(status.Statuses))}
	for p, s := range status.Statuses {
		var v string
		if s == nil {
			v = "OK"
		} else {
			v = s.Error()
		}
		r.Status[p] = v
	}
	return r
}
