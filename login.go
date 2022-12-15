package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"

	"fediwiki/session"

	"golang.org/x/oauth2"
)

var loginTemplate *template.Template

func loginHandler(clientDB OAuthClientStore, sessionDB session.Store) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("URL", r.URL.Path)
		session, err := session.Start(sessionDB, w, r)
		if err != nil {
			log.Println(err)
		}
		log.Println("SessionID", session.Id)
		switch r.Method {
		case "GET":
			if code := r.URL.Query().Get("code"); code != "" {
				claimedstate := r.URL.Query().Get("state")
				log.Println("code", code, claimedstate)
				if claimedstate != session.Get("OAuthState") {
					w.WriteHeader(400)
					io.WriteString(w, "Bad request state")
					return
				}
				host := session.Get("OAuthHost")
				if host == "" {
					w.WriteHeader(500)
					io.WriteString(w, "Internal error")
					return
				}

				client, err := clientDB.GetClient(host)

				oauthconf := oauth2.Config{
					ClientID:     client.ClientId,
					ClientSecret: client.ClientSecret,
					Scopes:       []string{"read:accounts"},
					Endpoint: oauth2.Endpoint{
						AuthURL:  "https://" + host + "/oauth/authorize",
						TokenURL: "https://" + host + "/oauth/token",
					},
					RedirectURL: client.RedirectURI,
				}

				// FIXME: Use the token to make a request to /api/v1/accounts/verify_credentials. (NOTE: this is mastodon specific)
				tok, err := oauthconf.Exchange(context.TODO(), code)
				if err != nil {
					w.WriteHeader(400)
					log.Println(err)
					io.WriteString(w, "Invalid code")
					return
				}
				session.Set("OAuthBearerToken", tok.AccessToken)
				session.Set("OAuthAuthenticatedUsername", session.Get("ClaimedUsername"))
				if err := sessionDB.SaveSession(session); err != nil {
					w.WriteHeader(500)
					return
				}
				http.Redirect(w, r, pagesRoot, http.StatusSeeOther)
				return
			}
			var b bytes.Buffer
			if err := loginTemplate.Execute(&b, ""); err != nil {
				w.WriteHeader(500)
				return
			}
			if err := sessionDB.SaveSession(session); err != nil {
				w.WriteHeader(500)
				return
			}
			w.WriteHeader(200)
			pageTemplate.Execute(w, PageTemplateData{Title: "Login", Content: template.HTML(string(b.Bytes()))})
			return
		case "POST":
			if err := r.ParseForm(); err != nil {
				w.WriteHeader(400)
				io.WriteString(w, "Invalid form data")
				return
			}
			username := r.Form.Get("username")
			session.Set("ClaimedUsername", username)

			userRe := regexp.MustCompile(`@(.+)@(.+)`)
			pieces := userRe.FindStringSubmatch(username)
			if pieces == nil {
				w.WriteHeader(400)
				io.WriteString(w, "Bad username")
				return
			}
			user := pieces[1]
			host := pieces[2]

			webfingerURI := fmt.Sprintf("https://%s/.well-known/webfinger?resource=%s@%s", host, user, host)
			resp, err := http.Get(webfingerURI)
			if err != nil {
				log.Println(err)
				w.WriteHeader(400)
				io.WriteString(w, "Bad username")
				return
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Println(err)
				w.WriteHeader(500)
				io.WriteString(w, "Error")
				return
			}
			fmt.Println(string(body))

			var webFingerResponse struct {
				Subject string   `json:"subject"`
				Aliases []string `json:"aliases"`
				Links   []struct {
					Rel  string `json:"rel"`
					Type string `json:"type"`
					Href string `json:"href"`
				} `json:"links"`
			}
			var actorID string
			if err := json.Unmarshal(body, &webFingerResponse); err != nil {
				log.Println(err)
				w.WriteHeader(400)
				io.WriteString(w, "Bad username")
				return
			}
			for _, link := range webFingerResponse.Links {
				if link.Type == "application/activity+json" && link.Rel == "self" {
					actorID = link.Href
					break
				}

			}
			fmt.Println(actorID)
			parsedActor, err := url.Parse(actorID)
			if err != nil {
				log.Println(err)
				w.WriteHeader(400)
				io.WriteString(w, "Bad username")
			}
			client, err := clientDB.GetClient(parsedActor.Hostname())
			if err != nil {
				log.Println("Registering app for " + parsedActor.Hostname())

				c, err := registerApp(clientDB, parsedActor.Hostname())
				if err != nil {
					log.Println(err)
					w.WriteHeader(400)
					fmt.Fprintf(w, "Could not register app with %s", parsedActor.Hostname())
					return
				}
				client = c
			}
			oauthconf := oauth2.Config{
				ClientID:     client.ClientId,
				ClientSecret: client.ClientSecret,
				Scopes:       []string{"read:accounts"},
				Endpoint: oauth2.Endpoint{
					AuthURL:  "https://" + parsedActor.Hostname() + "/oauth/authorize",
					TokenURL: "https://" + parsedActor.Hostname() + "/oauth/token",
				},
				RedirectURL: client.RedirectURI,
			}

			var stateRand [60]byte
			if _, err := rand.Read(stateRand[:]); err != nil {
				w.WriteHeader(500)
				return
			}
			theState := base64.URLEncoding.EncodeToString(stateRand[:])

			session.Set("OAuthState", theState)
			session.Set("OAuthHost", parsedActor.Hostname())
			if err := sessionDB.SaveSession(session); err != nil {
				w.WriteHeader(500)
				return
			}
			linkURL := oauthconf.AuthCodeURL(theState)
			http.Redirect(w, r, linkURL, http.StatusSeeOther)
		default:
			w.WriteHeader(405)
			// Should be PUT, but html is stupid and doesn't let us send a PUT request from a form
			// without javascript
			w.Header().Add("Allow", "GET,POST")
			io.WriteString(w, "Invalid method")
		}
	}
}

