# Statter

Lightweight status monitoring application for HTTP services.

```bash
go run main.go -config=example_conf.yml -serve=true
```

## TODO

1. Update service response data to:
	a. include aggregates of data over time.
	b. exclude private data like headers.
	c. be formatted as JSON.

2. Improve main loop to run without monitoring enabled.

3. Clean up and seperate logic into files (ie server and monotring seperated)

4. Provide additional error handling for server/monitroing errors.

5. Add tests
