# digdirator

Digdirator is a Kubernetes cluster operator for automated registration and lifecycle management of ID-porten clients.

## CRD

The operator introduces two new Kind´s:  
`IDPortenClient` (shortname `idportenclient`) and `MaskinportenClient` (shortname `maskinportenclient`), and acts upon changes to resources of its kind.

See the specs:
[config/crd/nais.io_idportenclients.yaml](config/crd/nais.io_idportenclients.yaml) and
[config/crd/nais.io_maskinportenclients.yaml](config/crd/nais.io_maskinportenclients.yaml) for details.

An example of resources is available in [config/samples/idportenclient.yaml](config/samples/idportenclient.yaml) and [config/samples/maskinportenclient.yaml](config/samples/maskinportenclient.yaml).

## Lifecycle

![overview][overview]

[overview]: ./docs/sequence.png "Sequence diagram"

## Development

### Installation

```shell script
make install
```

### Configuration

Set up the required environment variables as per the [config](./pkg/config/config.go) 

```yaml
# ./digdirator.yaml

digdir:
  admin:
    base-url: "base URL for digdir admin API"
  auth:
    audience: "audience for JWT assertion"
    client-id: "client ID / issuer for JWT assertion"
    scopes: "space separated list of scopes for JWT assertion"
  idporten:
    base-url: "base URL endpoint for idporten API"
    cert-chain-path: "Path to PEM file containing certificate chain for authenticating to DigDir."
    kms-key-path: "example: projects/my-project/locations/us-east1/keyRings/my-key-ring/cryptoKeys/my-key/cryptoKeyVersions/123"
  maskinporten:
    base-url: "base URL endpoint for maskinporten API"
    cert-chain-path: "Path to PEM file containing certificate chain for authenticating to DigDir."
    kms-key-path: "example: projects/my-project/locations/us-east1/keyRings/my-key-ring/cryptoKeys/my-key/cryptoKeyVersions/123"
cluster-name: local
development-mode: true
```

Then, assuming you have a Kubernetes cluster running locally (e.g. using [minikube](https://github.com/kubernetes/minikube)):

```shell script
ulimit -n 4096  # for controller-gen
make run
kubectl apply -f ./config/samples/idportenclient.yaml
```
