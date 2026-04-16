all: build

fmt:
	gofmt -w .

vet: fmt
	go vet ./...

build: vet
	go build -o imgstat ./cmd/imgstat/

clean:
	rm -f imgstat

.PHONY: all fmt vet build clean
