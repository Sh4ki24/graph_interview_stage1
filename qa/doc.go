// Package qa contains smoke/integration tests that call the running API.
//
// The tests are guarded by the "qa" build tag so that a plain `go test ./...`
// does not require a live server. Run them against a running stack with:
//
//	go test -tags qa ./qa/... -count=1 -v
//
// or via `make test-qa` (after `make compose-up`).
package qa