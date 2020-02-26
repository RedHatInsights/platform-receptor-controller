package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const (
	IDENTITY_HEADER_NAME = "x-rh-identity"
)

type MockClient struct {
}

func (mc MockClient) SendMessage(recipient string, route []string, payload interface{}, directive string) (*uuid.UUID, error) {
	myUUID, _ := uuid.NewRandom()
	return &myUUID, nil
}

func (mc MockClient) Close() {
}

func (mc MockClient) DisconnectReceptorNetwork() {
}

var _ = Describe("JobReciever", func() {

	var (
		jr                  *JobReceiver
		validIdentityHeader string
	)

	BeforeEach(func() {
		apiMux := mux.NewRouter()
		cm := controller.NewConnectionManager()
		mc := MockClient{}
		cm.Register("1234", "345", mc)
		jr = NewJobReceiver(cm, apiMux, nil)
		jr.Routes()

		identity := `{ "identity": {"account_number": "540155", "type": "User", "internal": { "org_id": "1979710" } } }`
		validIdentityHeader = base64.StdEncoding.EncodeToString([]byte(identity))
	})

	Describe("Connecting to the job receiver", func() {
		Context("With a valid identity header", func() {
			It("Should be able send a job to a connected customer", func() {

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

		})

		Context("Without an identity header", func() {
			It("Should fail to send a job to a connected customer", func() {

				postBody := "{\"account\": \"1234\", \"recipient\": \"345\", \"payload\": [\"678\"], \"directive\": \"fred:flintstone\"}"

				req, err := http.NewRequest("POST", "/job", strings.NewReader(postBody))
				Expect(err).NotTo(HaveOccurred())

				rr := httptest.NewRecorder()

				jr.router.ServeHTTP(rr, req)

				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			})

		})

	})
})
