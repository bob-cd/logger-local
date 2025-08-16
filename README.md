### Reference Bob logger

This is a simple logger which stores and serves logs from local files.

#### Requirements
- [Go](https://golang.org/dl/) 1.24+

#### Running
- `go build main.go` to compile the code and obtain a binary `main`.
- `./main` will start on port `8002` by default, set the env var `PORT` to change.

#### API

Here `{path}` represents `{pipeline-group}/{pipeline-name}/{run-id}/{artifact-name}`.

- `GET /bob_logs/runs/{runId}`: Streams log lines if the id exists.
- `PUT /bob_logs/runs/{runId}`: Ingests log data via PUT body for a given runId.
- `DELETE /bob_logs/runs/{runId}`: Deletes the logs for the runId.
- `GET /ping`: Responds with an `Ack`.
