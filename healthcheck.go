// Package healthcheck provides a standard web application healthcheck interface
// for its dependencies.
package healthcheck

import (
	"io"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
)

// Provider defines the interface for other modules to supply health metrics to
// the health module for determining healthy state.
//
// Provider implements Closer, which will be called when the module shutsdown.
// It is expected that this function be able to cascade any cancellation
// operations to underlying goroutines implemented by the Provider.
//
// If the Provider panics, the health module will attempt to restart the
// provider by calling Start with a continual backoff. A provider crash will
// assume that the underlying system is unhealthy.
//
// Name should return a human friendly name of the provider. This will be used
// as a provider key in the healthcheck's HTTP endpoint.
//
// Start should initiate the healthcheck process. It is expected that any
// implementing Provider be able to continue running on its own without being
// called again by the health module. This function must not block.
//
// IsHealthy should return nil on a healthy state, and a descriptive error if
// the underlying system is unhealthy. This function should not perform any
// external system calls on invocation.
type Provider interface {
	io.Closer

	Name() string
	Start() error
	IsHealthy() error
}

// ProviderStatuses is a map of provider names and their health status as an
// error. If the provider is reporting a healthy status, the error will be nil.
type ProviderStatuses map[string]error

// Status is the aggregate status of all application health providers.
type Status struct {
	Healthy  bool
	Statuses ProviderStatuses
}

// Supervisor can be implemented to watch and recover health providers.
type Supervisor func(provider Provider) chan struct{}

// DefaultSupervisor will continually recover a health provider if it panics,
// with a backoff.
func DefaultSupervisor(provider Provider) chan struct{} {
	done := make(chan struct{}, 1)
	go func() {
		var wg sync.WaitGroup

		wg.Add(1)
		provider.Start()

		defer func() {
			for {
				select {
				case <-done:
					provider.Close()
					wg.Done()
					return
				default:
					if r := recover(); r != nil {
						backoff := 30 * time.Second
						logrus.WithFields(logrus.Fields{
							"name": provider.Name(),
							"err":  r,
						}).Errorf("Provider panic! Recovering in %d seconds...", backoff)

						time.Sleep(backoff)
						logrus.WithField("name", provider.Name()).Warn("Restarting provider after panic")
						provider.Start()
					}
				}
			}
		}()
		wg.Wait()
	}()
	return done
}

// Monitor manages aggregation of all health providers statuses and maintaining
// the health supervisors.
type Monitor struct {
	providers     []Provider
	supervisor    Supervisor
	supervisorChs []chan struct{}
}

// New creates a Monitor instance.
func New(supervisor Supervisor, providers ...Provider) *Monitor {
	if supervisor == nil {
		supervisor = DefaultSupervisor
	}
	return &Monitor{supervisor: supervisor, providers: providers}
}

// Start monitoring health for the application.
func (m *Monitor) Start() {
	var names []string
	logrus.Info("Starting health provider manager")

	for _, p := range m.providers {
		ch := m.supervisor(p)
		m.supervisorChs = append(m.supervisorChs, ch)
		names = append(names, p.Name())
	}
	logrus.WithField("providers", names).Debug("Started health providers")
}

// Close will shutdown the monitor and all providers.
func (m *Monitor) Close() {
	logrus.Warn("Shutting down HealthMonitor")
	for _, ch := range m.supervisorChs {
		ch <- struct{}{}
	}
}

// Status aggregates the latest health statuses from the providers.
func (m *Monitor) Status() Status {
	all := Status{Healthy: true, Statuses: make(ProviderStatuses, 0)}
	for _, p := range m.providers {
		status := p.IsHealthy()
		if status != nil {
			all.Healthy = false
		}
		all.Statuses[p.Name()] = status
	}
	return all
}
