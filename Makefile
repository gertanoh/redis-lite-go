# ################## Helpers ######################

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'


.PHONY: confirm
confirm:
	@echo -n "Are you sure? [y/N] " && read ans && [ $${ans:-N} = y ]

# ################## Dev ######################

.PHONY: redis
redis:
	@go run .
	
# ################## QA ######################
.PHONY: audit
audit:
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	# staticcheck ./...
	@echo 'Running tests'
	go test -race -vet=off ./...

	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify


# ################## Build ######################
linker_flags = '-s -w'

## build: build the application
.PHONY: build
build:
	@echo 'Building redis-lite'
	go build -ldflags=${linker_flags} -o=./redis-lite
