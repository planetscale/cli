package planetscale

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestUsers_GetCurrent(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/user")
		w.WriteHeader(200)
		_, err := w.Write([]byte(`{"id":"user-1","display_name":"Ada Lovelace","name":"Ada","email":"ada@example.com"}`))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	user, err := client.Users.GetCurrent(context.Background())
	c.Assert(err, qt.IsNil)
	c.Assert(user.ID, qt.Equals, "user-1")
	c.Assert(user.DisplayName, qt.Equals, "Ada Lovelace")
	c.Assert(user.Email, qt.Equals, "ada@example.com")
}
