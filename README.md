# Statter

Lightweight status monitoring application for HTTP services.

```bash
go run main.go -config=example_conf.yml -serve=true -monitor=true
```

## Roadmap

1. Update service response data to:  
	a. include aggregates of data over time.  
	b. exclude private data like headers.  
	c. be formatted as JSON.
2. Change flag defaults to monitor and serve by default (disable with "false") flags.
3. Provide additional error handling for server/monitoring errors.
4. Add tests.
