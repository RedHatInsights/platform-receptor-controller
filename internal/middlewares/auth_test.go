package middlewares_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/RedHatInsights/platform-receptor-controller/internal/middlewares"
)

const (
	TOKEN_HEADER_CLIENT_NAME  = "x-rh-receptor-controller-client-id"
	TOKEN_HEADER_ACCOUNT_NAME = "x-rh-receptor-controller-account"
	TOKEN_HEADER_PSK_NAME     = "x-rh-receptor-controller-psk"
	authFailure               = "Authentication failed"
)

func GetTestHandler() http.HandlerFunc {
	fn := func(rw http.ResponseWriter, req *http.Request) {}

	return http.HandlerFunc(fn)
}

func boiler(req *http.Request, expectedStatusCode int, expectedBody string, amw *middlewares.AuthMiddleware) {
	rr := httptest.NewRecorder()
	handler := amw.Authenticate(GetTestHandler())
	handler.ServeHTTP(rr, req)

	Expect(rr.Code).To(Equal(expectedStatusCode))
	Expect(rr.Body.String()).To(Equal(expectedBody))
}

var _ = Describe("Auth", func() {
	var (
		req *http.Request
		amw *middlewares.AuthMiddleware
	)

	BeforeEach(func() {
		knownSecrets := make(map[string]interface{})
		knownSecrets["test_client_1"] = "12345"
		amw = &middlewares.AuthMiddleware{Secrets: knownSecrets}
		r, err := http.NewRequest("GET", "/api/receptor-controller/v1/job", nil)
		if err != nil {
			panic("Test error unable to get new request")
		}
		req = r
	})

	Describe("Using token authentication", func() {
		Context("With no missing token auth headers", func() {
			It("Should return 200 when the key is correct", func() {
				req.Header.Add(TOKEN_HEADER_CLIENT_NAME, "test_client_1")
				req.Header.Add(TOKEN_HEADER_ACCOUNT_NAME, "0000001")
				req.Header.Add(TOKEN_HEADER_PSK_NAME, "12345")

				boiler(req, 200, "", amw)
			})

			It("Should return a 401 when the key is incorrect", func() {
				req.Header.Add(TOKEN_HEADER_CLIENT_NAME, "test_client_1")
				req.Header.Add(TOKEN_HEADER_ACCOUNT_NAME, "0000001")
				req.Header.Add(TOKEN_HEADER_PSK_NAME, "678910")

				boiler(req, 401, authFailure+"\n", amw)
			})

			It("Should return a 401 when the client id is unknown", func() {
				req.Header.Add(TOKEN_HEADER_CLIENT_NAME, "test_client_nil")
				req.Header.Add(TOKEN_HEADER_ACCOUNT_NAME, "0000001")
				req.Header.Add(TOKEN_HEADER_PSK_NAME, "12345")

				boiler(req, 401, authFailure+"\n", amw)
			})
		})

		Context("With missing token auth headers", func() {
			It("Should return 401 when the client id header is missing", func() {
				req.Header.Add(TOKEN_HEADER_ACCOUNT_NAME, "0000001")
				req.Header.Add(TOKEN_HEADER_PSK_NAME, "12345")

				boiler(req, 401, authFailure+"\n", amw)
			})

			It("Should return 401 when the account header is missing", func() {
				req.Header.Add(TOKEN_HEADER_CLIENT_NAME, "test_client_1")
				req.Header.Add(TOKEN_HEADER_PSK_NAME, "12345")

				boiler(req, 401, authFailure+"\n", amw)
			})

			It("Should return 401 when the psk header is missing", func() {
				req.Header.Add(TOKEN_HEADER_CLIENT_NAME, "test_client_1")
				req.Header.Add(TOKEN_HEADER_ACCOUNT_NAME, "0000001")

				boiler(req, 401, authFailure+"\n", amw)
			})
		})
	})
})
