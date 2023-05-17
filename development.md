# Development

You'll need [Go installed][go] to be able to contribute to `catalog-importer`.

You can run all of the tests using `go test`:

```
go test ./...
```

To build the binary, run `make`: this will place a built version of the tool in
the `bin` directory. If you work for incident.io and have a local instance of 
the app running, then you can point it to your local environment using the
`--api-key` flag, or an environment variable:

```
export INCIDENT_ENDPOINT="http://localhost:3001/api/public"
```

[go]: https://go.dev/doc/install
