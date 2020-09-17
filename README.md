![Gatus](static/logo-with-name.png)

![build](https://github.com/TwinProduction/gatus/workflows/build/badge.svg?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/TwinProduction/gatus)](https://goreportcard.com/report/github.com/TwinProduction/gatus)
[![Docker pulls](https://img.shields.io/docker/pulls/twinproduction/gatus.svg)](https://cloud.docker.com/repository/docker/twinproduction/gatus)

A service health dashboard in Go that is meant to be used as a docker 
image with a custom configuration file.

I personally deploy it in my Kubernetes cluster and have it monitor the status of my
core applications: https://status.twinnation.org/


## Table of Contents

- [Features](#features)
- [Usage](#usage)
  - [Configuration](#configuration)
  - [Conditions](#conditions)
  - [Alerting](#alerting)
    - [Configuring Slack alerts](#configuring-slack-alerts)
    - [Configuring PagerDuty alerts](#configuring-pagerduty-alerts)
    - [Configuring Twilio alerts](#configuring-twilio-alerts)
    - [Configuring custom alerts](#configuring-custom-alerts)
- [Docker](#docker)
- [Running the tests](#running-the-tests)
- [Using in Production](#using-in-production)
- [FAQ](#faq)
  - [Sending a GraphQL request](#sending-a-graphql-request)


## Features

The main features of Gatus are:
- **Highly flexible health check conditions**: While checking the response status may be enough for some use cases, Gatus goes much further and allows you to add conditions on the response time, the response body and even the IP address.
- **Ability to use Gatus for user acceptance tests**: Thanks to the point above, you can leverage this application to create automated user acceptance tests.
- **Very easy to configure**: Not only is the configuration designed to be as readable as possible, it's also extremely easy to add a new service or a new endpoint to monitor.
- **Alerting**: While having a pretty visual dashboard is useful to keep track of the state of your application(s), you probably don't want to stare at it all day. Thus, notifications via Slack are supported out of the box with the ability to configure a custom alerting provider for any needs you might have, whether it be a different provider like PagerDuty or a custom application that manages automated rollbacks. 
- **Metrics**
- **Low resource consumption**: As with most Go applications, the resource footprint that this application requires is negligibly small.


## Usage

By default, the configuration file is expected to be at `config/config.yaml`.

You can specify a custom path by setting the `GATUS_CONFIG_FILE` environment variable.

Here's a simple example:

```yaml
metrics: true         # Whether to expose metrics at /metrics
services:
  - name: twinnation  # Name of your service, can be anything
    url: "https://twinnation.org/health"
    interval: 30s     # Duration to wait between every status check (default: 60s)
    conditions:
      - "[STATUS] == 200"         # Status must be 200
      - "[BODY].status == UP"     # The json path "$.status" must be equal to UP
      - "[RESPONSE_TIME] < 300"   # Response time must be under 300ms
  - name: example
    url: "https://example.org/"
    interval: 30s
    conditions:
      - "[STATUS] == 200"
```

This example would look like this:

![Simple example](.github/assets/example.png)

Note that you can also add environment variables in the configuration file (i.e. `$DOMAIN`, `${DOMAIN}`)


### Configuration

| Parameter                                | Description                                                                   | Default        |
| ---------------------------------------- | ----------------------------------------------------------------------------- | -------------- |
| `debug`                                  | Whether to enable debug logs                                                  | `false`        |
| `metrics`                                | Whether to expose metrics at /metrics                                         | `false`        |
| `services`                               | List of services to monitor                                                   | Required `[]`  |
| `services[].name`                        | Name of the service. Can be anything.                                         | Required `""`  |
| `services[].url`                         | URL to send the request to                                                    | Required `""`  |
| `services[].conditions`                  | Conditions used to determine the health of the service                        | `[]`           |
| `services[].interval`                    | Duration to wait between every status check                                   | `60s`          |
| `services[].method`                      | Request method                                                                | `GET`          |
| `services[].graphql`                     | Whether to wrap the body in a query param (`{"query":"$body"}`)               | `false`        |
| `services[].body`                        | Request body                                                                  | `""`           |
| `services[].headers`                     | Request headers                                                               | `{}`           |
| `services[].alerts[].type`               | Type of alert. Valid types: `slack`, `pagerduty`, `twilio`, `custom`          | Required `""`  |
| `services[].alerts[].enabled`            | Whether to enable the alert                                                   | `false`        |
| `services[].alerts[].failure-threshold`  | Number of failures in a row needed before triggering the alert                | `3`            |
| `services[].alerts[].success-threshold`  | Number of successes in a row before an ongoing incident is marked as resolved | `2`            |
| `services[].alerts[].send-on-resolved`   | Whether to send a notification once a triggered alert is marked as resolved   | `false`        |
| `services[].alerts[].description`        | Description of the alert. Will be included in the alert sent                  | `""`           |
| `alerting`                               | Configuration for alerting                                                    | `{}`           |
| `alerting.slack`                         | Webhook to use for alerts of type `slack`                                     | `""`           |
| `alerting.pagerduty`                     | PagerDuty Events API v2 integration key. Used for alerts of type `pagerduty`  | `""`           |
| `alerting.twilio`                        | Settings for alerts of type `twilio`                                          | `""`           |
| `alerting.twilio.sid`                    | Twilio account SID                                                            | Required `""`  |
| `alerting.twilio.token`                  | Twilio auth token                                                             | Required `""`  |
| `alerting.twilio.from`                   | Number to send Twilio alerts from                                             | Required `""`  |
| `alerting.twilio.to`                     | Number to send twilio alerts to                                               | Required `""`  |
| `alerting.custom`                        | Configuration for custom actions on failure or alerts                         | `""`           |
| `alerting.custom.url`                    | Custom alerting request url                                                   | `""`           |
| `alerting.custom.body`                   | Custom alerting request body.                                                 | `""`           |
| `alerting.custom.headers`                | Custom alerting request headers                                               | `{}`           |


### Conditions

Here are some examples of conditions you can use:

| Condition                    | Description                                             | Passing values           | Failing values |
| -----------------------------| ------------------------------------------------------- | ------------------------ | -------------- |
| `[STATUS] == 200`            | Status must be equal to 200                             | 200                      | 201, 404, ...  |
| `[STATUS] < 300`             | Status must lower than 300                              | 200, 201, 299            | 301, 302, ...  |
| `[STATUS] <= 299`            | Status must be less than or equal to 299                | 200, 201, 299            | 301, 302, ...  |
| `[STATUS] > 400`             | Status must be greater than 400                         | 401, 402, 403, 404       | 400, 200, ...  |
| `[RESPONSE_TIME] < 500`      | Response time must be below 500ms                       | 100ms, 200ms, 300ms      | 500ms, 501ms   |
| `[BODY] == 1`                | The body must be equal to 1                             | 1                        | Anything else  |
| `[BODY].data.id == 1`        | The jsonpath `$.data.id` is equal to 1                  | `{"data":{"id":1}}`      |  |
| `[BODY].data[0].id == 1`     | The jsonpath `$.data[0].id` is equal to 1               | `{"data":[{"id":1}]}`    |  |
| `len([BODY].data) > 0`       | Array at jsonpath `$.data` has less than 5 elements     | `{"data":[{"id":1}]}`    |  |
| `len([BODY].name) == 8`      | String at jsonpath `$.name` has a length of 8           | `{"name":"john.doe"}`    | `{"name":"bob"}` |


### Alerting



#### Configuring Slack alerts

```yaml
alerting:
  slack: "https://hooks.slack.com/services/**********/**********/**********"
services:
  - name: twinnation
    interval: 30s
    url: "https://twinnation.org/health"
    alerts:
      - type: slack
        enabled: true
        description: "healthcheck failed 3 times in a row"
        send-on-resolved: true
      - type: slack
        enabled: true
        failure-threshold: 5
        description: "healthcheck failed 5 times in a row"
        send-on-resolved: true
    conditions:
      - "[STATUS] == 200"
      - "[BODY].status == UP"
      - "[RESPONSE_TIME] < 300"
```

Here's an example of what the notifications look like:

![Slack notifications](.github/assets/slack-alerts.png)


#### Configuring PagerDuty alerts

It is highly recommended to set `services[].alerts[].send-on-resolved` to `true` for alerts 
of type `pagerduty`, because unlike other alerts, the operation resulting from setting said 
parameter to `true` will not create another incident, but mark the incident as resolved on 
PagerDuty instead. 

```yaml
alerting:
  pagerduty: "********************************"
services:
  - name: twinnation
    interval: 30s
    url: "https://twinnation.org/health"
    alerts:
      - type: pagerduty
        enabled: true
        failure-threshold: 3
        success-threshold: 5
        description: "healthcheck failed 3 times in a row"
        send-on-resolved: true
    conditions:
      - "[STATUS] == 200"
      - "[BODY].status == UP"
      - "[RESPONSE_TIME] < 300"
```


#### Configuring Twilio alerts

```yaml
alerting:
  twilio:
    sid: "..."
    token: "..."
    from: "+1-234-567-8901"
    to: "+1-234-567-8901"
services:
  - name: twinnation
    interval: 30s
    url: "https://twinnation.org/health"
    alerts:
      - type: twilio
        enabled: true
        failure-threshold: 5
        description: "healthcheck failed 5 times in a row"
        send-on-resolved: true
    conditions:
      - "[STATUS] == 200"
      - "[BODY].status == UP"
      - "[RESPONSE_TIME] < 300"
```


#### Configuring custom alerts

While they're called alerts, you can use this feature to call anything. 

For instance, you could automate rollbacks by having an application that keeps tracks of new deployments, and by 
leveraging Gatus, you could have Gatus call that application endpoint when a service starts failing. Your application
would then check if the service that started failing was recently deployed, and if it was, then automatically 
roll it back.

The values `[ALERT_DESCRIPTION]` and `[SERVICE_NAME]` are automatically substituted for the alert description and the 
service name respectively in the body (`alerting.custom.body`) as well as the url (`alerting.custom.url`).

If you have `send-on-resolved` set to `true`, you may want to use `[ALERT_TRIGGERED_OR_RESOLVED]` to differentiate
the notifications. It will be replaced for either `TRIGGERED` or `RESOLVED`, based on the situation.

For all intents and purpose, we'll configure the custom alert with a Slack webhook, but you can call anything you want.

```yaml
alerting:
  custom:
    url: "https://hooks.slack.com/services/**********/**********/**********"
    method: "POST"
    body: |
      {
        "text": "[ALERT_TRIGGERED_OR_RESOLVED]: [SERVICE_NAME] - [ALERT_DESCRIPTION]"
      }
services:
  - name: twinnation
    interval: 30s
    url: "https://twinnation.org/health"
    alerts:
      - type: custom
        enabled: true
        failure-threshold: 10
        success-threshold: 3
        send-on-resolved: true
        description: "healthcheck failed 10 times in a row"
    conditions:
      - "[STATUS] == 200"
      - "[BODY].status == UP"
      - "[RESPONSE_TIME] < 300"
```


## Docker

```
docker run -p 8080:8080 --name gatus twinproduction/gatus
```


## Running the tests

```
go test ./... -mod vendor
```


## Using in Production

See the [example](example) folder.


## FAQ

### Sending a GraphQL request

By setting `services[].graphql` to true, the body will automatically be wrapped by the standard GraphQL `query` parameter.

For instance, the following configuration:
```yaml
services:
  - name: filter users by gender
    url: http://localhost:8080/playground
    method: POST
    graphql: true
    body: |
      {
        user(gender: "female") {
          id
          name
          gender
          avatar
        }
      }
    headers:
      Content-Type: application/json
    conditions:
      - "[STATUS] == 200"
      - "[BODY].data.user[0].gender == female"
```

will send a `POST` request to `http://localhost:8080/playground` with the following body:
```json
{"query":"      {\n        user(gender: \"female\") {\n          id\n          name\n          gender\n          avatar\n        }\n      }"}
```
