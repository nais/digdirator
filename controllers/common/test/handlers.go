package test

import (
	"fmt"
	"net/http"
	"os"
	"path"
)

func MaskinportenHandler(clientID, orgno string) http.HandlerFunc {
	return handler(clientID, orgno, "maskinporten")
}

func IDPortenHandler(clientID string) http.HandlerFunc {
	return handler(clientID, "", "idporten")
}

func handler(clientID, orgno, clientType string) http.HandlerFunc {
	const testdataDir = "../common/testdata"
	clientExists := false
	clientTypeDir := path.Join(testdataDir, clientType)

	respond := func(w http.ResponseWriter, body string) {
		_, _ = w.Write([]byte(body))
	}

	respondFile := func(w http.ResponseWriter, name string) {
		body, _ := os.ReadFile(path.Join(testdataDir, name))
		_, _ = w.Write(body)
	}

	respondFileForClientType := func(w http.ResponseWriter, name string) {
		body, _ := os.ReadFile(path.Join(clientTypeDir, name))
		_, _ = w.Write(body)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case matchesPath(r, "/.well-known/openid-configuration", "/.well-known/oauth-authorization-server"):
			respond(w, fmt.Sprintf(`{
                "issuer": "http://%[1]s",
                "jwks_uri": "http://%[1]s/jwks",
                "token_endpoint": "http://%[1]s/token"
			}`, r.Host))
		case matchesPath(r, "/token"):
			respond(w, `{ "access_token": "token" }`)
		case matchesPath(r, "/api/v1/delegationsources"):
			respondFile(w, "delegationsources.json")
		case matchesPath(r, "/api/v1/clients"):
			switch r.Method {
			// GET (list) clients
			case http.MethodGet:
				if clientExists {
					respondFileForClientType(w, "list-response-exists.json")
					return
				}
				respondFileForClientType(w, "list-response.json")
			// POST (create) client
			case http.MethodPost:
				respondFileForClientType(w, "create-response.json")
			}
		case matchesPath(r, "/api/v1/clients/"+clientID):
			switch r.Method {
			// PUT (update) existing client
			case http.MethodPut:
				respondFileForClientType(w, "update-response.json")
			// DELETE existing client
			case http.MethodDelete:
				w.WriteHeader(http.StatusOK)
			}
		// POST (register) JWKS for client
		case matchesMethodPath(r, http.MethodPost, "/api/v1/clients/"+clientID+"/jwks"):
			if clientExists {
				respondFile(w, "register-jwks-response-exists.json")
				return
			}
			respondFile(w, "register-jwks-response.json")
			clientExists = true
		case matchesPath(r, "/api/v1/scopes"):
			switch r.Method {
			// GET (list) all scopes owned by the authenticated organization
			case http.MethodGet:
				if clientExists {
					respondFileForClientType(w, "list-scopes-response-exists.json")
					return
				}
				respondFileForClientType(w, "list-scopes-response.json")
			// POST (create) scope
			case http.MethodPost:
				respondFileForClientType(w, "create-scope-response.json")
			// PUT (update) scope
			case http.MethodPut:
				respondFileForClientType(w, "update-scope-response.json")
			}
		// GET consumer access for a scope
		case matchesMethodPath(r, http.MethodGet, "/api/v1/scopes/access"):
			respondFileForClientType(w, "specific-scope-access-all.json")
		// PUT (add) consumer access for a scope
		case matchesMethodPath(r, http.MethodPut, "/api/v1/scopes/access/"+orgno):
			respondFileForClientType(w, "specific-scope-info.json")
		// DELETE (remove) consumer access for a scope
		case matchesMethodPath(r, http.MethodDelete, "/api/v1/scopes/access/999999999"):
			respondFileForClientType(w, "specific-scope-info.json")
		// GET accessible maskinporten scopes
		case matchesMethodPath(r, http.MethodGet, "/api/v1/scopes/access/all"):
			respondFileForClientType(w, "scopes-access-all.json")
		// GET open maskinporten scopes
		case matchesMethodPath(r, http.MethodGet, "/api/v1/scopes/all"):
			respondFileForClientType(w, "list-scopes-response-exists.json")
		}
	}
}

func matchesPath(r *http.Request, paths ...string) bool {
	for _, p := range paths {
		if r.URL.Path == p {
			return true
		}
	}
	return false
}

func matchesMethodPath(r *http.Request, method string, path string) bool {
	return r.URL.Path == path && r.Method == method
}
