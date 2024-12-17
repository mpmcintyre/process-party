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
	go test -parallel=4 ./tests

tests-profile:
	go test -cpuprofile cpu.prof -memprofile mem.prof -parallel=4  -bench . ./tests

tests-verbose:
	go test -cpuprofile cpu.prof -memprofile mem.prof -bench . ./tests -v