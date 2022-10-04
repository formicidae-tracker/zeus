all: arke-zeus-config check

.PHONY: check clean

arke-zeus-config: *.go
	go generate
	go build

check:
	go test
	go vet

clean:
	rm -f arke-zeus-config
