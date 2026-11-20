package social

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Regression: projects with SQL 07 but without SQL 08 must still find players.
func TestSearchFallsBackToLegacyRPC(t *testing.T) {
	oldCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth/v1/token":
			_, _ = w.Write([]byte(`{"access_token":"token","refresh_token":"renew","expires_in":3600}`))
		case "/rest/v1/rpc/bss_social_search_live":
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`{"code":"PGRST202","message":"Could not find the function public.bss_social_search_live(p_limit,p_offset,p_query) in the schema cache"}`))
		case "/rest/v1/rpc/bss_social_search":
			oldCalls++
			var params map[string]any
			_ = json.NewDecoder(r.Body).Decode(&params)
			if params["p_query"] != "bss_" {
				t.Errorf("bad legacy query %v", params)
			}
			_, _ = w.Write([]byte(`{"players":[{"user_id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","profile_name":"BSS_Azura"}]}`))
		default:
			t.Errorf("unexpected API %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	client := NewCustom(server.URL, "key", t.TempDir(), server.Client())
	if err := client.Login("test@hyspayre.com", "pass", false); err != nil {
		t.Fatal(err)
	}
	res, e := client.Search("bss_", 0, 40)
	if e != nil {
		t.Fatal(e)
	}
	var data struct {
		Players []map[string]any `json:"players"`
		Legacy  bool             `json:"legacy"`
		HasMore bool             `json:"has_more"`
	}
	if e := json.Unmarshal(res, &data); e != nil {
		t.Fatal(e)
	}
	if !data.Legacy || data.HasMore || len(data.Players) != 1 || data.Players[0]["profile_name"] != "BSS_Azura" {
		t.Fatalf("bad fallback %s", res)
	}
	if oldCalls != 1 {
		t.Fatal("legacy API not called")
	}
	res, e = client.Search("b", 0, 40)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(string(res), "mindestens zwei Zeichen") {
		t.Fatalf("single char not explained %s", res)
	}
	if oldCalls != 1 {
		t.Fatal("legacy API must not be called for single char")
	}
}
func TestSearchDoesNotFallbackForOtherErrors(t *testing.T) {
	oldCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/v1/token":
			_, _ = w.Write([]byte(`{"access_token":"token","refresh_token":"renew","expires_in":3600}`))
		case "/rest/v1/rpc/bss_social_search_live":
			w.WriteHeader(403)
			_, _ = w.Write([]byte(`{"code":"42501","message":"permission denied"}`))
		case "/rest/v1/rpc/bss_social_search":
			oldCalls++
			_, _ = w.Write([]byte(`{"players":[]}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	client := NewCustom(server.URL, "key", t.TempDir(), server.Client())
	if e := client.Login("test@hyspayre.com", "pass", false); e != nil {
		t.Fatal(e)
	}
	_, err := client.Search("bss_", 0, 40)
	if err == nil || !strings.Contains(err.Error(), "permission denied") || oldCalls != 0 {
		t.Fatalf("must not mask authorization error: %v, %d", err, oldCalls)
	}
}
