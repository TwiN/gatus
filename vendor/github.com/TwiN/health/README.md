# health
![build](https://github.com/TwiN/health/workflows/build/badge.svg?branch=master) 
[![Go Report Card](https://goreportcard.com/badge/github.com/TwiN/health)](https://goreportcard.com/report/github.com/TwiN/health)
[![codecov](https://codecov.io/gh/TwiN/health/branch/master/graph/badge.svg)](https://codecov.io/gh/TwiN/health)
[![Go version](https://img.shields.io/github/go-mod/go-version/TwiN/health.svg)](https://github.com/TwiN/health)
[![Go Reference](https://pkg.go.dev/badge/github.com/TwiN/health.svg)](https://pkg.go.dev/github.com/TwiN/health)

Health is a library used for creating a very simple health endpoint.

While implementing a health endpoint is very simple, I've grown tired of implementing 
it over and over again.


## Installation
```
go get -u github.com/TwiN/health
```


## Usage
To retrieve the handler, you must use `health.Handler()` and are expected to pass it to the router like so:
```go
router := http.NewServeMux()
router.Handle("/health", health.Handler())
server := &http.Server{
    Addr:    ":8080",
    Handler: router,
}
```

By default, the handler will return `UP` when the status is down, and `DOWN` when the status is down.
If you prefer using JSON, however, you may initialize the health handler like so:
```go
router.Handle("/health", health.Handler().WithJSON(true))
```
The above will cause the response body to become `{"status":"UP"}` and `{"status":"DOWN"}` for both status respectively.

To change the health of the application, you can use `health.SetStatus(<status>)` where `<status>` is one `health.Up`
or `health.Down`:

```go
health.SetStatus(health.Up)
health.SetStatus(health.Down)
```


### Complete example
```go
package main

import (
    "net/http"
    "time"

    "github.com/TwiN/health"
)

func main() {
    router := http.NewServeMux()
    router.Handle("/health", health.Handler())
    server := &http.Server{
        Addr:         "0.0.0.0:8080",
        Handler:      router,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  15 * time.Second,
    }
    server.ListenAndServe()
}
```
