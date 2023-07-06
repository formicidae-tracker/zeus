all: arke-change-device-id check

.PHONY: check clean

arke-change-device-id: *.go
	go generate
	go build

check:
	go test
	go vet

clean:
	rm -f arke-change-device-id
