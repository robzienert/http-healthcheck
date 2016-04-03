// Package cassandra provides a default health monitor for gocql Sessions.
package cassandra

import (
	"errors"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gocql/gocql"
)

// HealthProvider is an implementation of the health.Provider interface for
// Cassandra.
type HealthProvider struct {
	session *gocql.Session
	healthy error
	done    chan struct{}
	lock    sync.RWMutex
}

// NewHealthProvider returns a new Cassandra health.Provider instance.
func NewHealthProvider(session *gocql.Session) *HealthProvider {
	return &HealthProvider{session: session, healthy: errors.New("health status unchecked")}
}

// Name of the HealthProvider
func (p *HealthProvider) Name() string {
	return "cassandra"
}

// Start initiates the provider's internal checker.
func (p *HealthProvider) Start() error {
	logrus.Debug("Starting Cassandra health provider")
	p.done = make(chan struct{}, 1)

	go func() {
		ticker := time.NewTicker(60 * time.Second)
		p.performCheck()
		for {
			select {
			case <-ticker.C:
				p.performCheck()
			case <-p.done:
				ticker.Stop()
				logrus.WithField("healthProvider", p.Name()).Warn("Received close signal, shutting down")
				return
			}
		}
	}()

	return nil
}

func (p *HealthProvider) performCheck() {
	logrus.Debug("Checking Cassandra health")
	p.lock.Lock()
	defer p.lock.Unlock()
	if err := p.session.Query("select now() from system.local").Exec(); err != nil {
		p.healthy = err
		return
	}
	p.healthy = nil
}

// IsHealthy returns the current health state of the provider.
func (p *HealthProvider) IsHealthy() error {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.healthy
}

// Close NO-OP.
func (p *HealthProvider) Close() error {
	p.done <- struct{}{}
	return nil
}
