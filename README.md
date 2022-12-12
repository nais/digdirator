# digdirator

Digdirator is a Kubernetes cluster operator for automated registration and lifecycle management of ID-porten and
Maskinporten clients (integrations) with feature Maskinporten Scopes (APIS).

## CRD

The operator introduces two new Kinds:  
`IDPortenClient` (shortname `idportenclient`) and `MaskinportenClient` (shortname `maskinportenclient`), and acts upon
changes to these.

See the specs in [liberator](https://github.com/nais/liberator) for details:

- [config/crd/nais.io_idportenclients.yaml](https://github.com/nais/liberator/blob/main/config/crd/bases/nais.io_idportenclients.yaml)
  and
- [config/crd/nais.io_maskinportenclients.yaml](https://github.com/nais/liberator/blob/main/config/crd/bases/nais.io_maskinportenclients.yaml)
  for details.

An example of resources is available in [config/samples/idportenclient.yaml](config/samples/idportenclient.yaml)
and [config/samples/maskinportenclient.yaml](config/samples/maskinportenclient.yaml).

## Lifecycle

![overview][overview]

[overview]: ./docs/sequence.png "Sequence diagram"

## Development

### Installation

```shell script
kubectl apply -f <path to CRDs from liberator>
```

### AuthN & AuthZ

Login to google and authenticate as a service account

```shell script
gcloud auth login --update-adc
```

### Configuration

Set up the required environment variables as per the [config](./pkg/config/config.go)

```yaml
# ./digdirator.yaml

cluster-name: local
development-mode: true
features:
  maskinporten: true
digdir:
  admin:
    base-url: "base URL for digdir admin API"
  idporten:
    client-id: "Secret path to the client ID / issuer for JWT assertion"
    cert-cain: "Secret path in Google Secret Manager to PEM file containing public certificate chain for authenticating to DigDir."
    kms:
      key-path: "KMS resource path to sign JWT assertion"
    scopes: "space separated list of scopes for JWT assertion"
    well-known-url: "URL to ID-porten well-known discovery metadata document."
  maskinporten:
    client-id: "Secret path to the client ID / issuer for JWT assertion"
    cert-cain: "Secret path in Google Secret Manager to PEM file containing public certificate chain for authenticating to DigDir."
    kms:
      key-path: "KMS resource path to sign JWT assertion"
    scopes: "space separated list of scopes for JWT assertion"
    well-known-url: "URL to Maskinporten well-known discovery metadata document."
```

Then, assuming you have a Kubernetes cluster running locally (e.g.
using [minikube](https://github.com/kubernetes/minikube)):

```shell script
ulimit -n 4096  # for controller-gen
make run
kubectl apply -f ./config/samples/idportenclient.yaml
```