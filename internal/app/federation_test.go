package app

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestFederationIdentityIsCurrentSessionOnly(t *testing.T) {
	a := testApp(t)
	alice := client(t, a, "alice")
	bob := client(t, a, "bob")
	for _, c := range []testClient{alice, bob} {
		w := c.call("GET", "/api/identity", nil)
		mustStatus(t, w, 200)
		var out map[string]string
		if e := json.Unmarshal(w.Body.Bytes(), &out); e != nil {
			t.Fatal(e)
		}
		if out["subject"] != c.user.Subject || out["issuer"] != a.Config.Issuer || len(out) != 2 {
			t.Fatal("identity response", out)
		}
	}
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/api/identity", nil))
	mustStatus(t, w, 401)
}
