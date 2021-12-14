## Usage
```console
docker-compose up
```
Once you've done the above, you should be able to access the Grafana dashboard at `http://localhost:3000`.

## Queries
Gatus uses Prometheus counters.

Total results per minute:
```
sum(rate(gatus_results_total[5m])*60) by (key)
```

Total successful results per minute:
```
sum(rate(gatus_results_total{success="true"}[5m])*60) by (key)
```

Total unsuccessful results per minute:
```
sum(rate(gatus_results_total{success="true"}[5m])*60) by (key)
```