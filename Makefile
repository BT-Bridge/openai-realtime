.PHONY: cli playground

cli:
	@$(MAKE) -C _examples cmd EXAMPLE=cli | tee _examples/cli/cli.output

playground:
	@$(MAKE) -C _examples cmd EXAMPLE=playground

.PHONY: clean
clean:
	@rm -f bin/*

.PHONY: test
test:
	@go test -v -coverprofile=coverage.out ./... 2>&1 | tee results.test

.PHONY: tidy
tidy:
	@go mod tidy

.PHONY: lint
lint: tidy
	@echo "Running linters..."
	@gofmt -s -w .
	@golangci-lint run --fix