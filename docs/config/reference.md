# Gatus Configuration Reference

This document describes all configuration options available in Gatus configuration files.

Gatus reads one or more YAML configuration files from a directory, `/config` by default (overridable via the `GATUS_CONFIG_FILE` environment variable). When multiple files are present, they are merged. All string values support `${ENV_VAR}` expansion; use `$$` for a literal dollar sign.

## Table of Contents

- [Top-level options](#top-level-options)
- [Endpoints](#endpoints)
  - [HTTP / TCP / ICMP endpoints](#http--tcp--icmp-endpoints)
  - [DNS endpoints](#dns-endpoints)
  - [SSH endpoints](#ssh-endpoints)
  - [Conditions](#conditions)
  - [Client configuration](#client-configuration)
  - [Per-endpoint UI options](#per-endpoint-ui-options)
  - [Alerts on an endpoint](#alerts-on-an-endpoint)
  - [Maintenance windows on an endpoint](#maintenance-windows-on-an-endpoint)
- [External endpoints](#external-endpoints)
- [Suites](#suites)
- [Alerting](#alerting)
  - [Alert defaults](#alert-defaults)
  - [Provider: aws-ses](#provider-aws-ses)
  - [Provider: clickup](#provider-clickup)
  - [Provider: custom](#provider-custom)
  - [Provider: datadog](#provider-datadog)
  - [Provider: discord](#provider-discord)
  - [Provider: email](#provider-email)
  - [Provider: gitea](#provider-gitea)
  - [Provider: github](#provider-github)
  - [Provider: gitlab](#provider-gitlab)
  - [Provider: googlechat](#provider-googlechat)
  - [Provider: gotify](#provider-gotify)
  - [Provider: homeassistant](#provider-homeassistant)
  - [Provider: ifttt](#provider-ifttt)
  - [Provider: ilert](#provider-ilert)
  - [Provider: incident-io](#provider-incident-io)
  - [Provider: line](#provider-line)
  - [Provider: matrix](#provider-matrix)
  - [Provider: mattermost](#provider-mattermost)
  - [Provider: messagebird](#provider-messagebird)
  - [Provider: n8n](#provider-n8n)
  - [Provider: newrelic](#provider-newrelic)
  - [Provider: ntfy](#provider-ntfy)
  - [Provider: opsgenie](#provider-opsgenie)
  - [Provider: pagerduty](#provider-pagerduty)
  - [Provider: plivo](#provider-plivo)
  - [Provider: pushover](#provider-pushover)
  - [Provider: rocketchat](#provider-rocketchat)
  - [Provider: sendgrid](#provider-sendgrid)
  - [Provider: signal](#provider-signal)
  - [Provider: signl4](#provider-signl4)
  - [Provider: slack](#provider-slack)
  - [Provider: splunk](#provider-splunk)
  - [Provider: squadcast](#provider-squadcast)
  - [Provider: teams](#provider-teams)
  - [Provider: teams-workflows](#provider-teams-workflows)
  - [Provider: telegram](#provider-telegram)
  - [Provider: twilio](#provider-twilio)
  - [Provider: vonage](#provider-vonage)
  - [Provider: webex](#provider-webex)
  - [Provider: zapier](#provider-zapier)
  - [Provider: zulip](#provider-zulip)
- [Storage](#storage)
- [Web server](#web-server)
- [Security](#security)
- [UI](#ui)
- [Maintenance](#maintenance)
- [Connectivity checker](#connectivity-checker)
- [SSH tunneling](#ssh-tunneling)
- [Remote instances (alpha)](#remote-instances-alpha)
- [Announcements](#announcements)

---

## Top-level options

```yaml
debug: false
metrics: false
skip-invalid-config-update: false
concurrency: 3

security: ...
alerting: ...
endpoints: ...
external-endpoints: ...
suites: ...
storage: ...
web: ...
ui: ...
maintenance: ...
remote: ...
connectivity: ...
tunneling: ...
announcements: ...
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `debug` | bool | `false` | Enable debug logging. **Deprecated** — use the `GATUS_LOG_LEVEL` environment variable instead. |
| `metrics` | bool | `false` | Expose a Prometheus metrics endpoint at `/metrics`. |
| `skip-invalid-config-update` | bool | `false` | When watching for config file changes, ignore an update that would produce an invalid configuration instead of stopping. |
| `concurrency` | int | `3` | Maximum number of endpoints or suites evaluated concurrently. Set to `0` for unlimited. |

---

## Endpoints

Endpoints are the services Gatus monitors. Each endpoint has a URL, an interval, and one or more conditions that determine its health.

```yaml
endpoints:
  - name: my-api
    group: backend
    url: https://api.example.com/health
    interval: 1m
    conditions:
      - "[STATUS] == 200"
      - "[RESPONSE_TIME] < 500"
```

### HTTP / TCP / ICMP endpoints

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `name` | string | **required** | Unique display name. Combined with `group` to form the endpoint key; must not contain quotes or backslashes. |
| `group` | string | `""` | Group this endpoint belongs to. Endpoints with the same group are shown together in the UI. |
| `enabled` | bool | `true` | Set to `false` to disable monitoring without removing the endpoint. |
| `url` | string | **required** | URL to monitor. Supported schemes: `http://`, `https://`, `tcp://`, `icmp://`, `starttls://`, `ssh://`, or a bare IP/hostname for DNS checks (see [DNS endpoints](#dns-endpoints)). |
| `method` | string | `GET` | HTTP method (`GET`, `POST`, `PUT`, `PATCH`, `DELETE`, etc.). Ignored for non-HTTP endpoints. |
| `body` | string | `""` | HTTP request body. |
| `graphql` | bool | `false` | Wrap `body` in a GraphQL `{"query": ...}` envelope. |
| `headers` | map[string]string | `{}` | HTTP request headers. |
| `extra-labels` | map[string]string | `{}` | Extra labels added to Prometheus metrics for this endpoint. Has no effect when `metrics: false`. |
| `interval` | duration | `1m` | How often to evaluate the endpoint. |
| `conditions` | []string | **required** | One or more [condition expressions](#conditions) that must all pass for the endpoint to be considered healthy. |
| `alerts` | []Alert | `[]` | [Alert configurations](#alerts-on-an-endpoint) for this endpoint. |
| `maintenance-windows` | []Maintenance | `[]` | [Maintenance windows](#maintenance-windows-on-an-endpoint) scoped to this endpoint. |
| `dns` | DNS | — | Present only for [DNS endpoints](#dns-endpoints). |
| `ssh` | SSH | — | Present only for [SSH endpoints](#ssh-endpoints). |
| `client` | Client | (defaults) | [HTTP/network client settings](#client-configuration). |
| `ui` | EndpointUI | (defaults) | [Per-endpoint UI customisation](#per-endpoint-ui-options). |

#### Supported URL schemes

| Scheme | What is checked |
|--------|----------------|
| `http://` / `https://` | HTTP request; status code, body, response time, certificate expiry |
| `tcp://host:port` | TCP connection success |
| `icmp://host` | ICMP ping (requires raw socket privileges or the binary to be `setuid`) |
| `starttls://host:port` | STARTTLS handshake (e.g., SMTP/IMAP) |
| `ssh://host:port` | SSH connection (requires `ssh` config block) |
| bare IP / hostname | DNS resolver query (requires `dns` config block) |

---

### DNS endpoints

When the `dns` block is present, `url` is the address of the DNS server to query (e.g., `8.8.8.8`).

```yaml
endpoints:
  - name: dns-check
    url: "8.8.8.8"
    dns:
      query-name: "example.com"
      query-type: "A"
    conditions:
      - "[BODY] == pat(*.*.*.*)"
      - "[DNS_RCODE] == NOERROR"
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `query-name` | string | **required** | Domain name to resolve. |
| `query-type` | string | **required** | DNS record type: `A`, `AAAA`, `CNAME`, `MX`, `NS`, `TXT`, `SOA`, etc. |

The response body (`[BODY]`) is the resolved value (IP address, CNAME target, etc.). `[DNS_RCODE]` contains the DNS response code (`NOERROR`, `NXDOMAIN`, `SERVFAIL`, …).

---

### SSH endpoints

SSH endpoints execute a command on a remote host over SSH and check the output.

```yaml
endpoints:
  - name: remote-disk
    url: "ssh://bastion.example.com:22"
    body: "df -h /"
    ssh:
      username: monitor
      private-key: |
        -----BEGIN OPENSSH PRIVATE KEY-----
        ...
        -----END OPENSSH PRIVATE KEY-----
    conditions:
      - "[CONNECTED] == true"
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `username` | string | `""` | SSH username. |
| `password` | string | `""` | SSH password. Use either `password` or `private-key`. |
| `private-key` | string | `""` | PEM-encoded SSH private key. Use either `private-key` or `password`. |

---

### Conditions

A condition is a string expression of the form `[PLACEHOLDER] OPERATOR VALUE`.

**Operators:** `==`, `!=`, `<`, `>`, `<=`, `>=`

**Placeholders:**

| Placeholder | Type | Description |
|-------------|------|-------------|
| `[STATUS]` | int | HTTP response status code. |
| `[RESPONSE_TIME]` | int | Round-trip time in milliseconds. |
| `[IP]` | string | Resolved IP address of the host. |
| `[CONNECTED]` | bool | Whether the connection was established (`true`/`false`). |
| `[CERTIFICATE_EXPIRATION]` | duration | Time until the TLS certificate expires. Can be compared to a duration like `48h`. |
| `[DOMAIN_EXPIRATION]` | duration | Time until the domain registration expires. Requires `interval >= 5m`. |
| `[DNS_RCODE]` | string | DNS response code (e.g., `NOERROR`, `NXDOMAIN`). |
| `[BODY]` | string | Full response body. |
| `[BODY].field` | any | JSONPath into the response body (e.g., `[BODY].status`, `[BODY].data[0].name`). |
| `[CONTEXT].key` | any | Value stored in a suite's shared context (suites only). |

**Functions:**

| Function | Applies to | Description |
|----------|-----------|-------------|
| `len(placeholder)` | string, array | Length of the value (string length or array element count). |
| `has(placeholder)` | any | `true` if the key exists, `false` otherwise (useful for optional JSON fields). |
| `pat(pattern)` | string | Glob-style pattern match — `*` matches any sequence (e.g., `pat(192.168.*.*)`). |
| `any(v1,v2,...)` | any | `true` if the placeholder equals any of the listed values. |

**Examples:**

```yaml
conditions:
  - "[STATUS] == 200"
  - "[STATUS] != 404"
  - "[RESPONSE_TIME] < 1000"
  - "[BODY].status == UP"
  - "[BODY].items[0].id == 42"
  - "len([BODY].items) > 0"
  - "has([BODY].data) == true"
  - "[IP] == pat(10.0.*.*)"
  - "[STATUS] == any(200,201,204)"
  - "[CERTIFICATE_EXPIRATION] > 168h"
  - "[DOMAIN_EXPIRATION] > 720h"
  - "[DNS_RCODE] == NOERROR"
  - "[CONNECTED] == true"
```

---

### Client configuration

The `client` block controls connection behaviour for HTTP, TCP, and ICMP endpoints.

```yaml
endpoints:
  - name: internal-api
    url: https://internal.example.com/health
    client:
      timeout: 30s
      insecure: false
      ignore-redirect: false
      proxy-url: http://proxy.example.com:3128
      dns-resolver: tcp://8.8.8.8:53
      network: ip4
      tls:
        certificate-file: /certs/client.crt
        private-key-file: /certs/client.key
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `timeout` | duration | `10s` | Maximum time to wait for a response. |
| `insecure` | bool | `false` | Skip TLS certificate verification. |
| `ignore-redirect` | bool | `false` | Do not follow HTTP redirects. |
| `proxy-url` | string | `""` | HTTP proxy URL (e.g., `http://user:pass@proxy:3128`). |
| `dns-resolver` | string | `""` | Custom DNS resolver address (e.g., `tcp://8.8.8.8:53`). |
| `network` | string | `ip` | Network family for ICMP: `ip` (both), `ip4` (IPv4 only), `ip6` (IPv6 only). |
| `tunnel` | string | `""` | Name of an [SSH tunnel](#ssh-tunneling) to route traffic through. |
| `oauth2` | OAuth2 | — | OAuth2 client-credentials configuration. |
| `identity-aware-proxy` | IAP | — | Google Cloud Identity-Aware Proxy configuration. |
| `tls` | TLS | — | Mutual-TLS client certificate configuration. |

#### OAuth2 client credentials

```yaml
client:
  oauth2:
    token-url: https://auth.example.com/oauth/token
    client-id: my-client
    client-secret: ${OAUTH2_SECRET}
    scopes:
      - read:health
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `token-url` | string | **required** | Token endpoint URL. |
| `client-id` | string | **required** | OAuth2 client ID. |
| `client-secret` | string | **required** | OAuth2 client secret. |
| `scopes` | []string | **required** | List of requested scopes. |

#### Google Cloud IAP

```yaml
client:
  identity-aware-proxy:
    audience: /projects/123/apps/my-app
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `audience` | string | **required** | IAP audience identifier. |

#### Client TLS certificate

```yaml
client:
  tls:
    certificate-file: /certs/client.crt
    private-key-file: /certs/client.key
    renegotiation: once
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `certificate-file` | string | `""` | Path to PEM-encoded client certificate. |
| `private-key-file` | string | `""` | Path to PEM-encoded private key. |
| `renegotiation` | string | `""` | TLS renegotiation policy: `never`, `once`, or `freely`. |

---

### Per-endpoint UI options

```yaml
endpoints:
  - name: my-api
    url: https://api.example.com/health
    ui:
      hide-url: true
      hide-hostname: false
      hide-conditions: false
      hide-port: false
      hide-errors: false
      dont-resolve-failed-conditions: false
      resolve-successful-conditions: false
      badge:
        response-time:
          thresholds: [50, 200, 300, 500, 750]
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `hide-url` | bool | `false` | Hide the URL from the dashboard (useful for endpoints with secrets in the URL). |
| `hide-hostname` | bool | `false` | Hide the resolved hostname. |
| `hide-port` | bool | `false` | Hide the port number. |
| `hide-conditions` | bool | `false` | Hide condition evaluation results. |
| `hide-errors` | bool | `false` | Hide error messages. |
| `dont-resolve-failed-conditions` | bool | `false` | Do not expand the value of a failed condition for display. |
| `resolve-successful-conditions` | bool | `false` | Expand the value of passing conditions for display. |
| `badge.response-time.thresholds` | []int | `[50,200,300,500,750]` | Five ascending millisecond thresholds that control the colour bands on the response-time badge (green → red). |

---

### Alerts on an endpoint

Each endpoint can send alerts via one or more configured alert providers.

```yaml
endpoints:
  - name: my-api
    url: https://api.example.com/health
    alerts:
      - type: slack
        failure-threshold: 3
        success-threshold: 2
        send-on-resolved: true
        description: "API health check failed"
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `type` | string | **required** | Alert provider type (see [Alerting](#alerting) for the full list). |
| `enabled` | bool | `true` | Enable or disable this alert. |
| `failure-threshold` | int | `3` | Number of consecutive failures before the alert fires. |
| `success-threshold` | int | `2` | Number of consecutive successes required to resolve the alert. |
| `minimum-reminder-interval` | duration | `0` | Minimum interval between repeated notifications while the alert is active. Must be `>= 5m` when set. |
| `description` | string | `""` | Free-text description included in the alert message. |
| `send-on-resolved` | bool | `false` | Send a follow-up notification when the endpoint recovers. |
| `provider-override` | map | `{}` | Provider-specific fields that override the global provider configuration for this alert (keys vary by provider). |

---

### Maintenance windows on an endpoint

Endpoints can have maintenance windows during which failures are not alerted.

```yaml
endpoints:
  - name: my-api
    url: https://api.example.com/health
    maintenance-windows:
      - start: "02:00"
        duration: 2h
        timezone: America/New_York
        every: [Wednesday]
```

See the [Maintenance](#maintenance) section for the field reference — the fields are identical.

---

## External endpoints

External endpoints receive their health status from an external push (not polled by Gatus). They appear in the dashboard like regular endpoints.

```yaml
external-endpoints:
  - name: my-external-service
    group: third-party
    token: ${EXTERNAL_TOKEN}
    alerts:
      - type: slack
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `name` | string | **required** | Display name. |
| `group` | string | `""` | Group the endpoint belongs to. |
| `enabled` | bool | `true` | Enable or disable the endpoint. |
| `token` | string | **required** | Secret token that must be supplied in the `Authorization: Bearer <token>` header when pushing status. |
| `alerts` | []Alert | `[]` | [Alert configurations](#alerts-on-an-endpoint). |

Status is pushed via `POST /api/v1/endpoints/{key}/external?success=true` (or `false`).

---

## Suites

Suites execute a sequence of endpoints in order, passing context between them. Useful for multi-step API workflows (login → fetch → assert).

```yaml
suites:
  - name: checkout-flow
    group: e2e
    interval: 10m
    timeout: 5m
    context:
      base-url: https://shop.example.com
    endpoints:
      - name: login
        url: "${base-url}/api/login"
        method: POST
        body: '{"user":"test","pass":"test"}'
        conditions:
          - "[STATUS] == 200"
        store:
          token: "[BODY].token"

      - name: place-order
        url: "${base-url}/api/orders"
        method: POST
        headers:
          Authorization: "Bearer [CONTEXT].token"
        conditions:
          - "[STATUS] == 201"
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `name` | string | **required** | Unique suite name. |
| `group` | string | `""` | Group the suite belongs to. |
| `enabled` | bool | `true` | Enable or disable the suite. |
| `interval` | duration | `10m` | How often to run the suite. |
| `timeout` | duration | `5m` | Maximum total execution time for one suite run. |
| `context` | map | `{}` | Initial key/value pairs available to suite endpoints as `[CONTEXT].key`. |
| `endpoints` | []Endpoint | **required** | Ordered list of endpoints to execute. |

**Suite-specific endpoint fields:**

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `store` | map[string]string | — | Extract values from the response and store them in the suite context. Key is the context variable name; value is a condition placeholder (e.g., `"[BODY].token"`). |
| `always-run` | bool | `false` | Run this endpoint even if a previous suite step failed. |

All other [endpoint fields](#http--tcp--icmp-endpoints) are also valid inside a suite's endpoint list.

---

## Alerting

The `alerting` block configures one or more alert providers. Provider keys map to the `type` used in endpoint alert configurations.

```yaml
alerting:
  slack:
    webhook-url: https://hooks.slack.com/services/...
    default-alert:
      failure-threshold: 3
      send-on-resolved: true
  pagerduty:
    integration-key: ${PAGERDUTY_KEY}
```

### Alert defaults

Every provider supports a `default-alert` key that sets endpoint-level alert defaults for that provider. These values are merged with (and overridden by) per-endpoint alert settings.

```yaml
alerting:
  slack:
    webhook-url: https://hooks.slack.com/services/...
    default-alert:
      failure-threshold: 5
      success-threshold: 1
      send-on-resolved: true
      description: "Monitored by Gatus"
```

`default-alert` accepts all fields from [Alerts on an endpoint](#alerts-on-an-endpoint) except `type`.

### Provider overrides

Most providers support an `overrides` list that allows group-specific configuration:

```yaml
alerting:
  slack:
    webhook-url: https://hooks.slack.com/services/default-channel/...
    overrides:
      - group: payments
        webhook-url: https://hooks.slack.com/services/payments-channel/...
```

Each entry in `overrides` has a `group` field plus any subset of the provider's own fields.

---

### Provider: aws-ses

Send email alerts via Amazon Simple Email Service.

```yaml
alerting:
  aws-ses:
    access-key-id: ${AWS_ACCESS_KEY_ID}
    secret-access-key: ${AWS_SECRET_ACCESS_KEY}
    region: us-east-1
    from: gatus@example.com
    to: ops@example.com
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `access-key-id` | string | | AWS access key ID (uses default credential chain if omitted). |
| `secret-access-key` | string | | AWS secret access key. |
| `region` | string | | AWS region (e.g., `us-east-1`). |
| `from` | string | **yes** | Sender email address. |
| `to` | string | **yes** | Recipient email address(es), comma-separated. |

---

### Provider: clickup

Create ClickUp tasks when an endpoint fails.

```yaml
alerting:
  clickup:
    list-id: "123456789"
    token: ${CLICKUP_TOKEN}
    assignees:
      - "user-id-1"
    priority: high
    status: Open
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `api-url` | string | | ClickUp API base URL (defaults to `https://api.clickup.com`). |
| `list-id` | string | **yes** | ID of the list where tasks are created. |
| `token` | string | **yes** | ClickUp authentication token. |
| `assignees` | []string | | User IDs to assign to new tasks. |
| `status` | string | | Initial task status. |
| `priority` | string | | Task priority: `urgent`, `high`, `normal`, `low`, or `none`. |
| `notify-all` | bool | | Notify all list members. |
| `name` | string | | Task name template. |
| `content` | string | | Task body content (Markdown). |

---

### Provider: custom

Send alerts to any HTTP endpoint.

```yaml
alerting:
  custom:
    url: https://webhook.example.com/alerts
    method: POST
    headers:
      Content-Type: application/json
      Authorization: Bearer ${WEBHOOK_TOKEN}
    body: |
      {
        "endpoint": "[ENDPOINT_NAME]",
        "status": "[ALERT_TRIGGERED]",
        "description": "[ALERT_DESCRIPTION]",
        "conditions": "[ENDPOINT_CONDITIONS]"
      }
    placeholders:
      ALERT_TRIGGERED:
        true: "firing"
        false: "resolved"
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `url` | string | **yes** | Webhook URL. |
| `method` | string | | HTTP method (default `GET`). |
| `body` | string | | Request body. Supports built-in placeholders (see below). |
| `headers` | map[string]string | | Request headers. |
| `placeholders` | map[string]map[string]string | | Map of placeholder name → `{true: "value", false: "value"}` to substitute when the alert fires or resolves. |
| `client` | Client | | [HTTP client settings](#client-configuration). |

**Built-in body/header placeholders:**

| Placeholder | Description |
|-------------|-------------|
| `[ENDPOINT_NAME]` | Endpoint name. |
| `[ENDPOINT_GROUP]` | Endpoint group. |
| `[ENDPOINT_URL]` | Endpoint URL. |
| `[ALERT_TRIGGERED]` | `true` when firing, `false` when resolved (before `placeholders` substitution). |
| `[ALERT_DESCRIPTION]` | Alert description. |
| `[ENDPOINT_CONDITIONS]` | Condition results as a newline-separated string. |
| `[RESULT_ERRORS]` | Errors from the last evaluation. |

---

### Provider: datadog

Create Datadog events.

```yaml
alerting:
  datadog:
    api-key: ${DATADOG_API_KEY}
    site: datadoghq.com
    tags:
      - "env:production"
      - "team:platform"
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `api-key` | string | **yes** | Datadog API key. |
| `site` | string | | Datadog site: `datadoghq.com` (default) or `datadoghq.eu`. |
| `tags` | []string | | Additional tags attached to events. |

---

### Provider: discord

Send alerts to a Discord channel via webhook.

```yaml
alerting:
  discord:
    webhook-url: https://discord.com/api/webhooks/...
    title: "Gatus Alert"
    message-content: "<@&role-id>"
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `webhook-url` | string | **yes** | Discord incoming webhook URL. |
| `title` | string | | Embed title. |
| `message-content` | string | | Message content prepended to the embed (useful for `@` mentions). |

---

### Provider: email

Send email alerts via SMTP.

```yaml
alerting:
  email:
    from: gatus@example.com
    username: gatus@example.com
    password: ${SMTP_PASSWORD}
    host: smtp.example.com
    port: 587
    to: ops@example.com
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `from` | string | **yes** | Sender email address. |
| `username` | string | | SMTP authentication username. |
| `password` | string | | SMTP authentication password. |
| `host` | string | **yes** | SMTP server hostname. |
| `port` | int | **yes** | SMTP server port (1–65535). |
| `to` | string | **yes** | Recipient email address(es), comma-separated. |
| `client` | Client | | [HTTP client settings](#client-configuration) (controls TLS behaviour). |

---

### Provider: gitea

Open Gitea issues on failure, close them on recovery.

```yaml
alerting:
  gitea:
    repository-url: https://gitea.example.com/org/repo
    token: ${GITEA_TOKEN}
    assignees:
      - octocat
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `repository-url` | string | **yes** | Full URL to the Gitea repository. |
| `token` | string | **yes** | Personal access token with read/write access to issues and read access to metadata. |
| `assignees` | []string | | Usernames to assign to opened issues. |
| `client` | Client | | [HTTP client settings](#client-configuration). |

---

### Provider: github

Open GitHub issues on failure, close them on recovery.

```yaml
alerting:
  github:
    repository-url: https://github.com/org/repo
    token: ${GITHUB_TOKEN}
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `repository-url` | string | **yes** | Full URL to the GitHub repository. |
| `token` | string | **yes** | Personal access token or fine-grained token with read/write access to issues. |

---

### Provider: gitlab

Send alerts to GitLab via webhook.

```yaml
alerting:
  gitlab:
    webhook-url: https://gitlab.example.com/...
    authorization-key: ${GITLAB_KEY}
    severity: critical
    environment-name: production
    service: my-api
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `webhook-url` | string | **yes** | GitLab webhook URL. |
| `authorization-key` | string | **yes** | GitLab authorization key for the webhook. |
| `severity` | string | | Alert severity: `critical`, `high`, `medium`, `low`, `info`, or `unknown`. |
| `monitoring-tool` | string | | Monitoring tool name (default `gatus`). |
| `environment-name` | string | | GitLab environment associated with the alert. |
| `service` | string | | Affected service name. |

---

### Provider: googlechat

Send alerts to a Google Chat space via webhook.

```yaml
alerting:
  googlechat:
    webhook-url: https://chat.googleapis.com/v1/spaces/...
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `webhook-url` | string | **yes** | Google Chat incoming webhook URL. |
| `client` | Client | | [HTTP client settings](#client-configuration). |

---

### Provider: gotify

Push notifications via Gotify.

```yaml
alerting:
  gotify:
    server-url: https://gotify.example.com
    token: ${GOTIFY_TOKEN}
    priority: 5
    title: "Gatus"
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `server-url` | string | **yes** | Gotify server URL. |
| `token` | string | **yes** | Gotify application token. |
| `priority` | int | | Message priority (default `5`). |
| `title` | string | | Message title. |

---

### Provider: homeassistant

Fire Home Assistant webhook automations.

```yaml
alerting:
  homeassistant:
    url: https://homeassistant.local:8123
    token: ${HA_TOKEN}
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `url` | string | **yes** | Home Assistant base URL. |
| `token` | string | **yes** | Long-lived access token. |

---

### Provider: ifttt

Trigger IFTTT webhooks.

```yaml
alerting:
  ifttt:
    webhook-key: ${IFTTT_KEY}
    event-name: gatus_alert
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `webhook-key` | string | **yes** | IFTTT webhook key. |
| `event-name` | string | **yes** | IFTTT event name to trigger. |

---

### Provider: ilert

Create ilert alerts.

```yaml
alerting:
  ilert:
    integration-key: ${ILERT_KEY}
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `integration-key` | string | **yes** | ilert integration key. |

---

### Provider: incident-io

Send alerts to incident.io.

```yaml
alerting:
  incident-io:
    url: https://api.incident.io/v2/alert_events/http/...
    auth-token: ${INCIDENT_IO_TOKEN}
    source-url: https://gatus.example.com
    metadata:
      team: platform
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `url` | string | **yes** | incident.io alert event webhook URL (must start with `https://api.incident.io/v2/alert_events/http/`). |
| `auth-token` | string | **yes** | Authentication token. |
| `source-url` | string | | URL pointing to the Gatus dashboard for context. |
| `metadata` | map | | Arbitrary metadata attached to the alert. |

---

### Provider: line

Send LINE messages.

```yaml
alerting:
  line:
    channel-access-token: ${LINE_TOKEN}
    user-ids:
      - Uxxxxxxxxxxxx
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `channel-access-token` | string | **yes** | LINE Messaging API channel access token. |
| `user-ids` | []string | **yes** | LINE user IDs to send messages to. |

---

### Provider: matrix

Send Matrix room messages.

```yaml
alerting:
  matrix:
    access-token: ${MATRIX_TOKEN}
    internal-room-id: "!roomid:matrix.org"
    server-url: https://matrix.example.com
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `server-url` | string | | Matrix homeserver URL (default `https://matrix-client.matrix.org`). |
| `access-token` | string | **yes** | Bot user access token. |
| `internal-room-id` | string | **yes** | Room ID to send messages to (e.g., `!abc123:matrix.org`). |

---

### Provider: mattermost

Send Mattermost channel messages via incoming webhook.

```yaml
alerting:
  mattermost:
    webhook-url: https://mattermost.example.com/hooks/...
    channel: "#alerts"
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `webhook-url` | string | **yes** | Mattermost incoming webhook URL. |
| `channel` | string | | Channel override (e.g., `#alerts`). Uses the webhook's default channel if omitted. |
| `client` | Client | | [HTTP client settings](#client-configuration). |

---

### Provider: messagebird

Send SMS alerts via MessageBird.

```yaml
alerting:
  messagebird:
    access-key: ${MESSAGEBIRD_KEY}
    originator: "Gatus"
    recipients: "+15550001234"
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `access-key` | string | **yes** | MessageBird API access key. |
| `originator` | string | **yes** | Sender name or phone number. |
| `recipients` | string | **yes** | Recipient phone numbers, comma-separated. |

---

### Provider: n8n

Trigger an n8n workflow via webhook.

```yaml
alerting:
  n8n:
    webhook-url: https://n8n.example.com/webhook/...
    title: "Gatus Alert"
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `webhook-url` | string | **yes** | n8n webhook URL. |
| `title` | string | | Message title. |

---

### Provider: newrelic

Send events to New Relic.

```yaml
alerting:
  newrelic:
    insert-key: ${NEWRELIC_INSERT_KEY}
    account-id: ${NEWRELIC_ACCOUNT_ID}
    region: US
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `insert-key` | string | **yes** | New Relic Insights Insert key. |
| `account-id` | string | **yes** | New Relic account ID. |
| `region` | string | | `US` (default) or `EU`. |

---

### Provider: ntfy

Push notifications via ntfy.

```yaml
alerting:
  ntfy:
    topic: gatus-alerts
    url: https://ntfy.sh
    priority: 3
    token: ${NTFY_TOKEN}
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `topic` | string | **yes** | ntfy topic to publish to. |
| `url` | string | | ntfy server URL (default `https://ntfy.sh`). |
| `priority` | int | | Message priority 1–5 (default `3`). |
| `token` | string | | Access token for protected topics (must start with `tk_`). |
| `email` | string | | Email address to notify in addition to the push notification. |
| `click` | string | | URL opened when the notification is tapped. |
| `disable-firebase` | bool | | Disable Firebase push delivery. |
| `disable-cache` | bool | | Disable server-side caching. |

---

### Provider: opsgenie

Create OpsGenie alerts.

```yaml
alerting:
  opsgenie:
    api-key: ${OPSGENIE_KEY}
    priority: P2
    tags:
      - production
      - gatus
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `api-key` | string | **yes** | OpsGenie API key. |
| `priority` | string | | Alert priority (`P1`–`P5`, default `P1`). |
| `source` | string | | Event source (default `gatus`). |
| `entity-prefix` | string | | Entity name prefix (default `gatus-`). |
| `alias-prefix` | string | | Alias prefix (default `gatus-healthcheck-`). |
| `tags` | []string | | Tags attached to alerts. |

---

### Provider: pagerduty

Trigger PagerDuty incidents.

```yaml
alerting:
  pagerduty:
    integration-key: ${PAGERDUTY_KEY}
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `integration-key` | string | **yes** | PagerDuty Events API v2 integration key (exactly 32 characters). |

---

### Provider: plivo

Send SMS alerts via Plivo.

```yaml
alerting:
  plivo:
    auth-id: ${PLIVO_AUTH_ID}
    auth-token: ${PLIVO_AUTH_TOKEN}
    from: "+15550001234"
    to:
      - "+15550005678"
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `auth-id` | string | **yes** | Plivo auth ID. |
| `auth-token` | string | **yes** | Plivo auth token. |
| `from` | string | **yes** | Sender phone number. |
| `to` | []string | **yes** | Recipient phone numbers. |

---

### Provider: pushover

Push notifications via Pushover.

```yaml
alerting:
  pushover:
    application-token: ${PUSHOVER_APP_TOKEN}
    user-key: ${PUSHOVER_USER_KEY}
    priority: 0
    sound: pushover
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `application-token` | string | **yes** | Pushover application token (30 characters). |
| `user-key` | string | **yes** | Pushover user or group key (30 characters). |
| `title` | string | | Notification title. |
| `priority` | int | | Message priority: `-2` (silent) to `2` (emergency), default `0`. |
| `resolved-priority` | int | | Priority for resolved notifications (default `0`). |
| `sound` | string | | Sound name (see Pushover documentation). |
| `ttl` | int | | Message TTL in seconds. |
| `device` | string | | Target a specific device (max 25 characters). |

---

### Provider: rocketchat

Send Rocket.Chat messages via incoming webhook.

```yaml
alerting:
  rocketchat:
    webhook-url: https://rocketchat.example.com/hooks/...
    channel: "#alerts"
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `webhook-url` | string | **yes** | Rocket.Chat incoming webhook URL. |
| `channel` | string | | Channel override. |

---

### Provider: sendgrid

Send email alerts via SendGrid.

```yaml
alerting:
  sendgrid:
    api-key: ${SENDGRID_KEY}
    from: gatus@example.com
    to: ops@example.com
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `api-key` | string | **yes** | SendGrid API key. |
| `from` | string | **yes** | Sender email address. |
| `to` | string | **yes** | Recipient email address(es), comma-separated. |
| `client` | Client | | [HTTP client settings](#client-configuration). |

---

### Provider: signal

Send Signal messages via the Signal REST API.

```yaml
alerting:
  signal:
    api-url: http://signal-cli-rest-api:8080
    number: "+15550001234"
    recipients:
      - "+15550005678"
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `api-url` | string | **yes** | Signal REST API base URL. |
| `number` | string | **yes** | Sender phone number (must be registered with the API). |
| `recipients` | []string | **yes** | Recipient phone numbers. |

---

### Provider: signl4

Send SIGNL4 push notifications.

```yaml
alerting:
  signl4:
    team-secret: ${SIGNL4_SECRET}
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `team-secret` | string | **yes** | SIGNL4 team secret. |

---

### Provider: slack

Send Slack messages via incoming webhook.

```yaml
alerting:
  slack:
    webhook-url: https://hooks.slack.com/services/...
    title: "Gatus"
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `webhook-url` | string | **yes** | Slack incoming webhook URL. |
| `title` | string | | Message title. |

---

### Provider: splunk

Send events to Splunk via HTTP Event Collector.

```yaml
alerting:
  splunk:
    hec-url: https://splunk.example.com:8088/services/collector/event
    hec-token: ${SPLUNK_HEC_TOKEN}
    source: gatus
    sourcetype: gatus:alert
    index: monitoring
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `hec-url` | string | **yes** | Splunk HEC endpoint URL. |
| `hec-token` | string | **yes** | Splunk HEC token. |
| `source` | string | | Event source field. |
| `sourcetype` | string | | Event sourcetype field. |
| `index` | string | | Target Splunk index. |

---

### Provider: squadcast

Trigger Squadcast incidents.

```yaml
alerting:
  squadcast:
    webhook-url: https://api.squadcast.com/v2/incidents/api/...
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `webhook-url` | string | **yes** | Squadcast webhook URL. |

---

### Provider: teams

Send Microsoft Teams messages via legacy incoming webhook connector.

```yaml
alerting:
  teams:
    webhook-url: https://outlook.office.com/webhook/...
    title: "Gatus"
```

> **Note:** Microsoft is retiring the legacy connector. Prefer [teams-workflows](#provider-teams-workflows).

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `webhook-url` | string | **yes** | Teams incoming webhook URL. |
| `title` | string | | Message title. |
| `client` | Client | | [HTTP client settings](#client-configuration). |

---

### Provider: teams-workflows

Send Microsoft Teams messages via the Power Automate Workflows connector.

```yaml
alerting:
  teams-workflows:
    webhook-url: https://prod-xx.westeurope.logic.azure.com/...
    title: "Gatus"
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `webhook-url` | string | **yes** | Teams Workflows webhook URL. |
| `title` | string | | Message title. |

---

### Provider: telegram

Send Telegram messages.

```yaml
alerting:
  telegram:
    token: ${TELEGRAM_TOKEN}
    id: "123456789"
    topic-id: "42"
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `token` | string | **yes** | Telegram bot token. |
| `id` | string | **yes** | Chat ID (user, group, or channel). |
| `topic-id` | string | | Topic (thread) ID for supergroups with topics enabled. |
| `api-url` | string | | Telegram Bot API URL (default `https://api.telegram.org`). |
| `client` | Client | | [HTTP client settings](#client-configuration). |

---

### Provider: twilio

Send SMS alerts via Twilio.

```yaml
alerting:
  twilio:
    sid: ${TWILIO_SID}
    token: ${TWILIO_TOKEN}
    from: "+15550001234"
    to: "+15550005678"
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `sid` | string | **yes** | Twilio account SID. |
| `token` | string | **yes** | Twilio auth token. |
| `from` | string | **yes** | Sender phone number. |
| `to` | string | **yes** | Recipient phone number. |
| `text-twilio-triggered` | string | | Custom SMS body for triggered alerts. |
| `text-twilio-resolved` | string | | Custom SMS body for resolved alerts. |

---

### Provider: vonage

Send SMS alerts via Vonage (formerly Nexmo).

```yaml
alerting:
  vonage:
    api-key: ${VONAGE_KEY}
    api-secret: ${VONAGE_SECRET}
    from: "Gatus"
    to:
      - "+15550001234"
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `api-key` | string | **yes** | Vonage API key. |
| `api-secret` | string | **yes** | Vonage API secret. |
| `from` | string | **yes** | Sender name or phone number. |
| `to` | []string | **yes** | Recipient phone numbers. |

---

### Provider: webex

Send Webex room messages.

```yaml
alerting:
  webex:
    webhook-url: https://webexapis.com/v1/webhooks/incoming/...
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `webhook-url` | string | **yes** | Webex Teams incoming webhook URL. |

---

### Provider: zapier

Trigger a Zapier webhook.

```yaml
alerting:
  zapier:
    webhook-url: https://hooks.zapier.com/hooks/catch/...
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `webhook-url` | string | **yes** | Zapier catch webhook URL. |

---

### Provider: zulip

Send Zulip stream messages.

```yaml
alerting:
  zulip:
    bot-email: gatus-bot@example.zulipchat.com
    bot-api-key: ${ZULIP_KEY}
    domain: example.zulipchat.com
    channel-id: "123456"
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `bot-email` | string | **yes** | Zulip bot email address. |
| `bot-api-key` | string | **yes** | Zulip bot API key. |
| `domain` | string | **yes** | Zulip server domain (e.g., `example.zulipchat.com`). |
| `channel-id` | string | **yes** | Zulip channel (stream) ID. |

---

## Storage

Controls where endpoint result history and events are persisted.

```yaml
storage:
  type: sqlite
  path: /data/gatus.db
  caching: true
  maximum-number-of-results: 200
  maximum-number-of-events: 100
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `type` | string | `memory` | Storage backend: `memory`, `sqlite`, or `postgres`. |
| `path` | string | `""` | Database path or connection string. Required for `sqlite` (file path) and `postgres` (DSN, e.g., `postgres://user:pass@host/db`). |
| `caching` | bool | `false` | Enable write-through caching to speed up dashboard reads. |
| `maximum-number-of-results` | int | `100` | Maximum number of evaluation results stored per endpoint. |
| `maximum-number-of-events` | int | `50` | Maximum number of status-change events stored per endpoint. |

---

## Web server

```yaml
web:
  address: 0.0.0.0
  port: 8080
  read-buffer-size: 8192
  tls:
    certificate-file: /certs/server.crt
    private-key-file: /certs/server.key
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `address` | string | `0.0.0.0` | Address to bind to. |
| `port` | int | `8080` | Port to listen on (0–65535). |
| `read-buffer-size` | int | `8192` | Per-connection read buffer size in bytes (minimum `4096`). |
| `tls` | TLS | — | Enable HTTPS. |

**TLS:**

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `certificate-file` | string | **yes** | Path to PEM-encoded server certificate. |
| `private-key-file` | string | **yes** | Path to PEM-encoded private key. |

---

## Security

Protect the Gatus dashboard with authentication.

### Basic authentication

```yaml
security:
  basic:
    username: admin
    password-bcrypt-base64: ${HASHED_PASSWORD}
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `username` | string | **yes** | Login username. |
| `password-bcrypt-base64` | string | **yes** | Base64-encoded bcrypt hash of the password. Generate with `htpasswd -bnBC 10 "" password \| tr -d ':\n' \| base64`. |

### OIDC authentication

```yaml
security:
  oidc:
    issuer-url: https://dev-12345.okta.com
    redirect-url: https://gatus.example.com/authorization-code/callback
    client-id: ${OIDC_CLIENT_ID}
    client-secret: ${OIDC_CLIENT_SECRET}
    scopes:
      - openid
      - profile
    allowed-subjects:
      - alice@example.com
    session-ttl: 8h
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `issuer-url` | string | **yes** | OIDC provider issuer URL. |
| `redirect-url` | string | **yes** | Callback URL; must end with `/authorization-code/callback`. |
| `client-id` | string | **yes** | OAuth2 client ID. |
| `client-secret` | string | **yes** | OAuth2 client secret. |
| `scopes` | []string | **yes** | Requested OIDC scopes (include `openid`). |
| `allowed-subjects` | []string | | Allowlist of subjects (e.g., email addresses) permitted to log in. Allows everyone when empty. |
| `session-ttl` | duration | `8h` | How long a login session remains valid. |

---

## UI

Customise the appearance of the dashboard.

```yaml
ui:
  title: "My Status Page"
  header: "My Status Page"
  logo: https://example.com/logo.png
  link: https://example.com
  dark-mode: true
  default-sort-by: name
  default-filter-by: none
  buttons:
    - name: Runbook
      link: https://wiki.example.com/runbook
  favicon:
    default: /favicon.ico
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `title` | string | `"Health Dashboard \| Gatus"` | HTML `<title>` of the page. |
| `description` | string | Gatus description | `<meta name="description">` content. |
| `header` | string | `"Gatus"` | Text displayed in the page header. |
| `dashboard-heading` | string | `"Health Dashboard"` | Main heading on the dashboard. |
| `dashboard-subheading` | string | `"Monitor the health of your services"` | Subheading below the main heading. |
| `logo` | string | `""` | URL or path to a logo image shown in the header. |
| `link` | string | `""` | URL the logo links to when clicked. |
| `custom-css` | string | `""` | Raw CSS injected into the page. |
| `dark-mode` | bool | `true` | Default to dark mode. |
| `default-sort-by` | string | `name` | Default sort order: `name`, `group`, or `health`. |
| `default-filter-by` | string | `none` | Default filter: `none`, `failing`, or `unstable`. |
| `login-subtitle` | string | `"System Monitoring Dashboard"` | Subtitle shown on the OIDC login page. |
| `buttons` | []Button | `[]` | Custom buttons displayed in the header. |
| `favicon` | Favicon | (built-in icons) | Custom favicon URLs. |

**Button:**

| Key | Type | Description |
|-----|------|-------------|
| `name` | string | Button label text. |
| `link` | string | Button destination URL. |

**Favicon:**

| Key | Type | Description |
|-----|------|-------------|
| `default` | string | Default favicon URL/path. |
| `size16x16` | string | 16×16 favicon URL/path. |
| `size32x32` | string | 32×32 favicon URL/path. |

---

## Maintenance

Define global maintenance windows during which endpoint failures are suppressed and not alerted.

```yaml
maintenance:
  enabled: true
  start: "23:00"
  duration: 2h
  timezone: America/New_York
  every:
    - Wednesday
    - Saturday
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `true` | Enable or disable the maintenance window. |
| `start` | string | **required** | Start time in `HH:MM` format (24-hour, 00:00–23:59). |
| `duration` | duration | **required** | Length of the maintenance window (e.g., `2h`, `30m`). |
| `timezone` | string | `UTC` | IANA timezone name (e.g., `America/New_York`, `Europe/Berlin`). |
| `every` | []string | `[]` | Days of the week the window applies: `Sunday`, `Monday`, `Tuesday`, `Wednesday`, `Thursday`, `Friday`, `Saturday`. Empty means every day. |

The same fields apply to per-endpoint `maintenance-windows` entries.

---

## Connectivity checker

Gatus can pause monitoring when the host loses internet connectivity.

```yaml
connectivity:
  checker:
    target: 8.8.8.8:53
    interval: 60s
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `checker.target` | string | **required** | DNS server address to probe (must end with `:53`). |
| `checker.interval` | duration | `60s` | How often to probe (minimum `5s`). |

---

## SSH tunneling

Route endpoint traffic through SSH tunnels. Each tunnel is defined as a named entry under `tunneling`.

```yaml
tunneling:
  jump-host:
    type: SSH
    host: bastion.example.com
    port: 22
    username: tunnel-user
    private-key: |
      -----BEGIN OPENSSH PRIVATE KEY-----
      ...
      -----END OPENSSH PRIVATE KEY-----

endpoints:
  - name: internal-db
    url: tcp://db.internal:5432
    client:
      tunnel: jump-host
    conditions:
      - "[CONNECTED] == true"
```

Each named tunnel accepts:

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `type` | string | **required** | Tunnel type. Only `SSH` is supported. |
| `host` | string | **required** | SSH server hostname. |
| `port` | int | `22` | SSH server port. |
| `username` | string | **required** | SSH username. |
| `private-key` | string | | PEM-encoded SSH private key. Provide either `private-key` or `password`. |
| `password` | string | | SSH password. |

Reference a tunnel from an endpoint's [client configuration](#client-configuration) via `client.tunnel: <name>`.

---

## Remote instances (alpha)

Aggregate results from multiple Gatus instances into a single dashboard.

```yaml
remote:
  instances:
    - endpoint-prefix: "us-east-"
      url: https://gatus-us-east.internal/api/v1
    - endpoint-prefix: "eu-west-"
      url: https://gatus-eu-west.internal/api/v1
  client:
    timeout: 5s
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `instances` | []Instance | `[]` | Remote Gatus instances to pull from. |
| `client` | Client | (defaults) | [HTTP client settings](#client-configuration) for remote calls. |

**Instance:**

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `endpoint-prefix` | string | | Prefix prepended to endpoint names from this instance to avoid collisions. |
| `url` | string | **yes** | Base URL of the remote Gatus API (e.g., `https://gatus.example.com`). |

---

## Announcements

Display banner announcements on the dashboard.

```yaml
announcements:
  - timestamp: 2024-03-15T10:00:00Z
    type: outage
    message: "Database cluster is being upgraded. Elevated latency expected."
  - timestamp: 2024-03-14T08:00:00Z
    type: operational
    message: "All systems returned to normal."
    archived: true
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `timestamp` | datetime | **required** | When the announcement was made (RFC 3339 / ISO 8601 UTC). |
| `type` | string | `none` | Announcement type controls the icon and colour: `outage`, `warning`, `information`, `operational`, or `none`. |
| `message` | string | **required** | Announcement text displayed to users. |
| `archived` | bool | `false` | When `true`, the announcement is shown in the history section rather than the active banner. |

---

## Environment variable substitution

Any string value in a configuration file can reference an environment variable:

```yaml
endpoints:
  - name: api
    url: https://${API_HOST}/health
    headers:
      Authorization: "Bearer ${API_TOKEN}"
```

| Syntax | Result |
|--------|--------|
| `${VAR}` | Replaced with the value of `VAR`. |
| `$$` | Replaced with a literal `$`. |

When a referenced variable is not set, the placeholder is left as-is (no error).
