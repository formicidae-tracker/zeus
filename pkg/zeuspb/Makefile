all: zeus_service.pb.go zeus_service_grpc.pb.go

%.pb.go: zeus_service.proto
	go generate

check:
	go test
	go vet

.PHONY: check
