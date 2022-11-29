# Enable Digdirator

How NAIS tenants can enable automated registration and lifecycle management of ID-porten and Maskinporten clients.

[Digdirator](https://github.com/nais/digdirator) is a feature that integrates with Digdir self-service API.
Before Digdirator can use the self-service API, the tenant must receive administration clients from Digdir,
one for each client type (Maskinporten & ID-porten).
The Digdir self-service API is secured with oAuth2 using a `business certificate`.

An overview of the setup is as follows:

* Clients exist in Digdir and a `business certificate` is configured
* Clients are configured with [scopes](#digdir-configuration) required
* Upload `business certificate` to Google Key Management Service in your project
* Upload business certificate `certificate-chain` to Secret Manager in your project

## Pre-requisites for Tenant

#### Digdir configuration

* Configure an administration client for ID-porten
* Configure an administration client for Maskinporten
* Responsible tenant have a `business certificate` registered in Digdir
* The ID-porten administration client is configured with scopes: `idporten:dcr.write idporten:dcr.read`
* The Maskinporten administration is configured with
  scopes: `idporten:dcr.write idporten:dcr.read idporten:scopes.write`

For each administration client the tenant have to provide the following information:

* The Client ID

> The Client ID is set by Digdirator as `jwt.claim.iss` to authenticate against Digdir self-service API

#### NAIS configuration

We do really care about your compadres (tenants) and we think that a separation of concerns is a good & secure way to
go.
It also helps us to keep the cluster secure and stable. The configuration setup for Digdirator favor security as
NAIS never have direct access to your business certificate.

When setup in Digdir is confirmed by tenant, before enabling Digdirator, the following steps must be completed:

##### Business certificate

> Update of a certificate only requires the tenant to provide the new `<version>`

The tenant upload their `business certificate` to
[Google Cloud KMS](https://cloud.google.com/kms/docs/how-tos). Digdirator never have direct access to the certificate.
Once it is uploaded the `business certificate` can only be used for cryptographic operations.
The `business certificate` can never be downloaded or retrieved from the KMS storage (not even from yourself).
An authenticated & authorized Digdirator can only request the `KMS` to sign a payload containing an unsigned
token-header
with claims. The KMS then returns a signed JWT, this JWT is later used to authenticate against Digdir self-service API.

When the certificate is [successfully uploaded](#import-certificates), the tenant must provide NAIS the following
information:

`projects/<project-id>/locations/<location>/keyRings/<key-ring>/cryptoKeys/<key-name>/cryptoKeyVersions/<version>`

Resource name can be copied to the clipboard in the Google Cloud Console: `Copy resource name`.

##### Certificate chain

> This information unlikely to change, only if a new certificate type is added to the Google KMS.


Now your probably are wondering why another secret storage we already configured KMS?

Well, when authenticating using a `buissness certificate` Digdir requires the `certificate chain` in the token header.

The public `certificate chain` is set to the `x5c` (X.509 certificate chain) Header Parameter, corresponding to the key
used to
digitally sign the JWS (JSON Web Signature).

`Google Cloud Key Mangement Service` is designed as a cryptographic system: nobody, including yourself, can get the keys
out: this means they're
locked inside the system, and you don't have to worry in practice about them leaking. The tradeoff is that the only
thing you can do with those keys is encrypting, decrypt, and other cryptographic operations.

But when you do have configuration info like a certificate chain or a password, where your software actually needs the
secret, not cryptographic operations, then `Secret Manager` is designed for that use case.

When the certificate chain is [successfully uploaded](#import-certificates) to `Secret Manager`, the tenant must provide
the following
information:

* Name of the secret
* Version of the secret
* Project-ID of the Secret Manager

Uploaded example format:

```Text
-----BEGIN CERTIFICATE-----
MIIFCDECEBC...
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIE3sKEA...
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIFZTKK...
-----END CERTIFICATE-----
```

## NAIS

NAIS configures Digdirator with a service account and roles to access Google Cloud KMS and Secret Manager in your
cluster project. The service account assigned Digdirator is given the role `roles/cloudkms.signerVerifier`,
which enables Sign, Verify, and GetPublicKey operations, more specific:

```Text
cloudkms.cryptoKeyVersions.useToSign
cloudkms.cryptoKeyVersions.useToVerify
cloudkms.cryptoKeyVersions.viewPublicKey
cloudkms.locations.get
cloudkms.locations.list
resourcemanager.projects.get
```

For Secret Manager the service account is given the role `roles/secretmanager.secretAccessor` which allows Digdirator to
access the payload of secrets:

```Text
resourcemanager.projects.get
resourcemanager.projects.list
secretmanager.versions.access
```

NAIS configure Digdirator with the information provided by the tenant, you sit down relax your fingers and NAIS handles
the rest.

### Summary

If we were to express the above information in yaml, it would look like this:

```yaml
Maskinporten:
  digdir:
    client-id: "123456789"
  kms:
    key: "projects/123456789/locations/europe-north1/keyRings/nais-test/cryptoKeys/digdirator/cryptoKeyVersions/1"
  secret-manager:
    name: "digdirator"
    project-id: "123456789"
    version: "1"
ID-porten:
  digdir:
    client-id: "123456789"
  kms:
    key: "projects/123456789/locations/europe-north1/keyRings/nais-test/cryptoKeys/digdirator/cryptoKeyVersions/1"
  secret-manager:
    name: "digdirator"
    project-id: "123456789"
    version: "1"
```

## Import Certificates

### Pre-requisites

[gcloud](https://cloud.google.com/sdk/docs/install) CLI is installed and configured with a user that have access to the
project.

Some configuration can be done in the Google Cloud Console, automatic wrap and import must be done with the `gcloud`
CLI.

#### Google Cloud KMS

1. Create a [target key and key ring](https://cloud.google.com/kms/docs/importing-a-key#create_targets) in your project
2. Create a [import job](https://cloud.google.com/kms/docs/importing-a-key#import_job) for the target key.
3. Make an import request for [key](https://cloud.google.com/kms/docs/importing-a-key#request_import)

4. Wrap and import of key can be done in automatically or manually.

    * Automatically [wrap and import](https://cloud.google.com/kms/docs/importing-a-key#automatically_wrap_and_import)
      by `gcloud` CLI
    * Manually is divided into 2 steps
        * [Manually wrap](https://cloud.google.com/kms/docs/wrapping-a-key) using OpenSSL for Linux or macOS.
        * [Manually import](https://cloud.google.com/kms/docs/importing-a-key#importing_a_manually-wrapped_key) in the
          Google
          Cloud Console or gcloud CLI.

#### Google Secret Manager

* Create a [secret](https://cloud.google.com/secret-manager/docs/creating-and-accessing-secrets#creating_a_secret) in
  your project

