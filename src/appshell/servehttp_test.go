package appshell

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServeHTTPRejectsMalformedPaths(t *testing.T) {
	app := &App{}
	cases := []string{
		"/media/snapchat/2",       // missing id segment
		"/media/snapchat/2/x",     // id not numeric
		"/media/snapchat/x/0",     // userId not numeric
		"/media/snapchat",         // missing userId and id
		"/media/",                 // empty
	}
	for _, path := range cases {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("path %q: got status %d, want 404", path, rec.Code)
		}
	}
}
