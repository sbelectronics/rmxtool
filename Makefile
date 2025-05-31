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

.PHONT: clean
clean:
	rm -f $(TOOL)
