# Chat Trace Viewer

Web tool for inspecting the history of a single chat from VictoriaLogs records.

This project is intentionally specialized: it was built for local diagnostics and internal workflows around a specific log format. It is not a general-purpose trace viewer or a polished external SaaS product, but rather a practical engineering utility for manually investigating individual cases.

The application:

- starts a local HTTP server;
- fetches raw records from VictoriaLogs;
- normalizes them into a timeline;
- shows anomalies and diagnostic summary data in the browser.

## Requirements

- Go `1.26+`
- access to VictoriaLogs

## Configuration

Base defaults live in `config/app.env`.
Local overrides can be placed in `config/app.local.env`, using `config/app.local.env.example` as a starting point.

Main variables:

- `APP_ADDR` - HTTP server address, defaults to `127.0.0.1:8080`
- `VICTORIALOGS_BASE_URL` - VictoriaLogs base URL
- `VICTORIALOGS_ACCOUNT_ID` - `AccountID` header value
- `VICTORIALOGS_PROJECT_ID` - `ProjectID` header value
- `VICTORIALOGS_USERNAME` - basic auth username, if needed
- `VICTORIALOGS_PASSWORD` - basic auth password, if needed
- `TRACE_DEFAULT_LOOKBACK` - default search window, for example `30d`
- `TRACE_MAX_LOG_LINES` - limit for lines fetched from the source
- `TRACE_MAX_RAW_LINES` - limit for lines returned to the UI

Example:

```env
APP_ADDR=127.0.0.1:8080
VICTORIALOGS_BASE_URL=http://localhost:9428
VICTORIALOGS_ACCOUNT_ID=0
VICTORIALOGS_PROJECT_ID=11
VICTORIALOGS_USERNAME=
VICTORIALOGS_PASSWORD=
TRACE_DEFAULT_LOOKBACK=30d
TRACE_MAX_LOG_LINES=500
TRACE_MAX_RAW_LINES=500
```

## Local Run

```bash
cp config/app.local.env.example config/app.local.env
make run
```

Then open `http://127.0.0.1:8080`.

## Verification

```bash
make test
make build
```

## Structure

- `cmd/chat-trace-viewer` - application entry point
- `internal/httpapi` - HTTP API and static web serving
- `internal/service` - chat trace assembly flow
- `internal/parser` - raw log parsing
- `internal/normalizer` - event normalization
- `internal/timeline` - timeline and anomaly building
- `internal/victorialogs` - VictoriaLogs client
- `web` - simple frontend without a build step
