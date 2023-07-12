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

Sample resources:

- [config/samples/idportenclient.yaml](config/samples/idportenclient.yaml)
- [config/samples/maskinportenclient.yaml](config/samples/maskinportenclient.yaml).

## Lifecycle

![overview][overview]

[overview]: ./docs/sequence.png "Sequence diagram"

## Usage

### Installation

```shell script
make install
```

### DigDir Setup

See the documentation over at DigDir for acquiring clients with the required scopes to access the self-service APIs:

- <https://docs.digdir.no/docs/idporten/oidc/oidc_api_admin_maskinporten#hvordan-f%C3%A5-tilgang->
- <https://docs.digdir.no/docs/idporten/oidc/oidc_api_admin#hvordan-f%C3%A5-tilgang->

Digdirator uses two separate clients (and thus also certificates) for administration of ID-porten and Maskinporten clients.

### Google Cloud Platform Setup

Digdirator makes use of a few Google Cloud Platform products; Secret Manager and Cloud Key Management Service (KMS).
We use these to store secrets and business certificates, respectively.
These are needed for authenticating the DigDir client with Maskinporten using the JWT-bearer authorization grant.

You should set up [Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity) for your GKE cluster.

Digdirator needs a Google IAM Service Account to access the GCP resources.
With Workload Identity, this should work automagically as we use Google's libraries that should automatically pick up the Kubernetes Service Account tokens and perform the necessary exchanges.

#### Secret Manager

Digdirator needs two secrets for each client:

1. contains the _client ID_ for the DigDir client
2. contains the _certificate chain_ (in PEM format) for the business certificate used to sign the client assertions when acquiring Maskinporten tokens

The Google Service Account needs the following IAM role for both secrets:

- Secret Manager Secret Accessor (`roles/secretmanager.secretAccessor`)

#### Cloud KMS

The Google Service Account needs the following IAM role for each _key_ in Cloud KMS:

- Cloud KMS CryptoKey Signer/Verifier (`roles/cloudkms.signerVerifier`)

