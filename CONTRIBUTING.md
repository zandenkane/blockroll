# contributing

Go 1.21+. SQLite3.

`make build` to compile. `make test` to run tests.

The privacy tier logic lives in pkg/privacy/. The HTTP handlers are in pkg/handler/. If you want to add a new trust tier level, start with the Privacy type in pkg/privacy/privacy.go.
