package crypto

import (
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
)

func LoadPemCertChainToX5C(path string) ([]string, error) {
	certPEMBlock, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("loading PEM cert chain from path %s: %w", path, err)
	}
	var certDERBlock *pem.Block
	base64EncodedCerts := make([]string, 0)
	for {
		certDERBlock, certPEMBlock = pem.Decode(certPEMBlock)
		if certDERBlock == nil {
			break
		}
		if certDERBlock.Type == "CERTIFICATE" {
			encoded := base64.StdEncoding.EncodeToString(certDERBlock.Bytes)
			base64EncodedCerts = append(base64EncodedCerts, encoded)
		}
	}
	return base64EncodedCerts, nil
}
