.PHONY: tests

run-toml:
	go run . ./examples/example.toml


run-json:
	go run . ./examples/example.json

run-yaml:
	go run . ./examples/example.yaml

run-yml:
	go run . ./examples/example.yml

tests:
	go test -cpuprofile cpu.prof -memprofile mem.prof -bench . ./tests -v