TOOL=build/_output/rmxtool

all: build

.PHONY: build
build:
	go build -o $(TOOL) ./cmd

.PHONY: go-format
go-format:
	go fmt $(shell sh -c "go list ./...")

.PHONY: reset
reset: build
	cp test.save test.img
	$(TOOL) wipe

.PHONY: test
test:
	go test ./...

.PHONY: release
release:
	GOOS=linux GOARCH=amd64 go build -o release/linux/amd64/rmxtool ./cmd
	GOOS=linux GOARCH=arm64 go build -o release/linux/arm64/rmxtool ./cmd
	GOOS=windows GOARCH=amd64 go build -o release/windows/amd64/rmxtool.exe ./cmd

.PHONT: clean
clean:
	rm -f $(TOOL)
