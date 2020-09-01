# digdirator

Digdirator is a Kubernetes cluster operator for automated registration and lifecycle management of ID-porten clients.

## CRD

The operator introduces a new Kind `IDPortenClient` (shortname `idportenclient`), and acts upon changes to resources of this kind.

See the spec in [config/crd/nais.io_idportenclients.yaml](config/crd/nais.io_idportenclients.yaml) for details.

An example resource is available in [config/samples/idportenclient.yaml](config/samples/idportenclient.yaml).

## Lifecycle

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
  auth:
    jwk: ""
  endpoint: ""
cluster-name: local
```

Then, assuming you have a Kubernetes cluster running locally (e.g. using [minikube](https://github.com/kubernetes/minikube)):

```shell script
ulimit -n 4096  # for controller-gen
make run
kubectl apply -f ./config/samples/idportenclient.yaml
```
