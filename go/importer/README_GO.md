# Go Port (importer)

This is the Go port of the Scala data importer (library + application) as a single module.

## Status
- Skeleton app and pipeline are in place.
- XSD generation is wired (uses `xgen`).
- Iceberg integration and table mapping are placeholders and must be implemented.

## Generate XSD Models
Run once to generate Go structs from XSD:

```
go generate ./...
```

If generated field names differ, update:
- `internal/whmovement/keys.go`

## Build

```
go build ./cmd/whmovement
```

## Run
Set environment variables (same as Scala):
- `AWS_REGION`
- `SOC_COD`
- `DATABASE_NAME`
- `DATABASE_FOLDER` (default: `/data/database/`)
- `S3_BUCKET_NAME`
- `S3_SOURCE_FOLDER`
- `S3_ARCHIVE_FOLDER`
- `S3_ARCHIVE_INVALID_FOLDER`
- `BATCH_SIZE` (default: 100)
- `BATCH_WEIGHT_IN_BYTE` (default: 5242880)
- `INTERVAL_IN_MINUTE` (default: 5)
- `SQS_QUEUE`
- `REINIT` (default: false)
- `DELAY_IN_MINUTE` (default: 1)

Then:

```
./whmovement
```

## Parity Gaps
- Implement Iceberg Glue catalog, schema setup, row-delta writes.
- Port table schemas and record creation for all `Mgt*` tables.
- Add OpenTelemetry tracing + metrics (currently logs only).

