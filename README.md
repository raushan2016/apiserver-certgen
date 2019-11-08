# apiserver-certgen
This is a tool based on https://github.com/kubernetes-sigs/apiserver-builder-alpha to generated self-signed certificates for configuring APIService in kubernetes
More details https://kubernetes.io/docs/tasks/access-kubernetes-api/setup-extension-api-server/

## Build
`make build` 

## To generate certificates 
./bin/apiserver-certgen --name <kubernetes service name> --namespace <service name namespace>

`Certificates will be generated in the path ./config/certificates`