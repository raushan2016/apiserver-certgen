export GO111MODULE=on
# Build tool
build:
	go build -o bin/certgen main.go

gen:
	./bin/certgen --name mir-instance-controller-apiserver --namespace mir-instance
