BINARY_NAME=hf-scraper
MAIN_PATH=./cmd/daemon
DIST_DIR=dist
PLATFORMS=linux/amd64 windows/amd64 darwin/amd64 darwin/arm64

release: clean
	mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*}; \
		GOARCH=$${platform#*/}; \
		OUT=$(BINARY_NAME)-$$GOOS-$$GOARCH; \
		if [ $$GOOS = "windows" ]; then OUT=$$OUT.exe; fi; \
		echo "Building $$GOOS/$$GOARCH..."; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build -o $(DIST_DIR)/$$OUT $(MAIN_PATH); \
	done

clean:
	rm -rf $(DIST_DIR)
