package middlewares

import (
	"errors"
	"log"
	"net/http"

	"github.com/redhatinsights/platform-go-middlewares/identity"
)

const authErrorHeader = "Authentication error: "

type serviceRequest struct {
	clientID string
	account  string
	psk      string
}

func newServiceRequest(clientID, account, psk string) (*serviceRequest, error) {
	switch {
	case clientID == "":
		return nil, errors.New(authErrorHeader + "Missing receptor-controller-client-id header")
	case account == "":
		return nil, errors.New(authErrorHeader + "Missing receptor-controller-account header")
	case psk == "":
		return nil, errors.New(authErrorHeader + "Missing receptor-controller-psk header")
	}
	return &serviceRequest{
		clientID: clientID,
		account:  account,
		psk:      psk,
	}, nil
}

func (sr *serviceRequest) validate(secrets map[string]interface{}) error {
	switch {
	case secrets[sr.clientID] == nil:
		return errors.New(authErrorHeader + "Provided ClientID not attached to any known keys")
	case sr.psk != secrets[sr.clientID]:
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
		if r.Header.Get("x-rh-identity") != "" { // identity header auth
			identity.EnforceIdentity(next).ServeHTTP(w, r)
		} else { // token auth
			sr, err := newServiceRequest(
				r.Header.Get("receptor-controller-client-id"),
				r.Header.Get("receptor-controller-account"),
				r.Header.Get("receptor-controller-psk"),
			)
			if err != nil {
				http.Error(w, err.Error(), 401)
				return
			}
			log.Printf("Received service to service request from %v using account:%v", sr.clientID, sr.account)
			if err := sr.validate(amw.Secrets); err != nil {
				http.Error(w, err.Error(), 401)
				return
			}

			next.ServeHTTP(w, r)
		}
	})
}
