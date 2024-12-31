.PHONY: tests

run-toml:
	go run . ./examples/example.toml


run-json:
	go run . ./examples/example.json

run-yaml:
	go run . ./examples/example.yaml

run-yml:
	go run . ./examples/example.yml

unit-test:
	go test ./internal -v -timeout 2s

tests: unit-tests
	go test ./tests -timeout 2s

unit-test-verbose:
	go test -cpuprofile cpu.prof -memprofile mem.prof -bench . ./internal -v -timeout 10s

tests-verbose: unit-test-verbose
	go test -cpuprofile cpu.prof -memprofile mem.prof -bench . ./tests -v -timeout 10s