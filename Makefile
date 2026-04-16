all: build

fmt:
	gofmt -w .

vet: fmt
	go vet ./...

build: vet
	go build .

clean:
	rm -f imgstat

.PHONY: all fmt vet build clean
