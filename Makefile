TEMPL := go run github.com/a-h/templ/cmd/templ@v0.3.1070

.PHONY: check test examples lint generate

check: test examples

test:
	go vet ./...
	go test -race ./...

examples:
	cd examples && go vet ./... && go build ./...

lint:
	golangci-lint run ./...

generate:
	cd examples && $(TEMPL) generate
