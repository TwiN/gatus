# Config using environment variables with SQLite storage example

This example demonstrates how to run Gatus using environment variables for configuration, with SQLite as the storage backend.

## Testing Instructions

Both options use the example configs.

### With plain configuration

```bash
docker-compose -f compose-plain.yaml up -d
```

### With base64-encoded configuration

```bash
docker-compose -f compose-base64.yaml up -d
```

