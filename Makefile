.PHONY: tests mocks

ifeq ($(OS),Windows_NT)
    EXT := .exe
else
    EXT :=
endif

run-toml:
	go run . ./examples/example.toml


run-json:
	go run . ./examples/example.json

run-yaml:
	go run . ./examples/example.yaml

run-yml:
	go run . ./examples/example.yml


mocks:
	go build -o ./tests/mocks/build/fake_process$(EXT) ./tests/mocks/fake_process.go

unit-test:
	go test ./internal -v -timeout 2s

tests: mocks unit-tests
	go test ./tests -timeout 2s

unit-test-verbose: mocks
	go test -cpuprofile cpu.prof -memprofile mem.prof -bench . ./internal -v -timeout 30s

tests-verbose: unit-test-verbose
	go test -cpuprofile cpu.prof -memprofile mem.prof -bench . ./tests -v -timeout 30s