func registerApp(db OAuthClientStore, hostname string) (OAuthClient, error) {
	// FIXME: Guess the type of host. For now, assuming Mastodon and
	// the mastodon api to register the app.
	// Ideally we'd use a less vendor-specific API
	fmt.Println("Registering app")
	registerURL := "https://" + hostname + "/api/v1/apps"
	values := make(url.Values)
	values.Set("client_name", "Fediwiki")
	values.Set("redirect_uris", "http://localhost:3333/login")
	values.Set("scopes", "read read:accounts")
	req, err := http.PostForm(registerURL, values)
	if err != nil {
		return OAuthClient{}, err
	}
	defer req.Body.Close()
	resp, err := io.ReadAll(req.Body)
	if err != nil {
		return OAuthClient{}, err
	}
	fmt.Println(string(resp))

	var appRegisterResponse OAuthClient
	if err := json.Unmarshal(resp, &appRegisterResponse); err != nil {
		return appRegisterResponse, err
	}
	fmt.Println(&appRegisterResponse)

	if err := db.StoreClient(hostname, appRegisterResponse); err != nil {
		return appRegisterResponse, err
	}
	fmt.Printf("%v", appRegisterResponse)
	return appRegisterResponse, nil
}

func logoutHandler(sessionDB session.Store) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("URL", r.URL.Path)
		session, err := session.Start(sessionDB, w, r)
		if err != nil {
			http.Redirect(w, r, pagesRoot, http.StatusSeeOther)
			return
		}
		if err := sessionDB.DestroySession(session); err != nil {
			w.WriteHeader(500)
			log.Println(err)
			fmt.Fprintf(w, "Something went wrong.\n")
			return
		}
		http.Redirect(w, r, pagesRoot, http.StatusSeeOther)
	}
}
