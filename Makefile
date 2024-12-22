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
	go test ./internal -v

tests: unit-tests
	go test ./tests

unit-test-verbose:
	go test -cpuprofile cpu.prof -memprofile mem.prof -bench . ./internal -v

tests-verbose: unit-test-verbose
	go test -cpuprofile cpu.prof -memprofile mem.prof -bench . ./tests -v