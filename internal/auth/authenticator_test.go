package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/stretchr/testify/assert"
)

const (
	testClientID     = "some-client-id"
	testClientSecret = "some-client-secret"

	testPayload = "client_id=some-client-id&scope=read_databases+write_databases+read_user+read_organization"
)

func TestVerifyDevice(t *testing.T) {
	tests := []struct {
		desc          string
		deviceCodeRes string
		errExpected   bool
		want          *DeviceVerification
	}{
		{
			desc: "returns device verification when authentication is successful",
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
				ExpiresAt:               clock.NewMock().Now().Add(time.Duration(1800) * time.Second),
			},
		},
		{
			desc: "returns device verification with check interval of 5 seconds when interval is 0",
			deviceCodeRes: `{
			"device_code": "some_device_code",
			"user_code": "1234567",
			"verification_uri": "http://example.com/device",
			"verification_uri_complete": "http://example.com/device?user_code=1234567",
			"expires_in": 1800,
			"interval": 0
			}`,
			want: &DeviceVerification{
				VerificationCompleteURL: "http://example.com/device?user_code=1234567",
				VerificationURL:         "http://example.com/device",
				DeviceCode:              "some_device_code",
				UserCode:                "1234567",
				CheckInterval:           time.Second * 5,
				ExpiresAt:               clock.NewMock().Now().Add(time.Duration(1800) * time.Second),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/oauth/authorize_device", func(w http.ResponseWriter, r *http.Request) {
				// HTTP handlers run in goroutines, thus we cannot use t.Fatal.
				// See: https://pkg.go.dev/testing#T.FailNow.
				payload, err := io.ReadAll(r.Body)
				if err != nil {
					panicf("failed to read request body: %v", err)
				}

				assert.Equal(t, testPayload, string(payload))
				if _, err := io.WriteString(w, tt.deviceCodeRes); err != nil {
					panicf("failed to write response bytes: %v", err)
				}
			})

			srv := httptest.NewServer(mux)
			defer srv.Close()

			mockClock := clock.NewMock()
			authenticator, err := New(cleanhttp.DefaultClient(), testClientID, testClientSecret, SetBaseURL(srv.URL), WithMockClock(mockClock))
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}

			got, err := authenticator.VerifyDevice(context.TODO())
			if err != nil {
				if tt.errExpected {
					// TODO(iheanyi): Assert error responses and stuff here.
				} else {
					t.Fatalf("unexpected error verifying device: %v", err)
				}
			}

			assert.Equal(t, tt.want, got, "unexpected device verification")
		})
	}
}

func panicf(format string, a ...any) {
	panic(fmt.Sprintf(format, a...))
}
