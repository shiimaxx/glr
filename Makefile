.PHONY: lint
lint:
	@golint -set_exit_status ./...

.PHONY: test
test:
	@go test ./...

