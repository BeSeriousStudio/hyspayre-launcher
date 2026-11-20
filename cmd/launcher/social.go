package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (s *server) authStatus(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r, "GET") {
		return
	}
	profile, e := s.social.MyProfile()
	if e != nil {
		if strings.Contains(e.Error(), "Bitte melde dich an") || strings.Contains(e.Error(), "Anmeldung abgelaufen") {
			sendJSON(w, map[string]any{"authenticated": false}, 200)
			return
		}
		fail(w, e)
		return
	}
	sendJSON(w, map[string]any{"authenticated": true, "profile": json.RawMessage(profile)}, 200)
}
func (s *server) authLogin(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r, "POST") {
		return
	}
	var x struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Remember bool   `json:"remember"`
	}
	if e := decode(r, &x); e != nil {
		fail(w, e)
		return
	}
	if e := s.social.Login(x.Email, x.Password, x.Remember); e != nil {
		fail(w, e)
		return
	}
	sendJSON(w, map[string]bool{"ok": true}, 200)
}
func (s *server) authLogout(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r, "POST") {
		return
	}
	s.social.Logout()
	sendJSON(w, map[string]bool{"ok": true}, 200)
}
func (s *server) socialFriends(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r, "GET") {
		return
	}
	result, e := s.social.Friends()
	if e != nil {
		fail(w, e)
		return
	}
	sendJSON(w, json.RawMessage(result), 200)
}
func (s *server) socialSearch(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r, "POST") {
		return
	}
	var x struct {
		Query  string `json:"query"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}
	if e := decode(r, &x); e != nil {
		fail(w, e)
		return
	}
	result, e := s.social.Search(x.Query, x.Offset, x.Limit)
	if e != nil {
		fail(w, e)
		return
	}
	sendJSON(w, json.RawMessage(result), 200)
}
func (s *server) socialProfile(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r, "POST") {
		return
	}
	var x struct {
		UserID string `json:"user_id"`
	}
	if e := decode(r, &x); e != nil {
		fail(w, e)
		return
	}
	result, e := s.social.Profile(x.UserID)
	if e != nil {
		fail(w, e)
		return
	}
	sendJSON(w, json.RawMessage(result), 200)
}
func (s *server) socialAction(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r, "POST") {
		return
	}
	var x struct {
		UserID string `json:"user_id"`
		Action string `json:"action"`
	}
	if e := decode(r, &x); e != nil {
		fail(w, e)
		return
	}
	result, e := s.social.Action(x.UserID, x.Action)
	if e != nil {
		fail(w, e)
		return
	}
	sendJSON(w, json.RawMessage(result), 200)
}
