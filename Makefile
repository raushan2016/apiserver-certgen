export GO111MODULE=on
# Build tool
build:
	go build -o bin/apiserver-certgen main.go

gen:
	./bin/apiserver-certgen --name mir-instance-controller-apiserver --namespace mir-instance
