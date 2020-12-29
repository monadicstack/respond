test:
	@ \
	go test ./...

coverage:
	@ \
	go test -cover -timeout 5s ./...

coverage-report:
	@ \
	go test -coverprofile=coverage.out && \
	go tool cover -func=coverage.out && \
	rm coverage.out
