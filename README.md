# gatus

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
    url: https://twinnation.org/actuator/health
    interval: 15s     # Duration to wait between every status check (opt. default: 10s)
    conditions:
      - "[STATUS] == 200"
  - name: github
    url: https://api.github.com/healthz
    conditions:
      - "[STATUS] == 200"
```

Note that you can also add environment variables in the your configuration file (i.e. `$DOMAIN`, `${DOMAIN}`)


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
