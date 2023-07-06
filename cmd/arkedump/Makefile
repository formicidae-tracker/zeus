all: arkedump check

.PHONY: check clean

arkedump: *.go
	go generate
	go build

check:
	go test
	go vet

clean:
	rm -f arkedump
