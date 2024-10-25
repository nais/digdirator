KUBEBUILDER_VERSION := 3.9.0
K8S_VERSION         := 1.26.1
arch                := amd64
os                  := $(shell uname -s | tr '[:upper:]' '[:lower:]')

# Run tests excluding integration tests
test: fmt vet
	go test ./... -coverprofile cover.out -short

# Run against the configured Kubernetes cluster in ~/.kube/config
run: fmt vet
	go run cmd/digdirator/main.go

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

kubebuilder:
	test -d /usr/local/kubebuilder || (sudo mkdir -p /usr/local/kubebuilder && sudo chown "${USER}" /usr/local/kubebuilder)
	wget -qO - "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-${K8S_VERSION}-$(os)-$(arch).tar.gz" | tar -xz -C /usr/local
	wget -qO /usr/local/kubebuilder/bin/kubebuilder https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${KUBEBUILDER_VERSION}/kubebuilder_$(os)_$(arch)
	chmod +x /usr/local/kubebuilder/bin/*

pem2jwk:
	go build -o bin/pem2jwk cmd/pem2jwk/*.go

install:
	kubectl apply -f https://raw.githubusercontent.com/nais/liberator/main/config/crd/bases/nais.io_idportenclients.yaml
	kubectl apply -f https://raw.githubusercontent.com/nais/liberator/main/config/crd/bases/nais.io_maskinportenclients.yaml
	kubectl apply -f ./hack/resources/

sample:
	kubectl apply -f ./config/samples/idportenclient.yaml
	kubectl apply -f ./config/samples/maskinportenclient.yaml
