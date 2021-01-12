#
# Runs the test suite for the module.
#
test:
	@ \
	go test -timeout 5s ./...

#
# Runs the test suite for the whole module, spitting out the the code coverage report to find gaps.
#
coverage:
	@ \
	go test -coverprofile=coverage.out -timeout 5s ./... && \
	go tool cover -func=coverage.out && \
	rm coverage.out
