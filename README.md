# gatus

![build](https://github.com/TwinProduction/gatus/workflows/build/badge.svg?branch=master)
[![Docker pulls](https://img.shields.io/docker/pulls/twinproduction/gatus.svg)](https://cloud.docker.com/repository/docker/twinproduction/gatus)

A service health dashboard in Go that is meant to be used as a docker 
image with a custom configuration file.

Live example: https://status.twinnation.org/


## Usage

By default, the configuration file is expected to be at `config/config.yaml`.

You can specify a custom path by setting the `GATUS_CONFIG_FILE` environment variable.

```yaml
metrics: true         # Whether to expose metrics at /metrics
services:
  - name: twinnation  # Name of your service, can be anything
    url: https://twinnation.org/health
    interval: 15s     # Duration to wait between every status check (default: 10s)
    conditions:
      - "[STATUS] == 200"         # Status must be 200
      - "[RESPONSE_TIME] < 300"   # Response time must be under 300ms
  - name: github
    url: https://api.github.com/healthz
    interval: 2m
    conditions:
      - "[STATUS] == 200"
  - name: Example
    url: https://example.org/
    interval: 30s
    conditions:
      - "[STATUS] == 200"
```

Note that you can also add environment variables in the your configuration file (i.e. `$DOMAIN`, `${DOMAIN}`)


### Conditions

Here are some examples of conditions you can use:

| Condition                             | Description                               | Passing values           | Failing values          |
| ------------------------------------- | ----------------------------------------- | ------------------------ | ----------------------- |
| `[STATUS] == 200`                     | Status must be equal to 200               | 200                      | 201, 404, 500           |
| `[STATUS] < 300`                      | Status must lower than 300                | 200, 201, 299            | 301, 302, 400, 500      |
| `[STATUS] <= 299`                     | Status must be less than or equal to 299  | 200, 201, 299            | 301, 302, 400, 500      |
| `[STATUS] > 400`                      | Status must be greater than 400           | 401, 402, 403, 404       | 200, 201, 300, 400      |
| `[RESPONSE_TIME] < 500`               | Response time must be below 500ms         | 100ms, 200ms, 300ms      | 500ms, 1500ms           |
| `[BODY] == 1`                         | The body must be equal to 1               | 1                        | literally anything else |
| `[BODY].data.id == 1`                 | The jsonpath `$.data.id` is equal to 1    | `{"data":{"id":1}}`      | literally anything else |

**NOTE**: `[BODY]` with JSON path (i.e. `[BODY].id == 1`) is currently in BETA. For the most part, the only thing that doesn't work is arrays.


## Docker

Building the Docker image is done as following:

```
docker build . -t gatus
```

You can then run the container with the following command:

```
docker run -p 8080:8080 --name gatus gatus
```


## Running the tests

```
go test ./... -mod vendor
```


## Using in Production

See the [example](example) folder.
