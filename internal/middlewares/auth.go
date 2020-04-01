package middlewares

import (
	"context"
	"errors"
	"net/http"

	"github.com/redhatinsights/platform-go-middlewares/identity"

	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	"github.com/sirupsen/logrus"
)

const (
	authErrorMessage   = "Authentication failed"
	authErrorLogHeader = "Authentication error: "
	identityHeader     = "x-rh-identity"
	clientHeader       = "x-rh-receptor-controller-client-id"
	accountHeader      = "x-rh-receptor-controller-account"
	pskHeader          = "x-rh-receptor-controller-psk"
)

// Principal interface can be implemented and expanded by various principal objects (type depends on middleware being used)
type Principal interface {
	GetAccount() string
}

type key int

var principalKey key

type serviceToServicePrincipal struct {
	account, clientID string
}

func (sp serviceToServicePrincipal) GetAccount() string {
	return sp.account
}

func (sp serviceToServicePrincipal) GetClientID() string {
	return sp.clientID
}

type identityPrincipal struct {
	account string
}

func (ip identityPrincipal) GetAccount() string {
	return ip.account
}

// GetPrincipal takes the request context and determines which middleware (identity header vs service to service) was used
// before returning a principal object.
func GetPrincipal(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey).(serviceToServicePrincipal)
	if !ok {
		id, ok := ctx.Value(identity.Key).(identity.XRHID)
		p := identityPrincipal{account: id.Identity.AccountNumber}
		return p, ok
	}
	return p, ok
}

type serviceCredentials struct {
	clientID string
	account  string
	psk      string
}

func newServiceCredentials(clientID, account, psk string) (*serviceCredentials, error) {
	switch {
	case clientID == "":
		return nil, errors.New(authErrorLogHeader + "Missing x-rh-receptor-controller-client-id header")
	case account == "":
		return nil, errors.New(authErrorLogHeader + "Missing x-rh-receptor-controller-account header")
	case psk == "":
		return nil, errors.New(authErrorLogHeader + "Missing x-rh-receptor-controller-psk header")
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
		return errors.New(authErrorLogHeader + "Provided ClientID not attached to any known keys")
	case sc.psk != scv.knownServiceCredentials[sc.clientID]:
		return errors.New(authErrorLogHeader + "Provided PSK does not match known key for this client")
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
				logger.Log.WithFields(logrus.Fields{"error": err}).Debug("Authentication failure")
				http.Error(w, authErrorMessage, 401)
				return
			}
			logger.Log.Debugf("Received service to service request from %v using account:%v", sr.clientID, sr.account)
			validator := serviceCredentialsValidator{knownServiceCredentials: amw.Secrets}
			if err := validator.validate(sr); err != nil {
				logger.Log.WithFields(logrus.Fields{"error": err}).Debug("Authentication failure")
				http.Error(w, authErrorMessage, 401)
				return
			}

			principal := serviceToServicePrincipal{account: sr.account, clientID: sr.clientID}

			ctx := context.WithValue(r.Context(), principalKey, principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}