Follow [Google's documentation for importing keys](https://cloud.google.com/kms/docs/importing-a-key).

The private key for business certificate should be imported with the purpose set to `ASYMMETRIC_SIGN`, and the algorithm set to one of the `RSASSA-PKCS1 v1_5` variants.

### Configuration

Digdirator can be configured using either command-line flags or equivalent environment variables (i.e. `-`, `.` -> `_`
and uppercase), with `DIGDIRATOR_` as prefix. E.g.:

```text
digdir.admin.base-url -> DIGDIRATOR_ADMIN_BASE_URL
```

The following flags are available:

```shell
--cluster-name string                               The cluster in which this application should run.
--development-mode string                           Toggle for development mode. (default "false")
--digdir.admin.base-url string                      Base URL endpoint for interacting with Digdir Client Registration API
--digdir.common.access-token-lifetime int           Default lifetime (in seconds) for access tokens for all clients. (default 3600)
--digdir.common.client-name string                  Default name for all provisioned clients. Appears in the login prompt for ID-porten. (default "ARBEIDS- OG VELFERDSETATEN")
--digdir.common.client-uri string                   Default client URI for all provisioned clients. Appears in the back-button for the login prompt for ID-porten. (default "https://www.nav.no")
--digdir.common.session-lifetime int                Default lifetime (in seconds) for sessions (authorization and refresh token lifetime) for all clients. (default 7200)
--digdir.idporten.cert-chain string                 Secret path in Google Secret Manager to PEM file containing certificate chain for authenticating to DigDir.
--digdir.idporten.client-id string                  Client ID / issuer for JWT assertion when authenticating to DigDir.
--digdir.idporten.kms.key-path string               IDPorten KMS resource path used to sign JWT assertion when authenticating to DigDir.
--digdir.idporten.scopes string                     List of scopes for JWT assertion when authenticating to DigDir with IDporten.
--digdir.idporten.well-known-url string             URL to ID-porten well-known discovery metadata document.
--digdir.maskinporten.cert-chain string             Secret path in Google Secret Manager to PEM file containing certificate chain for authenticating to DigDir.
--digdir.maskinporten.client-id string              Client ID / issuer for JWT assertion when authenticating to DigDir.
--digdir.maskinporten.default.client-scope string   Default scope for provisioned Maskinporten clients, if none specified in spec. (default "nav:test/api")
--digdir.maskinporten.default.scope-prefix string   Default scope prefix for provisioned Maskinporten scopes. (default "nav")
--digdir.maskinporten.kms.key-path string           Maskinporten Google KmsConfig resource path used to sign JWT assertion when authenticating to DigDir.
--digdir.maskinporten.scopes string                 List of scopes for JWT assertion when authenticating to DigDir with Maskinporten.
--digdir.maskinporten.well-known-url string         URL to Maskinporten well-known discovery metadata document.
--features.maskinporten                             Feature toggle for maskinporten
--metrics-address string                            The address the metric endpoint binds to. (default ":8080")
```

At minimum, the following configuration must be provided:

- `cluster-name`
- `digdir.admin.base-url`
- `digdir.idporten.cert-chain`
- `digdir.idporten.client-id`
- `digdir.idporten.kms.key-path`
- `digdir.idporten.scopes`
- `digdir.idporten.well-known-url`
- `digdir.maskinporten.cert-chain`
- `digdir.maskinporten.client-id`
- `digdir.maskinporten.kms.key-path`
- `digdir.maskinporten.scopes`
- `digdir.maskinporten.well-known-url`

Equivalently, one can specify these properties using JSON, TOML, YAML, HCL, envfile and Java properties config files.
Digdirator looks for a file named `digdirator.<ext>` in the directories [`.`, `/etc/`].

Example configuration in YAML:

```yaml
# ./digdirator.yaml

cluster-name: local
development-mode: true
features:
  maskinporten: true
digdir:
  admin:
  base-url: "https://integrasjon-ver2.difi.no"
  idporten:
    client-id: "projects/<project>/secrets/<idporten-client-id>/versions/<version>"
    cert-chain: "projects/<project>/secrets/<idporten-cert-chain>/versions/<version>"
    kms:
      key-path: "projects/<project>/locations/<location>/keyRings/<keyring>/cryptoKeys/<key>/cryptoKeyVersions/<version>"
    scopes: "idporten:dcr.write idporten:dcr.read"
    well-known-url: "https://oidc-ver2.difi.no/idporten-oidc-provider/.well-known/openid-configuration"
  maskinporten:
    client-id: "projects/<project>/secrets/<maskinporten-client-id>/versions/<version>"
    cert-chain: "projects/<project>/secrets/<maskinporten-cert-chain>/versions/<version>"
    kms:
      key-path: "projects/<project>/locations/<location>/keyRings/<keyring>/cryptoKeys/<key>/cryptoKeyVersions/<version>"
    scopes: "idporten:dcr.write idporten:dcr.read idporten:scopes.write"
    well-known-url: "https://test.maskinporten.no/.well-known/oauth-authorization-server"
```

## Development

If you're running locally, make sure you have access to the GCP resources and that you're authenticated with Application Default Credentials:

```shell script
gcloud auth login --update-adc
```

Then, assuming you have a Kubernetes cluster running locally (e.g.
using [minikube](https://github.com/kubernetes/minikube)):

```shell script
ulimit -n 4096  # for controller-gen
make run
make sample
```

## Verifying the Digdirator image and its contents

The image is signed "keylessly" (is that a word?) using [Sigstore cosign](https://github.com/sigstore/cosign).
To verify its authenticity run
```
cosign verify \
--certificate-identity "https://github.com/nais/digdirator/.github/workflows/build.yml@refs/heads/master" \
--certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
ghcr.io/nais/digdirator@sha256:<shasum>
```

The images are also attested with SBOMs in the [CycloneDX](https://cyclonedx.org/) format.
You can verify these by running
```
cosign verify-attestation --type cyclonedx \
--certificate-identity "https://github.com/nais/digdirator/.github/workflows/build.yml@refs/heads/master" \
--certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
ghcr.io/nais/digdirator@sha256:<shasum>
```