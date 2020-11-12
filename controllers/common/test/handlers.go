package test

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

type HandlerType string

const (
	IDPortenHandlerType     HandlerType = "idporten"
	MaskinportenHandlerType HandlerType = "maskinporten"
)

func DigdirHandler(clientID string, handlerType HandlerType) http.HandlerFunc {
	var clientExists = false
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/idporten-oidc-provider/token":
			response := `{ "access_token": "token" }`
			_, _ = w.Write([]byte(response))
		// GET (list) clients
		case r.URL.Path == "/clients" && r.Method == http.MethodGet:
			var path string
			if clientExists {
				path = fmt.Sprintf("../common/testdata/%s/list-response-exists.json", handlerType)
			} else {
				path = fmt.Sprintf("../common/testdata/%s/list-response.json", handlerType)
			}
			response, _ := ioutil.ReadFile(path)
			_, _ = w.Write(response)
		// POST (create) client
		case r.URL.Path == "/clients" && r.Method == http.MethodPost:
			response, _ := ioutil.ReadFile(fmt.Sprintf("../common/testdata/%s/create-response.json", handlerType))
			_, _ = w.Write(response)
		// PUT (update) existing client
		case r.URL.Path == fmt.Sprintf("/clients/%s", clientID) && r.Method == http.MethodPut:
			response, _ := ioutil.ReadFile(fmt.Sprintf("../common/testdata/%s/update-response.json", handlerType))
			_, _ = w.Write(response)
		// DELETE existing client
		case r.URL.Path == fmt.Sprintf("/clients/%s", clientID) && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		// POST JWKS (overwriting)
		case r.URL.Path == fmt.Sprintf("/clients/%s/jwks", clientID) && r.Method == http.MethodPost:
			var path string
			if clientExists {
				path = "../common/testdata/register-jwks-response-exists.json"
			} else {
				path = "../common/testdata/register-jwks-response.json"
				clientExists = true
			}
			response, _ := ioutil.ReadFile(path)
			_, _ = w.Write(response)
		}
	}
}
