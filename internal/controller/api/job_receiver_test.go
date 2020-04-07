package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const (
	IDENTITY_HEADER_NAME      = "x-rh-identity"
	TOKEN_HEADER_CLIENT_NAME  = "x-rh-receptor-controller-client-id"
	TOKEN_HEADER_ACCOUNT_NAME = "x-rh-receptor-controller-account"
	TOKEN_HEADER_PSK_NAME     = "x-rh-receptor-controller-psk"
)

type MockClient struct {
}

func (mc MockClient) SendMessage(ctx context.Context, recipient string, route []string, payload interface{}, directive string) (*uuid.UUID, error) {
	myUUID, _ := uuid.NewRandom()
	return &myUUID, nil
}

func (mc MockClient) Ping(context.Context, string, []string) (interface{}, error) {
	return nil, nil
}

func (mc MockClient) Close() {
}

func (mc MockClient) GetCapabilities() interface{} {
	return struct{}{}
}

func init() {
	logger.InitLogger()
}

var _ = Describe("JobReceiver", func() {

	var (
		jr                  *JobReceiver
		validIdentityHeader string
	)

	BeforeEach(func() {
		apiMux := mux.NewRouter()
		cm := controller.NewConnectionManager()
		mc := MockClient{}
		cm.Register("1234", "345", mc)
		cfg := config.GetConfig()
		jr = NewJobReceiver(cm, apiMux, nil, cfg)
		jr.Routes()

		identity := `{ "identity": {"account_number": "540155", "type": "User", "internal": { "org_id": "1979710" } } }`
		validIdentityHeader = base64.StdEncoding.EncodeToString([]byte(identity))
	})

	Describe("Connecting to the job receiver", func() {
		Context("With a valid identity header", func() {
			It("Should be able to send a job to a connected customer", func() {

				postBody := "{\"account\": \"1234\", \"recipient\": \"345\", \"payload\": [\"678\"], \"directive\": \"fred:flintstone\"}"

				req, err := http.NewRequest("POST", "/job", strings.NewReader(postBody))
				Expect(err).NotTo(HaveOccurred())

				req.Header.Add(IDENTITY_HEADER_NAME, validIdentityHeader)

				rr := httptest.NewRecorder()

				jr.router.ServeHTTP(rr, req)

				Expect(rr.Code).To(Equal(http.StatusCreated))

				var m map[string]string
				json.Unmarshal(rr.Body.Bytes(), &m)
				Expect(m).Should(HaveKey("id"))
			})

			It("Should not allow sending a job to a disconnected customer", func() {

				postBody := "{\"account\": \"1234-not-here\", \"recipient\": \"345\", \"payload\": [\"678\"], \"directive\": \"fred:flintstone\"}"

				req, err := http.NewRequest("POST", "/job", strings.NewReader(postBody))
				Expect(err).NotTo(HaveOccurred())

				req.Header.Add(IDENTITY_HEADER_NAME, validIdentityHeader)

				rr := httptest.NewRecorder()

				jr.router.ServeHTTP(rr, req)

				Expect(rr.Code).To(Equal(http.StatusNotFound))
			})

			It("Should not allow sending a job with an empty account", func() {

				postBody := "{\"account\": \"\", \"recipient\": \"345\", \"payload\": [\"678\"], \"directive\": \"fred:flintstone\"}"

				req, err := http.NewRequest("POST", "/job", strings.NewReader(postBody))
				Expect(err).NotTo(HaveOccurred())

				req.Header.Add(IDENTITY_HEADER_NAME, validIdentityHeader)

				rr := httptest.NewRecorder()

				jr.router.ServeHTTP(rr, req)

				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			})

			It("Should not allow sending a job with malformed json", func() {

				postBody := "{\"account\" = \"1234-bad-json\", \"recipient\": \"345\", \"payload\": [\"678\"], \"directive\": \"fred:flintstone}"

				req, err := http.NewRequest("POST", "/job", strings.NewReader(postBody))
				Expect(err).NotTo(HaveOccurred())

				req.Header.Add(IDENTITY_HEADER_NAME, validIdentityHeader)

				rr := httptest.NewRecorder()

				jr.router.ServeHTTP(rr, req)

				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			})

			It("Should not allow sending a job with a string instead of json", func() {

				postBody := "account: 1234-string-value"

				req, err := http.NewRequest("POST", "/job", strings.NewReader(postBody))
				Expect(err).NotTo(HaveOccurred())

				req.Header.Add(IDENTITY_HEADER_NAME, validIdentityHeader)

				rr := httptest.NewRecorder()

				jr.router.ServeHTTP(rr, req)

				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			})

			It("Should not allow sending a job with missing required fields", func() {

				postBody := "{\"account\": \"1234\", \"recipient\": \"345\", \"payload\": [\"678\"]}"

				req, err := http.NewRequest("POST", "/job", strings.NewReader(postBody))
				Expect(err).NotTo(HaveOccurred())

				req.Header.Add(IDENTITY_HEADER_NAME, validIdentityHeader)

				rr := httptest.NewRecorder()

				jr.router.ServeHTTP(rr, req)

				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			})

			It("Should allow sending a job with unknown fields", func() {

				postBody := "{\"account\": \"1234\", \"recipient\": \"345\", \"payload\": [\"678\"], \"directive\": \"fred:flintstone\", \"extra\": \"field\"}"

				req, err := http.NewRequest("POST", "/job", strings.NewReader(postBody))
				Expect(err).NotTo(HaveOccurred())

				req.Header.Add(IDENTITY_HEADER_NAME, validIdentityHeader)

				rr := httptest.NewRecorder()

				jr.router.ServeHTTP(rr, req)

				Expect(rr.Code).To(Equal(http.StatusCreated))
			})

		})

		Context("Without an identity header or pre shared key", func() {
			It("Should fail to send a job to a connected customer", func() {

				postBody := "{\"account\": \"1234\", \"recipient\": \"345\", \"payload\": [\"678\"], \"directive\": \"fred:flintstone\"}"

				req, err := http.NewRequest("POST", "/job", strings.NewReader(postBody))
				Expect(err).NotTo(HaveOccurred())

				rr := httptest.NewRecorder()

				jr.router.ServeHTTP(rr, req)

				Expect(rr.Code).To(Equal(http.StatusUnauthorized))
			})

		})

		Context("With a valid token", func() {
			It("Should be able to send a job to a connected customer", func() {
				jr.config.ServiceToServiceCredentials["test_client_1"] = "12345"

				postBody := "{\"account\": \"1234\", \"recipient\": \"345\", \"payload\": [\"678\"], \"directive\": \"fred:flintstone\"}"

				req, err := http.NewRequest("POST", "/job", strings.NewReader(postBody))
				Expect(err).NotTo(HaveOccurred())

				req.Header.Add(TOKEN_HEADER_CLIENT_NAME, "test_client_1")
				req.Header.Add(TOKEN_HEADER_ACCOUNT_NAME, "0000001")
				req.Header.Add(TOKEN_HEADER_PSK_NAME, "12345")

				rr := httptest.NewRecorder()

				jr.router.ServeHTTP(rr, req)

				Expect(rr.Code).To(Equal(http.StatusCreated))

				var m map[string]string
				json.Unmarshal(rr.Body.Bytes(), &m)
				Expect(m).Should(HaveKey("id"))
			})
		})

		Context("With an invalid token", func() {
			It("Should not be able to send a job to a connected customer", func() {
				jr.config.ServiceToServiceCredentials["test_client_1"] = "12345"

				postBody := "{\"account\": \"1234\", \"recipient\": \"345\", \"payload\": [\"678\"], \"directive\": \"fred:flintstone\"}"

				req, err := http.NewRequest("POST", "/job", strings.NewReader(postBody))
				Expect(err).NotTo(HaveOccurred())

				req.Header.Add(TOKEN_HEADER_CLIENT_NAME, "test_client_1")
				req.Header.Add(TOKEN_HEADER_ACCOUNT_NAME, "0000001")
				req.Header.Add(TOKEN_HEADER_PSK_NAME, "6789")

				rr := httptest.NewRecorder()

				jr.router.ServeHTTP(rr, req)

				Expect(rr.Code).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("With an unknown client during token auth", func() {
			It("Should not be able to send a job to a connected customer", func() {
				jr.config.ServiceToServiceCredentials["test_client_1"] = "12345"

				postBody := "{\"account\": \"1234\", \"recipient\": \"345\", \"payload\": [\"678\"], \"directive\": \"fred:flintstone\"}"

				req, err := http.NewRequest("POST", "/job", strings.NewReader(postBody))
				Expect(err).NotTo(HaveOccurred())

				req.Header.Add(TOKEN_HEADER_CLIENT_NAME, "test_client_nil")
				req.Header.Add(TOKEN_HEADER_ACCOUNT_NAME, "0000001")
				req.Header.Add(TOKEN_HEADER_PSK_NAME, "12345")

				rr := httptest.NewRecorder()

				jr.router.ServeHTTP(rr, req)

				Expect(rr.Code).To(Equal(http.StatusUnauthorized))
			})
		})

	})
})
