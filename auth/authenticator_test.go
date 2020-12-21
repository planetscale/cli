package auth

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/stretchr/testify/assert"
)

const (
	testClientID     = "some-client-id"
	testClientSecret = "some-client-secret"
)

func TestVerifyDevice(t *testing.T) {
	tests := []struct {
		desc          string
		statusCode    int
		expectedBody  string
		deviceCodeRes string
		errExpected   bool
		want          *DeviceVerification
	}{
		{
			desc:         "returns device verification when authentication is successful",
			statusCode:   http.StatusOK,
			expectedBody: "client_id=some-client-id&scope=read_databases,write_databases",
			deviceCodeRes: `{
			"device_code": "some_device_code",
			"user_code": "1234567",
			"verification_uri": "http://example.com/device",
			"verification_uri_complete": "http://example.com/device?user_code=1234567",
			"expires_in": 1800,
			"interval": 5
			}`,
			want: &DeviceVerification{
				VerificationCompleteURL: "http://example.com/device?user_code=1234567",
				VerificationURL:         "http://example.com/device",
				DeviceCode:              "some_device_code",
				UserCode:                "1234567",
				CheckInterval:           time.Second * 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			srv, cleanup := setupServer(func(mux *http.ServeMux) {
				mux.HandleFunc("/oauth/authorize_device", func(w http.ResponseWriter, r *http.Request) {
					fmt.Println("in the endpoint")
					payload, err := ioutil.ReadAll(r.Body)
					if err != nil {
						t.Fatal(err)
					}

					assert.Equal(t, string(payload), tt.expectedBody)
					if tt.statusCode > 0 {
						w.WriteHeader(tt.statusCode)
					}

					_, err = w.Write([]byte(tt.deviceCodeRes))
					if err != nil {
						t.Fatal(err)
					}
				})
			})

			t.Cleanup(cleanup)

			authenticator, err := New(cleanhttp.DefaultClient(), testClientID, testClientSecret, SetBaseURL(srv.URL))
			if err != nil {
				t.Fatalf("error creating client: %s", err.Error())
			}

			got, err := authenticator.VerifyDevice(context.TODO())
			if err != nil {
				if tt.errExpected {
					// TODO(iheanyi): Assert error responses and stuff here.
				} else {
					t.Fatalf("unexpected error verifying device: %v", err)
				}
			}

			assert.Equal(t, got, tt.want, "unexpected device verification")
		})
	}

}

func setupServer(fn func(mux *http.ServeMux)) (*httptest.Server, func()) {
	mux := http.NewServeMux()

	fn(mux)
	server := httptest.NewServer(mux)

	return server, func() {
		server.Close()
	}
}
