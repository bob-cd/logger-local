### Reference Bob logger

This is a simple logger which stores and serves logs from local files.

#### Requirements

- [Go](https://golang.org/dl/) 1.24+

#### Running

- `go build main.go` to compile the code and obtain a binary `main`.
- `./main` will start on port `8002` by default, set the env var `PORT` to change.

#### API

Here `{path}` represents `{pipeline-group}/{pipeline-name}/{run-id}`.

- `GET /bob_logs/{path}`: Sends log lines if the run exists, send follow=true to stream live changes.
- `PUT /bob_logs/{path}`: Ingests log data via PUT body for a given run.
- `DELETE /bob_logs/{path}`: Deletes the logs for the run.
- `GET /ping`: Responds with an `Ack`.
