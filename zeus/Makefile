all: zeus check

.PHONY: check clean

zeus: *.go ../*.go ../zeuspb/*.go ../zeuspb/*.proto
	go generate
	go build

check:
	go test
	go vet

clean:
	rm -f zeus
