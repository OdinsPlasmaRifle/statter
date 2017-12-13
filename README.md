# Statter

Lightweight status monitoring application for HTTP services.

## Usage

Statter should be run via command line. Running a Statter server with automatic
monitoring and an API server enabled is as simple as:

```bash
go run main.go -config=conf.yaml
```

### Flags

Name | Description | Default
-----|-------------|--------
`config` | yaml config file | conf.yaml
`monitor` | run monitoring | true
`serve` | run statter server | true
`port` | statter server port | 8080

## Configuration

Statter can be configured using a YAML configuration file. The following
attributes are available:

**databaseFile**

A path to the preferred database file. The database defaults to a `statter.db`
file in the same location as the Statter executable.

Statter uses sqlite, so the database is very portable and easy to access oustide
of Statter itself.

**interval**

The length of time between each monitoring attempt on a service. This value is
an integer representing a length of time in minutes. It defaults to 5 minutes.

**services**

Used to set the list of services which should be monitored/served by Statter.
Each configured service should always have a `name` and `url` attribute defined.

In addition, you can configure `method`, `body` and/or `headers` attributes for
each service.

### Example

This example can be used as a template for Statter configuration files.

```yaml
databaseFile: statter.db
interval: 5
services:
    - name: test_one
      url: https://one.test.com
      method: GET
      body: '{"value": "one"}'
      headers:
          - name: Content-Type
            value: application/json
          - name: Origin
            value: http://example.com
    - name: test_two
      url: https://two.test.com
      method: GET
      body: '{"value": "two"}'
      headers:
          - name: Content-Type
            value: application/json
          - name: Origin
            value: http://example.com
```

## Roadmap

1. Update service (API) response data to:
	a. Include aggregates of data over time.
	b. Exclude private data like headers.
2. Add request time  to DB for each monitor task.
3. Improve error handling for server and monitoring.
4. Add tests.
