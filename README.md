# Statter

Lightweight status monitoring application for HTTP services.

## Usage

Statter should be run via command line. To start a Statter server with
monitoring enabled simply run:

```bash
go run main.go -config=conf.yaml
```

### Flags

Name | Description | Default
-----|-------------|--------
`config` | yaml config file | conf.yaml
`monitor` | run monitoring | true
`serve` | run statter server | true

## Configuration

Statter can be configured using a YAML configuration file. The following
attributes are available:

**database**

A path to the prefered database file location. The database defaults to a
`statter.db` file in the same location as the Statter executable.

**port**

The port at which the Statter API will be available. This defaults to 8080.

**interval**

The length of time between each monitoring attempt on a service. This value is
an integer representing a length of time in seconds. It defaults to 60 seconds.

**services**

Contains the list of services that should be monitored/served by Statter.
Each configured service should always have a `name` and `url` attribute defined.

In addition, you can configure `method`, `body` and/or `headers` attributes for
each service.

### Example

This example can be used as a template for Statter configuration files.

```yaml
database: statter.db
port: 8080
interval: 5
services:
    - name: test_one
      label: "Test One"
      description: "Test One Service"
      url: https://one.test.com
      method: GET
      body: '{"value": "one"}'
      headers:
          - name: Content-Type
            value: application/json
          - name: Origin
            value: http://example.com
    - name: test_two
      label: "Test Two"
      description: "Test Two Service"
      url: https://two.test.com
      method: GET
      body: '{"value": "two"}'
      headers:
          - name: Content-Type
            value: application/json
          - name: Origin
            value: http://example.com
```

## JSON API

The following endpoints are served by the JSON API:

* GET: `http://localhost:8080/services/`
* GET: `http://localhost:8080/responses/`

Both endpoints can be filtered using the service `name` via GET parameters:

`http://localhost:8080/services/?name=service_name`

## Roadmap

1. Add handling of non existent domains (missing domain schema).
2. Add request time to DB for each monitor task.
3. Improve error handling for server and monitoring.
4. Add tests.
