package social

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginAndSocialWorkflows(t *testing.T) {
	calls := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls[r.URL.Path]++
		if r.Header.Get("apikey") != "public_test_key" {
			t.Errorf("public key missing")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth/v1/token":
			if r.URL.Query().Get("grant_type") != "password" {
				t.Errorf("unexpected grant type")
			}
			var login map[string]string
			_ = json.NewDecoder(r.Body).Decode(&login)
			if login["email"] != "friend@hyspayre.test" || login["password"] != "long-password" {
				t.Errorf("incorrect auth request")
			}
			_, _ = w.Write([]byte(`{"access_token":"access-123","refresh_token":"refresh-123","expires_in":3600}`))
		case "/rest/v1/rpc/bss_me":
			if r.Header.Get("Authorization") != "Bearer access-123" {
				t.Errorf("access not provided")
			}
			_, _ = w.Write([]byte(`{"user_id":"12345678-1234-1234-1234-123456789abc","profile_name":"BSS_Azura","rank":{"name":"Gold 1","rp":403,"progress":3,"required":100}}`))
		case "/rest/v1/rpc/bss_social_search_live":
			var args map[string]any
			_ = json.NewDecoder(r.Body).Decode(&args)
			if args["p_query"] != "bss_" {
				t.Errorf("missing partial search")
			}
			_, _ = w.Write([]byte(`{"players":[{"profile_name":"BSS_Azura","user_id":"12345678-1234-1234-1234-123456789abc"}],"total":1,"has_more":false}`))
		case "/rest/v1/rpc/bss_social_friends":
			_, _ = w.Write([]byte(`{"connections":[{"profile_name":"BSS_Azura","status":"friend"}]}`))
		case "/rest/v1/rpc/bss_social_profile":
			_, _ = w.Write([]byte(`{"profile_name":"BSS_Azura","rank_name":"Gold 1","rank_rp":403}`))
		case "/rest/v1/rpc/bss_social_action":
			var args map[string]string
			_ = json.NewDecoder(r.Body).Decode(&args)
			if args["p_action"] != "request" {
				t.Errorf("wrong friend action")
			}
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/auth/v1/logout":
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	client := NewCustom(server.URL, "public_test_key", t.TempDir(), server.Client())
	if _, e := client.MyProfile(); e == nil {
		t.Fatal("expected login protection")
	}
	if e := client.Login("friend@hyspayre.test", "long-password", false); e != nil {
		t.Fatal(e)
	}
	for _, fn := range []func() (json.RawMessage, error){client.MyProfile, client.Friends, func() (json.RawMessage, error) { return client.Search("bss_", 0, 40) }, func() (json.RawMessage, error) { return client.Profile("12345678-1234-1234-1234-123456789abc") }, func() (json.RawMessage, error) {
		return client.Action("12345678-1234-1234-1234-123456789abc", "request")
	}} {
		result, e := fn()
		if e != nil {
			t.Fatal(e)
		}
		if !json.Valid(result) {
			t.Fatal("invalid JSON")
		}
	}
	if _, e := client.Action("12345678-1234-1234-1234-123456789abc", "grant_admin"); e == nil {
		t.Fatal("arbitrary action allowed")
	}
	if _, e := client.Search("", 0, 40); e == nil {
		t.Fatal("empty query should fail")
	}
	if _, e := client.Profile("bad-id"); e == nil {
		t.Fatal("invalid id accepted")
	}
	if calls["/rest/v1/rpc/bss_social_friends"] != 1 {
		t.Fatal("friends RPC not called")
	}
	client.Logout()
	if _, e := client.Friends(); e == nil || !strings.Contains(e.Error(), "melde dich an") {
		t.Fatal("logout not honored")
	}
}
