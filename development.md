# Development

You'll need [Go installed][go] to be able to contribute to `catalog-importer`.

You can run all of the tests using `go test`:

```
go test ./...
```

To build the binary, run `make`: this will place a built version of the tool in
the `bin` directory. If you work for incident.io and have a local instance of
the app running, then you can point it to your local environment using the
following environment variable:

```
export INCIDENT_API_KEY="inc_development_h3gHe9hegh39h3gh39He9hEgh3gh3gHe9h3gh39h3gh39hE9hegh3gh3Ghe9hEG"
export INCIDENT_ENDPOINT="http://localhost:3001/api/public"
```

To run it with a debugger attached you can edit the settings in `launch.json`

[go]: https://go.dev/doc/install
