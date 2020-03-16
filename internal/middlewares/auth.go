package middlewares

import (
	"errors"
	"log"
	"net/http"

	"github.com/redhatinsights/platform-go-middlewares/identity"
)

const (
	authErrorHeader = "Authentication error: "
	identityHeader  = "x-rh-identity"
	clientHeader    = "x-rh-receptor-controller-client-id"
	accountHeader   = "x-rh-receptor-controller-account"
	pskHeader       = "x-rh-receptor-controller-psk"
)

type serviceCredentials struct {
	clientID string
	account  string
	psk      string
}

func newServiceCredentials(clientID, account, psk string) (*serviceCredentials, error) {
	switch {
	case clientID == "":
		return nil, errors.New(authErrorHeader + "Missing x-rh-receptor-controller-client-id header")
	case account == "":
		return nil, errors.New(authErrorHeader + "Missing x-rh-receptor-controller-account header")
	case psk == "":
		return nil, errors.New(authErrorHeader + "Missing x-rh-receptor-controller-psk header")
	}
	return &serviceCredentials{
		clientID: clientID,
		account:  account,
		psk:      psk,
	}, nil
}

type serviceCredentialsValidator struct {
	knownServiceCredentials map[string]interface{}
}

func (scv *serviceCredentialsValidator) validate(sc *serviceCredentials) error {
	switch {
	case scv.knownServiceCredentials[sc.clientID] == nil:
		return errors.New(authErrorHeader + "Provided ClientID not attached to any known keys")
	case sc.psk != scv.knownServiceCredentials[sc.clientID]:
		return errors.New(authErrorHeader + "Provided PSK does not match known key for this client")
	}
	return nil
}

// AuthMiddleware allows the passage of parameters into the Authenticate middleware
type AuthMiddleware struct {
	Secrets map[string]interface{}
}

// Authenticate determines which authentication method should be used, and delegates identity header
// auth to the identity middleware
func (amw *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(identityHeader) != "" { // identity header auth
			identity.EnforceIdentity(next).ServeHTTP(w, r)
		} else { // token auth
			sr, err := newServiceCredentials(
				r.Header.Get(clientHeader),
				r.Header.Get(accountHeader),
				r.Header.Get(pskHeader),
			)
			if err != nil {
				http.Error(w, err.Error(), 401)
				return
			}
			log.Printf("Received service to service request from %v using account:%v", sr.clientID, sr.account)
			validator := serviceCredentialsValidator{knownServiceCredentials: amw.Secrets}
			if err := validator.validate(sr); err != nil {
				http.Error(w, err.Error(), 401)
				return
			}

			next.ServeHTTP(w, r)
		}
	})
}
