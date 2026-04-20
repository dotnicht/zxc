# Test Report

Date: 2026-04-20

## Summary

- Full test suite command: `go test ./...`
- Result: passed
- Total integration runtime for `zxc/test`: `279.380s`

## Full Suite Result

Successful packages from the full run:

- `zxc/internal/authz`
- `zxc/internal/config`
- `zxc/internal/jobs`
- `zxc/internal/middleware`
- `zxc/internal/request`
- `zxc/internal/service`
- `zxc/internal/storage`
- `zxc/internal/workflow`
- `zxc/test`

Packages without test files in the same run:

- `zxc/api/payload`
- `zxc/api/release`
- `zxc/api/target`
- `zxc/api/tenant`
- `zxc/api/user`
- `zxc/cmd/client`
- `zxc/cmd/migrator`
- `zxc/cmd/server`
- `zxc/cmd/storageui`
- `zxc/cmd/webhook`
- `zxc/cmd/worker`
- `zxc/internal/consts`
- `zxc/internal/db`
- `zxc/internal/deployer`
- `zxc/internal/models`
- `zxc/internal/queue`

## New Unit Test Coverage Added

New tests were added in these files:

- `internal/config/config_test.go`
- `internal/jobs/deploy_test.go`
- `internal/middleware/context_test.go`
- `internal/request/handler_test.go`
- `internal/service/ids_test.go`
- `internal/storage/client_test.go`
- `internal/workflow/runner_test.go`
- `internal/workflow/store_test.go`

Focused coverage command:

- `go test -cover ./internal/middleware ./internal/service ./internal/jobs ./internal/storage ./internal/config ./internal/request ./internal/workflow`

Focused coverage results:

- `zxc/internal/middleware`: `29.4%`
- `zxc/internal/service`: `1.3%`
- `zxc/internal/jobs`: `11.9%`
- `zxc/internal/storage`: `34.8%`
- `zxc/internal/config`: `35.3%`
- `zxc/internal/request`: `42.1%`
- `zxc/internal/workflow`: `47.7%`

## Notes

- The full `zxc/test` integration suite booted the Docker compose stack, created a tenant, target, payload, and release, then completed the deploy flow to `alive`.
- `internal/db/postgres.go` is currently back on plain `AutoMigrate` without explicit table rename steps.
