# pem2jwk

pem2jwk is a utility used to generate a public JWK from a given public key and certificate chain associated with the
key. The public key is assumed to be RSA with a key size of 2048 bits. The resulting JWK thus has the `alg` parameter
set to `RS256`.

## Context

The self-service APIs used by Digdirator requires usage of a client and an associated business certificate
(virksomhetssertifikat) for authentication with the `private_key_jwt` method,
ref. <https://docs.digdir.no/oidc_func_clientreg.html#klient-autentisering>.

Thus, you need to acquire the public key associated with the certificate as well as the complete certificate chain for
the certificate itself in order to generate the public JWK that should be registered at DigDir for the self-service API
client.

Do note that while the JWK itself does not expire, however the associated business certificate has its own expiration.
JWKs registered at DigDir will implicitly expire **1 year** after the initial registration, if not specified
otherwise.

## Usage

Run `make pem2jwk` in the root project directory to generate the binary.

```shell
Usage of ./bin/pem2jwk:
      --cert-chain-path string   The certificate chain including the certificate itself, in PEM format. (default "chain.pem")
      --output string            Path to output the resulting JWK to. (default "public.jwk")
      --public-key-path string   The public key associated with the certificate, in PEM format. (default "publickey.pem")
```
