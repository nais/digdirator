# jwk-updater

jwk-updater updates the registered JWKS (JWK Set) for a given DigDir client.
Optionally, a new JWK can be added to the existing Set. 
Note that DigDir enforces a maximum limit of 5 keys within a JWKS.

## Context

Clients at DigDir using the `private_key_jwt` method for
authentication (<https://docs.digdir.no/oidc_func_clientreg.html#klient-autentisering>)
must have an associated JWKS.

JWKs registered at DigDir have two separate expirations.

1. The public X.509 certificate within the JWK (found in the `x5c` parameter) has an expiry indicated by the `Not After` field.
2. The JWK itself also has an `exp` parameter. The value indicates when the key itself expires, in epoch time. If not specified in the update request, this is implicitly set to **1 year** in the future (which is also the maximum value).

This means that we need to regularly generate new keys with new certificates and register these for clients.
For clients managed by Digdirator, this is fairly easy. 
For other clients - such as the client used by Digdirator itself to authenticate with the management APIs - this is more complicated.

This utility helps with updating the JWKS for a given DigDir client by 

- updating the JWK(s) in the set, thus extending the `exp` parameter
- removing any JWK(s) that have expired certificates from the set
- and optionally, adding a new JWK (that hopefully contains a new certificate with an expiry in the distant future) to the set

## Usage

Run `make jwk-updater` in the root project directory to generate the binary.

```shell
Usage of ./bin/jwk-updater:
  --client-id string                            The client ID for the DigDir client to update.
  --digdir.admin.base-url string                Base URL endpoint for interacting with Digdir Client Registration API
  --digdir.maskinporten.cert-chain string       Secret path in Google Secret Manager to PEM file containing certificate chain for authenticating to DigDir.
  --digdir.maskinporten.client-id string        Client ID / issuer for JWT assertion when authenticating to DigDir.
  --digdir.maskinporten.kms.key-path string     Maskinporten Google KmsConfig resource path used to sign JWT assertion when authenticating to DigDir.
  --digdir.maskinporten.scopes string           List of scopes for JWT assertion when authenticating to DigDir with Maskinporten.
  --digdir.maskinporten.well-known-url string   URL to Maskinporten well-known discovery metadata document.
  --jwk-path string                             Path to the new public key in JWK format that should be added to the client.
```

All the flags above are required, with the exception of `jwk-path`
If `jwk-path` is empty, only existing keys that do not contain expired certificates will be POST-ed back to the DigDir APIs.