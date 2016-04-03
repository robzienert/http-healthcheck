# http-healthcheck

A library for standardizing component healthcheck monitoring and displaying
the results into a standard HTTP endpoint for web services.

The library consists of a pluggable health provider interface and a background
supervisor that will run the healthchecks behind the scenes. The latest state
can then be queried by hitting an HTTP endpoint without cascading healthchecks
to an application's dependencies.

The DefaultSupervisor will continually recover and relaunch a Provider whenever
a panic occurs, with a flat 30-second backoff.

```go
// Package main includes an example of implementing with a gin router.
package main

import (
  "github.com/robzienert/http-healthcheck"
  "github.com/robzienert/http-healthcheck/monitor/cassandra"
  "github.com/gin-gonic/gin"
)

func main() {
  r := gin.Default()
  
  var healthProviders []healthcheck.Provider
  
  healthProviders = []healthcheck.Provider{
    cassandra.NewHealthProvider(gocqlSession),
  }
  
  healthMonitor := healthcheck.New(healthcheck.DefaultSupervisor, healthProviders...)
  {
    defer healthMonitor.Close()
    healthMonitor.Start()
  }
  
  r.Get("/status", GetHealthStatus)
  r.Run()
}

func GetHealthStatus(c *gin.Context) {
  status := healthcheck.FromContext(c).Status()
  resp := healthcheck.MarshalHealthStatusResponse(status)
  if status.Healthy {
    c.IndentedJSON(http.StatusOK, resp)
  } else {
    c.IndentedJSON(http.StatusInternalServerError, resp)
  }
}
```

## Default monitors

http-healthcheck comes with some monitors out of the box:

* `cassandra` will check that a gocql Session is currently capable of communicating
  with its Cassandra ring.
