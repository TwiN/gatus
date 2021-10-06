![Gatus](.github/assets/logo-with-name.png)

![build](https://github.com/TwinProduction/gatus/workflows/build/badge.svg?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/TwinProduction/gatus?)](https://goreportcard.com/report/github.com/TwinProduction/gatus)
[![codecov](https://codecov.io/gh/TwinProduction/gatus/branch/master/graph/badge.svg)](https://codecov.io/gh/TwinProduction/gatus)
[![Go version](https://img.shields.io/github/go-mod/go-version/TwinProduction/gatus.svg)](https://github.com/TwinProduction/gatus)
[![Docker pulls](https://img.shields.io/docker/pulls/twinproduction/gatus.svg)](https://cloud.docker.com/repository/docker/twinproduction/gatus)
[![Join Discord server](https://img.shields.io/discord/442432928614449155.svg?label=&logo=discord&logoColor=ffffff&color=7389D8&labelColor=6A7EC2)](https://discord.gg/TDfjeHmXDQ)
[![Follow TwinProduction](https://img.shields.io/github/followers/TwinProduction?label=Follow&style=social)](https://github.com/TwinProduction)

Gatus is a health dashboard that gives you the ability to monitor your services using HTTP, ICMP, TCP, and even DNS
queries as well as evaluate the result of said queries by using a list of conditions on values like the status code,
the response time, the certificate expiration, the body and many others. The icing on top is that each of these health
checks can be paired with alerting via Slack, PagerDuty, Discord and even Twilio.

I personally deploy it in my Kubernetes cluster and let it monitor the status of my
core applications: https://status.twin.sh/

<details>
  <summary><b>Quick start</b></summary>

```
docker run -p 8080:8080 --name gatus twinproduction/gatus
```
For more details, see [Usage](#usage)
</details>


## Table of Contents
- [Why Gatus?](#why-gatus)
- [Features](#features)
- [Usage](#usage)
- [Configuration](#configuration)
  - [Conditions](#conditions)
    - [Placeholders](#placeholders)
    - [Functions](#functions)
  - [Storage](#storage)
  - [Client configuration](#client-configuration)
  - [Alerting](#alerting)
    - [Configuring Discord alerts](#configuring-discord-alerts)
    - [Configuring Mattermost alerts](#configuring-mattermost-alerts)
    - [Configuring Messagebird alerts](#configuring-messagebird-alerts)
    - [Configuring PagerDuty alerts](#configuring-pagerduty-alerts)
    - [Configuring Slack alerts](#configuring-slack-alerts)
    - [Configuring Teams alerts](#configuring-teams-alerts)
    - [Configuring Telegram alerts](#configuring-telegram-alerts)
    - [Configuring Twilio alerts](#configuring-twilio-alerts)
    - [Configuring custom alerts](#configuring-custom-alerts)
    - [Setting a default alert](#setting-a-default-alert)
  - [Maintenance](#maintenance)
- [Deployment](#deployment)
  - [Docker](#docker)
  - [Helm Chart](#helm-chart)
  - [Terraform](#terraform)
- [Running the tests](#running-the-tests)
- [Using in Production](#using-in-production)
- [FAQ](#faq)
  - [Sending a GraphQL request](#sending-a-graphql-request)
  - [Recommended interval](#recommended-interval)
  - [Default timeouts](#default-timeouts)
  - [Monitoring a TCP service](#monitoring-a-tcp-service)
  - [Monitoring a service using ICMP](#monitoring-a-service-using-icmp)
  - [Monitoring a service using DNS queries](#monitoring-a-service-using-dns-queries)
  - [Monitoring a service using STARTTLS](#monitoring-a-service-using-starttls)
  - [Monitoring a service using TLS](#monitoring-a-service-using-tls)
  - [Basic authentication](#basic-authentication)
  - [disable-monitoring-lock](#disable-monitoring-lock)
  - [Reloading configuration on the fly](#reloading-configuration-on-the-fly)
  - [Service groups](#service-groups)
  - [Exposing Gatus on a custom port](#exposing-gatus-on-a-custom-port)
  - [Badges](#badges)
    - [Uptime](#uptime)
    - [Response time](#response-time)
  - [API](#api)
  - [High level design overview](#high-level-design-overview)
- [Sponsors](#sponsors)

## Why Gatus?
Before getting into the specifics, I want to address the most common question:
> Why would I use Gatus when I can just use Prometheus’ Alertmanager, Cloudwatch or even Splunk?

Neither of these can tell you that there’s a problem if there are no clients actively calling the endpoint.
In other words, it's because monitoring metrics mostly rely on existing traffic, which effectively means that unless
your clients are already experiencing a problem, you won't be notified.

Gatus, on the other hand, allows you to configure health checks for each of your features, which in turn allows it to
monitor these features and potentially alert you before any clients are impacted.

A sign you may want to look into Gatus is by simply asking yourself whether you'd receive an alert if your load balancer
was to go down right now. Will any of your existing alerts by triggered? Your metrics won’t report an increase in errors
if there’s no traffic that makes it to your applications. This puts you in a situation where your clients are the ones
that will notify you about the degradation of your services rather than you reassuring them that you're working on
fixing the issue before they even know about it.


## Features
![Gatus dark mode](.github/assets/dark-mode.png)

The main features of Gatus are:
- **Highly flexible health check conditions**: While checking the response status may be enough for some use cases, Gatus goes much further and allows you to add conditions on the response time, the response body and even the IP address.
- **Ability to use Gatus for user acceptance tests**: Thanks to the point above, you can leverage this application to create automated user acceptance tests.
- **Very easy to configure**: Not only is the configuration designed to be as readable as possible, it's also extremely easy to add a new service or a new endpoint to monitor.
- **Alerting**: While having a pretty visual dashboard is useful to keep track of the state of your application(s), you probably don't want to stare at it all day. Thus, notifications via Slack, Mattermost, Messagebird, PagerDuty, Twilio and Teams are supported out of the box with the ability to configure a custom alerting provider for any needs you might have, whether it be a different provider or a custom application that manages automated rollbacks.
- **Metrics**
- **Low resource consumption**: As with most Go applications, the resource footprint that this application requires is negligibly small.
- **[Badges](#badges)**: ![Uptime 7d](https://status.twin.sh/api/v1/services/core_website-external/uptimes/7d/badge.svg) ![Response time 24h](https://status.twin.sh/api/v1/services/core_website-external/response-times/24h/badge.svg)


## Usage
By default, the configuration file is expected to be at `config/config.yaml`.

You can specify a custom path by setting the `GATUS_CONFIG_FILE` environment variable.

Here's a simple example:

```yaml
metrics: true         # Whether to expose metrics at /metrics
services:
  - name: website  # Name of your service, can be anything
    url: "https://twin.sh/health"
    interval: 30s     # Duration to wait between every status check (default: 60s)
    conditions:
      - "[STATUS] == 200"         # Status must be 200
      - "[BODY].status == UP"     # The json path "$.status" must be equal to UP
      - "[RESPONSE_TIME] < 300"   # Response time must be under 300ms
  - name: example
    url: "https://example.org/"
    interval: 5m
    conditions:
      - "[STATUS] == 200"
```

This example would look like this:

![Simple example](.github/assets/example.png)

Note that you can also use environment variables in the configuration file (e.g. `$DOMAIN`, `${DOMAIN}`)

If you want to test it locally, see [Docker](#docker).


## Configuration
| Parameter                                | Description                                                                   | Default        |
|:---------------------------------------- |:----------------------------------------------------------------------------- |:-------------- |
| `debug`                                  | Whether to enable debug logs.                                                 | `false`        |
| `metrics`                                | Whether to expose metrics at /metrics.                                        | `false`        |
| `storage`                                | [Storage configuration](#storage)                               | `{}`           |
| `services`                               | List of services to monitor.                                                  | Required `[]`  |
| `services[].enabled`                     | Whether to enable the service.                                                | `true`         |
| `services[].name`                        | Name of the service. Can be anything.                                         | Required `""`  |
| `services[].group`                       | Group name. Used to group multiple services together on the dashboard. <br />See [Service groups](#service-groups). | `""`           |
| `services[].url`                         | URL to send the request to.                                                   | Required `""`  |
| `services[].method`                      | Request method.                                                               | `GET`          |
| `services[].conditions`                  | Conditions used to determine the health of the service. <br />See [Conditions](#conditions). | `[]`           |
| `services[].interval`                    | Duration to wait between every status check.                                  | `60s`          |
| `services[].graphql`                     | Whether to wrap the body in a query param (`{"query":"$body"}`).              | `false`        |
| `services[].body`                        | Request body.                                                                 | `""`           |
| `services[].headers`                     | Request headers.                                                              | `{}`           |
| `services[].dns`                         | Configuration for a service of type DNS. <br />See [Monitoring a service using DNS queries](#monitoring-a-service-using-dns-queries). | `""`           |
| `services[].dns.query-type`              | Query type for DNS service.                                                   | `""`           |
| `services[].dns.query-name`              | Query name for DNS service.                                                   | `""`           |
| `services[].alerts[].type`               | Type of alert. <br />Valid types: `slack`, `discord`, `pagerduty`, `twilio`, `mattermost`, `messagebird`, `teams` `custom`. | Required `""`  |
| `services[].alerts[].enabled`            | Whether to enable the alert.                                                  | `false`        |
| `services[].alerts[].failure-threshold`  | Number of failures in a row needed before triggering the alert.               | `3`            |
| `services[].alerts[].success-threshold`  | Number of successes in a row before an ongoing incident is marked as resolved. | `2`           |
| `services[].alerts[].send-on-resolved`   | Whether to send a notification once a triggered alert is marked as resolved.  | `false`        |
| `services[].alerts[].description`        | Description of the alert. Will be included in the alert sent.                 | `""`           |
| `services[].client`                      | [Client configuration](#client-configuration).                                | `{}`          |
| `services[].ui`                          | UI configuration at the service level.                                        | `{}`           |
| `services[].ui.hide-hostname`            | Whether to include the hostname in the result.                                | `false`        |
| `services[].ui.dont-resolve-failed-conditions` | Whether to resolve failed conditions for the UI.                        | `false`        |
| `alerting`                               | [Alerting configuration](#alerting).                                          | `{}`           |
| `security`                               | Security configuration.                                                       | `{}`           |
| `security.basic`                         | Basic authentication security configuration.                                  | `{}`           |
| `security.basic.username`                | Username for Basic authentication.                                            | Required `""`  |
| `security.basic.password-sha512`         | Password's SHA512 hash for Basic authentication.                              | Required `""`  |
| `disable-monitoring-lock`                | Whether to [disable the monitoring lock](#disable-monitoring-lock).           | `false`        |
| `skip-invalid-config-update`             | Whether to ignore invalid configuration update. <br />See [Reloading configuration on the fly](#reloading-configuration-on-the-fly). | `false` |
| `web`                                    | Web configuration.                                                            | `{}`           |
| `web.address`                            | Address to listen on.                                                         | `0.0.0.0`      |
| `web.port`                               | Port to listen on.                                                            | `8080`         |
| `ui`                                     | UI configuration.                                                             | `{}`           |
| `ui.title`                               | Title of the page.                                                            | `Health Dashboard ǀ Gatus`      |
| `ui.logo`                                | URL to the logo to display                                                    | `""`         |


### Conditions
Here are some examples of conditions you can use:

| Condition                    | Description                                             | Passing values             | Failing values |
|:-----------------------------|:------------------------------------------------------- |:-------------------------- | -------------- |
| `[STATUS] == 200`            | Status must be equal to 200                             | 200                        | 201, 404, ...  |
| `[STATUS] < 300`             | Status must lower than 300                              | 200, 201, 299              | 301, 302, ...  |
| `[STATUS] <= 299`            | Status must be less than or equal to 299                | 200, 201, 299              | 301, 302, ...  |
| `[STATUS] > 400`             | Status must be greater than 400                         | 401, 402, 403, 404         | 400, 200, ...  |
| `[STATUS] == any(200, 429)`  | Status must be either 200 or 429                        | 200, 429                   | 201, 400, ...  |
| `[CONNECTED] == true`        | Connection to host must've been successful              | true, false                |  |
| `[RESPONSE_TIME] < 500`      | Response time must be below 500ms                       | 100ms, 200ms, 300ms        | 500ms, 501ms   |
| `[IP] == 127.0.0.1`          | Target IP must be 127.0.0.1                             | 127.0.0.1                  | 0.0.0.0        |
| `[BODY] == 1`                | The body must be equal to 1                             | 1                          | `{}`, `2`, ... |
| `[BODY].user.name == john`   | JSONPath value of `$.user.name` is equal to `john`      | `{"user":{"name":"john"}}` |  |
| `[BODY].data[0].id == 1`     | JSONPath value of `$.data[0].id` is equal to 1          | `{"data":[{"id":1}]}`      |  |
| `[BODY].age == [BODY].id`    | JSONPath value of `$.age` is equal JSONPath `$.id`      | `{"age":1,"id":1}`         |  |
| `len([BODY].data) < 5`       | Array at JSONPath `$.data` has less than 5 elements     | `{"data":[{"id":1}]}`      |  |
| `len([BODY].name) == 8`      | String at JSONPath `$.name` has a length of 8           | `{"name":"john.doe"}`      | `{"name":"bob"}` |
| `has([BODY].errors) == false` | JSONPath `$.errors` does not exist                     | `{"name":"john.doe"}`      | `{"errors":[]}` |
| `has([BODY].users) == true`  | JSONPath `$.users` exists                               | `{"users":[]}`             | `{}` |
| `[BODY].name == pat(john*)`  | String at JSONPath `$.name` matches pattern `john*`     | `{"name":"john.doe"}`      | `{"name":"bob"}` |
| `[BODY].id == any(1, 2)`     | Value at JSONPath `$.id` is equal to `1` or `2`         | 1, 2                       | 3, 4, 5 |
| `[CERTIFICATE_EXPIRATION] > 48h` | Certificate expiration is more than 48h away        | 49h, 50h, 123h             | 1h, 24h, ... |


#### Placeholders
| Placeholder                | Description                                                     | Example of resolved value |
|:-------------------------- |:--------------------------------------------------------------- |:------------------------- |
| `[STATUS]`                 | Resolves into the HTTP status of the request                    | 404
| `[RESPONSE_TIME]`          | Resolves into the response time the request took, in ms         | 10
| `[IP]`                     | Resolves into the IP of the target host                         | 192.168.0.232
| `[BODY]`                   | Resolves into the response body. Supports JSONPath.             | `{"name":"john.doe"}`
| `[CONNECTED]`              | Resolves into whether a connection could be established         | `true`
| `[CERTIFICATE_EXPIRATION]` | Resolves into the duration before certificate expiration        | `24h`, `48h`, 0 (if not using HTTPS)
| `[DNS_RCODE]`              | Resolves into the DNS status of the response                    | NOERROR


#### Functions
| Function   | Description                                                                                                      | Example                    |
|:-----------|:---------------------------------------------------------------------------------------------------------------- |:-------------------------- |
| `len`      | Returns the length of the object/slice. Works only with the `[BODY]` placeholder.                                | `len([BODY].username) > 8`
| `has`      | Returns `true` or `false` based on whether a given path is valid. Works only with the `[BODY]` placeholder.      | `has([BODY].errors) == false`
| `pat`      | Specifies that the string passed as parameter should be evaluated as a pattern. Works only with `==` and `!=`.   | `[IP] == pat(192.168.*)`
| `any`      | Specifies that any one of the values passed as parameters is a valid value. Works only with `==` and `!=`.       | `[BODY].ip == any(127.0.0.1, ::1)`

**NOTE**: Use `pat` only when you need to. `[STATUS] == pat(2*)` is a lot more expensive than `[STATUS] < 300`.


### Storage
| Parameter          | Description                                                                            | Default        |
|:------------------ |:-------------------------------------------------------------------------------------- |:-------------- |
| `storage`          | Storage configuration                                                                  | `{}`           |
| `storage.file`     | Path to persist the data in. If the type is `memory`, data is persisted on interval.   | `""`           |
| `storage.type`     | Type of storage. Valid types: `memory`, `sqlite`, `postgres` (ALPHA).                  | `"memory"`     |

- If `storage.type` is `memory` (default) and `storage.file` is set to a non-blank value.
  Furthermore, the data is periodically persisted, but everything remains in memory.
- If `storage.type` is `sqlite`, `storage.file` must not be blank:
```yaml
storage:
  type: sqlite
  file: data.db
```
See [examples/docker-compose-sqlite-storage](examples/docker-compose-sqlite-storage) for an example.

- If `storage.type` is `postgres`, `storage.file` must be the connection URL:
```yaml
storage:
  type: postgres
  file: "postgres://user:password@127.0.0.1:5432/gatus?sslmode=disable"
```
See [examples/docker-compose-postgres-storage](examples/docker-compose-postgres-storage) for an example.


### Client configuration
In order to support a wide range of environments, each monitored service has a unique configuration for 
the client used to send the request.

| Parameter                | Description                                                                   | Default        |
|:-------------------------|:----------------------------------------------------------------------------- |:-------------- |
| `client.insecure`        | Whether to skip verifying the server's certificate chain and host name.       | `false`        |
| `client.ignore-redirect` | Whether to ignore redirects (true) or follow them (false, default).           | `false`        |
| `client.timeout`         | Duration before timing out.                                                   | `10s`          |

Note that some of these parameters are ignored based on the type of service. For instance, there's no certificate involved
in ICMP requests (ping), therefore, setting `client.insecure` to `true` for a service of that type will not do anything.

This default configuration is as follows:
```yaml
client:
  insecure: false
  ignore-redirect: false
  timeout: 10s
```
Note that this configuration is only available under `services[]`, `alerting.mattermost` and `alerting.custom`.

Here's an example with the client configuration under `service[]`:
```yaml
services:
  - name: website
    url: "https://twin.sh/health"
    client:
      insecure: false
      ignore-redirect: false
      timeout: 10s
    conditions:
      - "[STATUS] == 200"
```


### Alerting
Gatus supports multiple alerting providers, such as Slack and PagerDuty, and supports different alerts for each
individual services with configurable descriptions and thresholds.

Note that if an alerting provider is not properly configured, all alerts configured with the provider's type will be
ignored.

| Parameter              | Description                                                                                                            | Default        |
|:-----------------------|:---------------------------------------------------------------------------------------------------------------------- |:-------|
| `alerting.discord`     | Configuration for alerts of type `discord`. <br />See [Configuring Discord alerts](#configuring-discord-alerts).             | `{}`   |
| `alerting.mattermost`  | Configuration for alerts of type `mattermost`. <br />See [Configuring Mattermost alerts](#configuring-mattermost-alerts).   | `{}`   |
| `alerting.messagebird` | Configuration for alerts of type `messagebird`. <br />See [Configuring Messagebird alerts](#configuring-messagebird-alerts). | `{}`   |
| `alerting.pagerduty`   | Configuration for alerts of type `pagerduty`. <br />See [Configuring PagerDuty alerts](#configuring-pagerduty-alerts).       | `{}`   |
| `alerting.slack`       | Configuration for alerts of type `slack`. <br />See [Configuring Slack alerts](#configuring-slack-alerts).                   | `{}`   |
| `alerting.teams`       | Configuration for alerts of type `teams`. <br />See [Configuring Teams alerts](#configuring-teams-alerts).                   | `{}`   |
| `alerting.telegram`    | Configuration for alerts of type `telegram`. <br />See [Configuring Telegram alerts](#configuring-telegram-alerts).          | `{}`   |
| `alerting.twilio`      | Settings for alerts of type `twilio`. <br />See [Configuring Twilio alerts](#configuring-twilio-alerts).                     | `{}`   |
| `alerting.custom`      | Configuration for custom actions on failure or alerts. <br />See [Configuring Custom alerts](#configuring-custom-alerts).    | `{}`   |


#### Configuring Discord alerts
| Parameter                                | Description                                  | Default        |
|:---------------------------------------- |:-------------------------------------------- |:-------------- |
| `alerting.discord`                       | Configuration for alerts of type `discord`   | `{}`           |
| `alerting.discord.webhook-url`           | Discord Webhook URL                          | Required `""`  |
| `alerting.discord.default-alert`         | Default alert configuration. <br />See [Setting a default alert](#setting-a-default-alert) | N/A       |

```yaml
alerting:
  discord: 
    webhook-url: "https://discord.com/api/webhooks/**********/**********"

services:
  - name: website
    url: "https://twin.sh/health"
    interval: 30s
    conditions:
      - "[STATUS] == 200"
      - "[BODY].status == UP"
      - "[RESPONSE_TIME] < 300"
    alerts:
      - type: discord
        enabled: true
        description: "healthcheck failed"
        send-on-resolved: true
```


#### Configuring Mattermost alerts
| Parameter                           | Description                                                                                 | Default        |
|:----------------------------------- |:------------------------------------------------------------------------------------------- |:-------------- |
| `alerting.mattermost`               | Configuration for alerts of type `mattermost`                                               | `{}`           |
| `alerting.mattermost.webhook-url`   | Mattermost Webhook URL                                                                      | Required `""`  |
| `alerting.mattermost.client`        | Client configuration. <br />See [Client configuration](#client-configuration).              | `{}`           |
| `alerting.mattermost.default-alert` | Default alert configuration. <br />See [Setting a default alert](#setting-a-default-alert). | N/A            |

```yaml
alerting:
  mattermost: 
    webhook-url: "http://**********/hooks/**********"
    client:
      insecure: true

services:
  - name: website
    url: "https://twin.sh/health"
    interval: 30s
    conditions:
      - "[STATUS] == 200"
      - "[BODY].status == UP"
      - "[RESPONSE_TIME] < 300"
    alerts:
      - type: mattermost
        enabled: true
        description: "healthcheck failed"
        send-on-resolved: true
```

Here's an example of what the notifications look like:

![Mattermost notifications](.github/assets/mattermost-alerts.png)


#### Configuring Messagebird alerts
| Parameter                            | Description                                                                   | Default        |
|:-------------------------------------|:----------------------------------------------------------------------------- |:-------------- |
| `alerting.messagebird`               | Settings for alerts of type `messagebird`                                     | `{}`           |
| `alerting.messagebird.access-key`    | Messagebird access key                                                        | Required `""`  |
| `alerting.messagebird.originator`    | The sender of the message                                                     | Required `""`  |
| `alerting.messagebird.recipients`    | The recipients of the message                                                 | Required `""`  |
| `alerting.messagebird.default-alert` | Default alert configuration. <br />See [Setting a default alert](#setting-a-default-alert) | N/A       |

Example of sending **SMS** text message alert using Messagebird:
```yaml
alerting:
  messagebird:
    access-key: "..."
    originator: "31619191918"
    recipients: "31619191919,31619191920"
services:
  - name: website
    interval: 30s
    url: "https://twin.sh/health"
    conditions:
      - "[STATUS] == 200"
      - "[BODY].status == UP"
      - "[RESPONSE_TIME] < 300"
    alerts:
      - type: messagebird
        enabled: true
        failure-threshold: 3
        send-on-resolved: true
        description: "healthcheck failed"
```


#### Configuring PagerDuty alerts
| Parameter                                | Description                                                                   | Default        |
|:---------------------------------------- |:----------------------------------------------------------------------------- |:-------------- |
| `alerting.pagerduty`                     | Configuration for alerts of type `pagerduty`                                  | `{}`           |
| `alerting.pagerduty.integration-key`     | PagerDuty Events API v2 integration key.                                      | `""`           |
| `alerting.pagerduty.default-alert`       | Default alert configuration. <br />See [Setting a default alert](#setting-a-default-alert) | N/A       |
| `alerting.pagerduty.integrations`        | Pagerduty integrations per team configurations                                |   `[]`         |
| `alerting.pagerduty.integrations[].integration-key` | Pagerduty integrationkey for a perticular team                     |   `""`         |
| `alerting.pagerduty.integrations[].group`           | the group that the integration key belongs to                      |   `""`         |

It is highly recommended to set `services[].alerts[].send-on-resolved` to `true` for alerts
of type `pagerduty`, because unlike other alerts, the operation resulting from setting said
parameter to `true` will not create another incident, but mark the incident as resolved on
PagerDuty instead.

Behavior:
- Team integration have priority over the general integration
- If no team integration is provided it will defaults to the general pagerduty integration 
- If no team integration and no general integration were provided it defaults to the first team integration provided


```yaml
alerting:
  pagerduty: 
    integration-key: "********************************"
    intergrations:
     - integration-key: "********************************"
       group: "core"

services:
  - name: website
    url: "https://twin.sh/health"
    interval: 30s
    conditions:
      - "[STATUS] == 200"
      - "[BODY].status == UP"
      - "[RESPONSE_TIME] < 300"
    alerts:
      - type: pagerduty
        enabled: true
        failure-threshold: 3
        success-threshold: 5
        send-on-resolved: true
        description: "healthcheck failed"
  - name: back-end
    group: core
    url: "https://example.org/"
    interval: 5m
    conditions:
      - "[STATUS] == 200"
      - "[CERTIFICATE_EXPIRATION] > 48h"
    alerts:
      - type: pagerduty
        enabled: true
        failure-threshold: 3
        success-threshold: 5
        send-on-resolved: true
        description: "healthcheck failed"
```


#### Configuring Slack alerts
| Parameter                        | Description                                                                   | Default        |
|:-------------------------------- |:----------------------------------------------------------------------------- |:-------------- |
| `alerting.slack`                 | Configuration for alerts of type `slack`                                      | `{}`           |
| `alerting.slack.webhook-url`     | Slack Webhook URL                                                             | Required `""`  |
| `alerting.slack.default-alert`   | Default alert configuration. <br />See [Setting a default alert](#setting-a-default-alert) | N/A       |

```yaml
alerting:
  slack: 
    webhook-url: "https://hooks.slack.com/services/**********/**********/**********"

services:
  - name: website
    url: "https://twin.sh/health"
    interval: 30s
    conditions:
      - "[STATUS] == 200"
      - "[BODY].status == UP"
      - "[RESPONSE_TIME] < 300"
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
```

Here's an example of what the notifications look like:

![Slack notifications](.github/assets/slack-alerts.png)


#### Configuring Teams alerts
| Parameter                        | Description                                                                   | Default        |
|:-------------------------------- |:----------------------------------------------------------------------------- |:-------------- |
| `alerting.teams`                 | Configuration for alerts of type `teams`                                      | `{}`           |
| `alerting.teams.webhook-url`     | Teams Webhook URL                                                             | Required `""`  |
| `alerting.teams.default-alert`   | Default alert configuration. <br />See [Setting a default alert](#setting-a-default-alert) | N/A       |

```yaml
alerting:
  teams:
    webhook-url: "https://********.webhook.office.com/webhookb2/************"

services:
  - name: website
    url: "https://twin.sh/health"
    interval: 30s
    conditions:
      - "[STATUS] == 200"
      - "[BODY].status == UP"
      - "[RESPONSE_TIME] < 300"
    alerts:
      - type: teams
        enabled: true
        description: "healthcheck failed"
        send-on-resolved: true
```

Here's an example of what the notifications look like:

![Teams notifications](.github/assets/teams-alerts.png)

#### Configuring Telegram alerts
| Parameter                           | Description                                                                   | Default        |
|:----------------------------------- |:----------------------------------------------------------------------------- |:-------------- |
| `alerting.telegram`                 | Configuration for alerts of type `telegram`                                   | `{}`           |
| `alerting.telegram.token`           | Telegram Bot Token                                                            | Required `""`  |
| `alerting.telegram.id`              | Telegram User ID                                                              | Required `""`  |
| `alerting.telegram.default-alert`   | Default alert configuration. <br />See [Setting a default alert](#setting-a-default-alert) | N/A       |

```yaml
alerting:
  telegram: 
    token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
    id: "0123456789"

services:
  - name: website
    url: "https://twin.sh/health"
    interval: 30s
    conditions:
      - "[STATUS] == 200"
      - "[BODY].status == UP"
    alerts:
      - type: telegram
        enabled: true
        send-on-resolved: true
```

Here's an example of what the notifications look like:

![Telegram notifications](.github/assets/telegram-alerts.png)


#### Configuring Twilio alerts
| Parameter                         | Description                                                                   | Default        |
|:--------------------------------- |:----------------------------------------------------------------------------- |:-------------- |
| `alerting.twilio`                 | Settings for alerts of type `twilio`                                          | `{}`           |
| `alerting.twilio.sid`             | Twilio account SID                                                            | Required `""`  |
| `alerting.twilio.token`           | Twilio auth token                                                             | Required `""`  |
| `alerting.twilio.from`            | Number to send Twilio alerts from                                             | Required `""`  |
| `alerting.twilio.to`              | Number to send twilio alerts to                                               | Required `""`  |
| `alerting.twilio.default-alert`   | Default alert configuration. <br />See [Setting a default alert](#setting-a-default-alert) | N/A       |

```yaml
alerting:
  twilio:
    sid: "..."
    token: "..."
    from: "+1-234-567-8901"
    to: "+1-234-567-8901"

services:
  - name: website
    interval: 30s
    url: "https://twin.sh/health"
    conditions:
      - "[STATUS] == 200"
      - "[BODY].status == UP"
      - "[RESPONSE_TIME] < 300"
    alerts:
      - type: twilio
        enabled: true
        failure-threshold: 5
        send-on-resolved: true
        description: "healthcheck failed"
```


#### Configuring custom alerts
| Parameter                         | Description                                                                   | Default        |
|:----------------------------------|:----------------------------------------------------------------------------- |:-------------- |
| `alerting.custom`                 | Configuration for custom actions on failure or alerts                         | `{}`           |
| `alerting.custom.url`             | Custom alerting request url                                                   | Required `""`  |
| `alerting.custom.method`          | Request method                                                                | `GET`          |
| `alerting.custom.body`            | Custom alerting request body.                                                 | `""`           |
| `alerting.custom.headers`         | Custom alerting request headers                                               | `{}`           |
| `alerting.custom.client`          | Client configuration. <br />See [Client configuration](#client-configuration).      | `{}`           |
| `alerting.custom.default-alert`   | Default alert configuration. <br />See [Setting a default alert](#setting-a-default-alert) | N/A       |

While they're called alerts, you can use this feature to call anything. 

For instance, you could automate rollbacks by having an application that keeps tracks of new deployments, and by 
leveraging Gatus, you could have Gatus call that application endpoint when a service starts failing. Your application
would then check if the service that started failing was recently deployed, and if it was, then automatically 
roll it back.

The placeholders `[ALERT_DESCRIPTION]` and `[SERVICE_NAME]` are automatically substituted for the alert description and
the service name. These placeholders can be used in the body (`alerting.custom.body`) and in the url (`alerting.custom.url`).

If you have an alert using the `custom` provider with `send-on-resolved` set to `true`, you can use the
`[ALERT_TRIGGERED_OR_RESOLVED]` placeholder to differentiate the notifications. 
The aforementioned placeholder will be replaced by `TRIGGERED` or `RESOLVED` accordingly, though it can be modified
(details at the end of this section).

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
  - name: website
    url: "https://twin.sh/health"
    interval: 30s
    conditions:
      - "[STATUS] == 200"
      - "[BODY].status == UP"
      - "[RESPONSE_TIME] < 300"
    alerts:
      - type: custom
        enabled: true
        failure-threshold: 10
        success-threshold: 3
        send-on-resolved: true
        description: "healthcheck failed"
```

Note that you can customize the resolved values for the `[ALERT_TRIGGERED_OR_RESOLVED]` placeholder like so:
```yaml
alerting:
  custom:
    placeholders:
      ALERT_TRIGGERED_OR_RESOLVED:
        TRIGGERED: "partial_outage"
        RESOLVED: "operational"
```
As a result, the `[ALERT_TRIGGERED_OR_RESOLVED]` in the body of first example of this section would be replaced by 
`partial_outage` when an alert is triggered and `operational` when an alert is resolved.


#### Setting a default alert
| Parameter                                     | Description                                                                   | Default |
|:----------------------------------------------|:------------------------------------------------------------------------------|:--------|
| `alerting.*.default-alert.enabled`            | Whether to enable the alert                                                   | N/A     |
| `alerting.*.default-alert.failure-threshold`  | Number of failures in a row needed before triggering the alert                | N/A     |
| `alerting.*.default-alert.success-threshold`  | Number of successes in a row before an ongoing incident is marked as resolved | N/A     |
| `alerting.*.default-alert.send-on-resolved`   | Whether to send a notification once a triggered alert is marked as resolved   | N/A     |
| `alerting.*.default-alert.description`        | Description of the alert. Will be included in the alert sent                  | N/A     |

While you can specify the alert configuration directly in the service definition, it's tedious and may lead to a very
long configuration file.

To avoid such problem, you can use the `default-alert` parameter present in each provider configuration:
```yaml
alerting:
  slack: 
    webhook-url: "https://hooks.slack.com/services/**********/**********/**********"
    default-alert:
      enabled: true
      description: "healthcheck failed"
      send-on-resolved: true
      failure-threshold: 5
      success-threshold: 5
```

As a result, your service configuration looks a lot tidier:
```yaml
services:
  - name: example
    url: "https://example.org"
    conditions:
      - "[STATUS] == 200"
    alerts:
      - type: slack

  - name: other-example
    url: "https://example.com"
    conditions:
      - "[STATUS] == 200"
    alerts:
      - type: slack
```

It also allows you to do things like this:
```yaml
services:
  - name: example
    url: "https://example.org"
    conditions:
      - "[STATUS] == 200"
    alerts:
      - type: slack
        failure-threshold: 5
      - type: slack
        failure-threshold: 10
      - type: slack
        failure-threshold: 15
```

Of course, you can also mix alert types:
```yaml
alerting:
  slack:
    webhook-url: "https://hooks.slack.com/services/**********/**********/**********"
    default-alert:
      enabled: true
      failure-threshold: 3
  pagerduty:
    integration-key: "********************************"
    default-alert:
      enabled: true
      failure-threshold: 5

services:
  - name: service-1
    url: "https://example.org"
    conditions:
      - "[STATUS] == 200"
    alerts:
      - type: slack
      - type: pagerduty

  - name: service-2
    url: "https://example.org"
    conditions:
      - "[STATUS] == 200"
    alerts:
      - type: slack
      - type: pagerduty
```

### Maintenance
If you have maintenance windows, you may not want to be annoyed by alerts. 
To do that, you'll have to use the maintenance configuration:

| Parameter               | Description                                                                   | Default         |
|:----------------------- |:----------------------------------------------------------------------------- |:--------------- |
| `maintenance.enabled`   | Whether the maintenance period is enabled                                     | `true`          |
| `maintenance.start`     | Time at which the maintenance window starts in `hh:mm` format (e.g. `23:00`)  | Required `""`   |
| `maintenance.duration`  | Duration of the maintenance window (e.g. `1h`, `30m`)                         | Required `""`   |
| `maintenance.every`     | Days on which the maintenance period applies (e.g. `[Monday, Thursday]`).<br />If left empty, the maintenance window applies every day | `[]` |

**Note that the maintenance configuration uses UTC.**

Here's an example:
```yaml
maintenance:
  start: 23:00
  duration: 1h
  every: [Monday, Thursday]
```
Note that you can also specify each day on separate lines:
```yaml
maintenance:
  start: 23:00
  duration: 1h
  every:
    - Monday
    - Thursday
```



## Deployment
Many examples can be found in the [examples](examples) folder, but this section will focus on the most popular ways of deploying Gatus.


### Docker
To run Gatus locally with Docker:
```
docker run -p 8080:8080 --name gatus twinproduction/gatus
```

Other than using one of the examples provided in the `examples` folder, you can also try it out locally by 
creating a configuration file, we'll call it `config.yaml` for this example, and running the following 
command:
```
docker run -p 8080:8080 --mount type=bind,source="$(pwd)"/config.yaml,target=/config/config.yaml --name gatus twinproduction/gatus
```

If you're on Windows, replace `"$(pwd)"` by the absolute path to your current directory, e.g.:
```
docker run -p 8080:8080 --mount type=bind,source=C:/Users/Chris/Desktop/config.yaml,target=/config/config.yaml --name gatus twinproduction/gatus
```

To build the image locally:
```
docker build . -t twinproduction/gatus
```


### Helm Chart
[Helm](https://helm.sh) must be installed to use the chart.
Please refer to Helm's [documentation](https://helm.sh/docs/) to get started.

Once Helm is set up properly, add the repository as follows:

```console
helm repo add gatus https://avakarev.github.io/gatus-chart
```

To get more details, please check chart's [configuration](https://github.com/avakarev/gatus-chart#configuration)
and [helmfile example](https://github.com/avakarev/gatus-chart#helmfileyaml-example)


### Terraform
Gatus can be deployed on Terraform by using the following module: [terraform-kubernetes-gatus](https://github.com/TwinProduction/terraform-kubernetes-gatus).



## Running the tests
```
go test ./... -mod vendor
```


## Using in Production
See the [Deployment](#deployment) section.


## FAQ
### Sending a GraphQL request
By setting `services[].graphql` to true, the body will automatically be wrapped by the standard GraphQL `query` parameter.

For instance, the following configuration:
```yaml
services:
  - name: filter-users-by-gender
    url: http://localhost:8080/playground
    method: POST
    graphql: true
    body: |
      {
        users(gender: "female") {
          id
          name
          gender
          avatar
        }
      }
    conditions:
      - "[STATUS] == 200"
      - "[BODY].data.users[0].gender == female"
```

will send a `POST` request to `http://localhost:8080/playground` with the following body:
```json
{"query":"      {\n        users(gender: \"female\") {\n          id\n          name\n          gender\n          avatar\n        }\n      }"}
```


### Recommended interval
> **NOTE**: This does not _really_ apply if `disable-monitoring-lock` is set to `true`, as the monitoring lock is what
> tells Gatus to only evaluate one service at a time.

To ensure that Gatus provides reliable and accurate results (i.e. response time), Gatus only evaluates one service at a time
In other words, even if you have multiple services with the exact same interval, they will not execute at the same time.

You can test this yourself by running Gatus with several services configured with a very short, unrealistic interval, 
such as 1ms. You'll notice that the response time does not fluctuate - that is because while services are evaluated on
different goroutines, there's a global lock that prevents multiple services from running at the same time.

Unfortunately, there is a drawback. If you have a lot of services, including some that are very slow or prone to time out (the default
timeout is 10s), then it means that for the entire duration of the request, no other services can be evaluated.

**This does mean that Gatus will be unable to evaluate the health of other services**. 
The interval does not include the duration of the request itself, which means that if a service has an interval of 30s 
and the request takes 2s to complete, the timestamp between two evaluations will be 32s, not 30s. 

While this does not prevent Gatus' from performing health checks on all other services, it may cause Gatus to be unable 
to respect the configured interval, for instance:
- Service A has an interval of 5s, and times out after 10s to complete 
- Service B has an interval of 5s, and takes 1ms to complete
- Service B will be unable to run every 5s, because service A's health evaluation takes longer than its interval

To sum it up, while Gatus can really handle any interval you throw at it, you're better off having slow requests with 
higher interval.

As a rule of the thumb, I personally set interval for more complex health checks to `5m` (5 minutes) and 
simple health checks used for alerting (PagerDuty/Twilio) to `30s`.


### Default timeouts
| Protocol | Timeout |
|:-------- |:------- |
| HTTP     | 10s
| TCP      | 10s
| ICMP     | 10s

To modify the timeout, see [Client configuration](#client-configuration).


### Monitoring a TCP service
By prefixing `services[].url` with `tcp:\\`, you can monitor TCP services at a very basic level:

```yaml
services:
  - name: redis
    url: "tcp://127.0.0.1:6379"
    interval: 30s
    conditions:
      - "[CONNECTED] == true"
```

Placeholders `[STATUS]` and `[BODY]` as well as the fields `services[].body`, `services[].insecure`, 
`services[].headers`, `services[].method` and `services[].graphql` are not supported for TCP services.

**NOTE**: `[CONNECTED] == true` does not guarantee that the service itself is healthy - it only guarantees that there's 
something at the given address listening to the given port, and that a connection to that address was successfully 
established.


### Monitoring a service using ICMP
By prefixing `services[].url` with `icmp:\\`, you can monitor services at a very basic level using ICMP, or more 
commonly known as "ping" or "echo":

```yaml
services:
  - name: ping-example
    url: "icmp://example.com"
    conditions:
      - "[CONNECTED] == true"
```

Only the placeholders `[CONNECTED]`, `[IP]` and `[RESPONSE_TIME]` are supported for services of type ICMP.
You can specify a domain prefixed by `icmp://`, or an IP address prefixed by `icmp://`.


### Monitoring a service using DNS queries
Defining a `dns` configuration in a service will automatically mark that service as a service of type DNS:
```yaml
services:
  - name: example-dns-query
    url: "8.8.8.8" # Address of the DNS server to use
    interval: 30s
    dns:
      query-name: "example.com"
      query-type: "A"
    conditions:
      - "[BODY] == 93.184.216.34"
      - "[DNS_RCODE] == NOERROR"
```

There are two placeholders that can be used in the conditions for services of type DNS:
- The placeholder `[BODY]` resolves to the output of the query. For instance, a query of type `A` would return an IPv4.
- The placeholder `[DNS_RCODE]` resolves to the name associated to the response code returned by the query, such as 
`NOERROR`, `FORMERR`, `SERVFAIL`, `NXDOMAIN`, etc.


### Monitoring a service using STARTTLS
If you have an email server that you want to ensure there are no problems with, monitoring it through STARTTLS 
will serve as a good initial indicator:
```yaml
services:
  - name: starttls-smtp-example
    url: "starttls://smtp.gmail.com:587"
    interval: 30m
    client:
        timeout: 5s
    conditions:
      - "[CONNECTED] == true"
      - "[CERTIFICATE_EXPIRATION] > 48h"
```


### Monitoring a service using TLS
Monitoring services using SSL/TLS encryption, such as LDAP over TLS, can help
detecting certificate expiration:
```yaml
services:
  - name: tls-ldaps-example
    url: "tls://ldap.example.com:636"
    interval: 30m
    client:
        timeout: 5s
    conditions:
      - "[CONNECTED] == true"
      - "[CERTIFICATE_EXPIRATION] > 48h"
```


### Basic authentication
You can require Basic authentication by leveraging the `security.basic` configuration:
```yaml
security:
  basic:
    username: "john.doe"
    password-sha512: "6b97ed68d14eb3f1aa959ce5d49c7dc612e1eb1dafd73b1e705847483fd6a6c809f2ceb4e8df6ff9984c6298ff0285cace6614bf8daa9f0070101b6c89899e22"
```

The example above will require that you authenticate with the username `john.doe` as well as the password `hunter2`.


### disable-monitoring-lock
Setting `disable-monitoring-lock` to `true` means that multiple services could be monitored at the same time.

While this behavior wouldn't generally be harmful, conditions using the `[RESPONSE_TIME]` placeholder could be impacted 
by the evaluation of multiple services at the same time, therefore, the default value for this parameter is `false`.

There are three main reasons why you might want to disable the monitoring lock:
- You're using Gatus for load testing (each services are periodically evaluated on a different goroutine, so 
technically, if you create 100 services with a 1 seconds interval, Gatus will send 100 requests per second)
- You have a _lot_ of services to monitor
- You want to test multiple services at very short interval (< 5s)


### Reloading configuration on the fly
For the sake on convenience, Gatus automatically reloads the configuration on the fly if the loaded configuration file
is updated while Gatus is running.

By default, the application will exit if the updating configuration is invalid, but you can configure
Gatus to continue running if the configuration file is updated with an invalid configuration by
setting `skip-invalid-config-update` to `true`.

Keep in mind that it is in your best interest to ensure the validity of the configuration file after each update you
apply to the configuration file while Gatus is running by looking at the log and making sure that you do not see the
following message:
```
The configuration file was updated, but it is not valid. The old configuration will continue being used.
```
Failure to do so may result in Gatus being unable to start if the application is restarted for whatever reason.

I recommend not setting `skip-invalid-config-update` to `true` to avoid a situation like this, but the choice is yours
to make.

**If you are not using a file storage**, updating the configuration while Gatus is running is effectively
the same as restarting the application.

**NOTE:** Updates may not be detected if the config file is bound instead of the config folder. See [#151](https://github.com/TwinProduction/gatus/issues/151).


### Service groups
Service groups are used for grouping multiple services together on the dashboard.

```yaml
services:
  - name: frontend
    group: core
    url: "https://example.org/"
    interval: 5m
    conditions:
      - "[STATUS] == 200"

  - name: backend
    group: core
    url: "https://example.org/"
    interval: 5m
    conditions:
      - "[STATUS] == 200"

  - name: monitoring
    group: internal
    url: "https://example.org/"
    interval: 5m
    conditions:
      - "[STATUS] == 200"

  - name: nas
    group: internal
    url: "https://example.org/"
    interval: 5m
    conditions:
      - "[STATUS] == 200"

  - name: random service that isn't part of a group
    url: "https://example.org/"
    interval: 5m
    conditions:
      - "[STATUS] == 200"
```

The configuration above will result in a dashboard that looks like this:

![Gatus Service Groups](.github/assets/service-groups.png)


### Exposing Gatus on a custom port
By default, Gatus is exposed on port `8080`, but you may specify a different port by setting the `web.port` parameter:
```yaml
web:
  port: 8081
```

If you're using a PaaS like Heroku that doesn't let you set a custom port and exposes it through an environment
variable instead, you can use that environment variable directly in the configuration file:
```yaml
web:
  port: ${PORT}
```

### Badges
### Uptime
![Uptime 1h](https://status.twin.sh/api/v1/services/core_website-external/uptimes/1h/badge.svg)
![Uptime 24h](https://status.twin.sh/api/v1/services/core_website-external/uptimes/24h/badge.svg)
![Uptime 7d](https://status.twin.sh/api/v1/services/core_website-external/uptimes/7d/badge.svg)

Gatus can automatically generate a SVG badge for one of your monitored services.
This allows you to put badges in your individual services' README or even create your own status page, if you 
desire.

The endpoint to generate a badge is the following:
```
/api/v1/services/{key}/uptimes/{duration}/badge.svg
```
Where:
- `{duration}` is `7d`, `24h` or `1h`
- `{key}` has the pattern `<GROUP_NAME>_<SERVICE_NAME>` in which both variables have ` `, `/`, `_`, `,` and `.` replaced by `-`.

For instance, if you want the uptime during the last 24 hours from the service `frontend` in the group `core`, 
the URL would look like this:
```
https://example.com/api/v1/services/core_frontend/uptimes/7d/badge.svg
```
If you want to display a service that is not part of a group, you must leave the group value empty:
```
https://example.com/api/v1/services/_frontend/uptimes/7d/badge.svg
```
Example:
```
![Uptime 24h](https://status.twin.sh/api/v1/services/core_website-external/uptimes/24h/badge.svg)
```
If you'd like to see a visual example of each badges available, you can simply navigate to the service's detail page.


### Response time
![Response time 1h](https://status.twin.sh/api/v1/services/core_website-external/response-times/1h/badge.svg)
![Response time 24h](https://status.twin.sh/api/v1/services/core_website-external/response-times/24h/badge.svg)
![Response time 7d](https://status.twin.sh/api/v1/services/core_website-external/response-times/7d/badge.svg)

The endpoint to generate a badge is the following:
```
/api/v1/services/{key}/response-times/{duration}/badge.svg
```
Where:
- `{duration}` is `7d`, `24h` or `1h`
- `{key}` has the pattern `<GROUP_NAME>_<SERVICE_NAME>` in which both variables have ` `, `/`, `_`, `,` and `.` replaced by `-`.


### API
Gatus provides a simple read-only API which can be queried in order to programmatically determine service status and history.

All services are available via a GET request to the following endpoint:
```
/api/v1/services/statuses
````
Example: https://status.twin.sh/api/v1/services/statuses

Specific services can also be queried by using the following pattern:
```
/api/v1/services/{group}_{service}/statuses
```
Example: https://status.twin.sh/api/v1/services/core_website-home/statuses

Gzip compression will be used if the `Accept-Encoding` HTTP header contains `gzip`.

The API will return a JSON payload with the `Content-Type` response header set to `application/json`. 
No such header is required to query the API.


### High level design overview
![Gatus diagram](.github/assets/gatus-diagram.png)


## Sponsors
You can find the full list of sponsors [here](https://github.com/sponsors/TwinProduction).

[<img src="https://github.com/math280h.png" width="50" />](https://github.com/math280h)
[<img src="https://github.com/pyroscope-io.png" width="50" />](https://github.com/pyroscope-io)
