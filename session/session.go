package session

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"time"

	"encoding/base64"
)

type Store interface {
	GetSession(id string) (*Session, error)
	SaveSession(*Session) error
	DestroySession(*Session) error
}

type Session struct {
	Id     string
	Values map[string]string
}

func Start(store Store, w http.ResponseWriter, r *http.Request) (*Session, error) {
	if store == nil {
		return nil, fmt.Errorf("No session store")
	}
	for _, cookie := range r.Cookies() {
		// This always complains about the Expiry being invalid,
		// so we ignore it.
		// if err := cookie.Valid(); err != nil {
		//      println("Invalid cookie", err.Error())
		//		continue
		// }
		if cookie.Name == "SessionID" {
			sess, err := store.GetSession(cookie.Value)
			if err != nil {
				log.Println("Invalid SessionID, starting new")
				return newSession(store, w, r)
			}
			return sess, nil
		}
	}
	log.Println("No session cookie, starting new")
	return newSession(store, w, r)
}

func newSession(store Store, w http.ResponseWriter, r *http.Request) (*Session, error) {
	var id [30]byte
	_, err := rand.Read(id[:])
	if err != nil {
		return nil, err
	}
	idStr := base64.URLEncoding.EncodeToString(id[:])
	sess := Session{
		Id:     idStr,
		Values: make(map[string]string),
	}

	// Expire the session after a week
	maxAge := 60 * 60 * 24 * 7

	cookie := http.Cookie{
		Name:     "SessionID",
		Value:    idStr,
		Expires:  time.Now().Add(time.Second * time.Duration(maxAge)),
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		MaxAge:   maxAge,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &cookie)
	if err := store.SaveSession(&sess); err != nil {
		return nil, err
	}

	return &sess, nil
}

func (s *Session) Set(key, value string) {
	if s.Values == nil {
		s.Values = make(map[string]string)
	}
	s.Values[key] = value
}

func (s *Session) Get(key string) string {
	if s.Values == nil {
		return ""
	}
	return s.Values[key]
}
