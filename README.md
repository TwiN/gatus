# gatus

A service health dashboard in Go


config should look something like

```yaml
services:
  - name: twinnation
    url: https://twinnation.org/actuator/health
    interval: 10
    failure-threshold: 3
    conditions:
      - "$STATUS == 200"
      - "IP == 200"
```