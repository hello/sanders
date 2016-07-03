PACKAGES = $(shell go list ./... | grep -v '/vendor/')
format:
	@echo "--> Running go fmt"
	@go fmt $(PACKAGES)
