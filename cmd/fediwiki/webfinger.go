package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"fediwiki/pages"
)

func webFingerHandler(actorsdb pages.PagesDatabase) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		resource := r.URL.Query().Get("resource")
		log.Println("WebFinger for ", resource)
		if resource == "" {
			w.WriteHeader(400)
			fmt.Fprintf(w, "Missing resource")
			return
		}
		if !strings.HasPrefix(resource, "acct:") {
			w.WriteHeader(400)
			fmt.Fprintf(w, "Invalid resource")
			return
		}
		acct := strings.TrimPrefix(resource, "acct:")
		domain := os.Getenv("fediwikidomain")
		if !strings.HasSuffix(acct, domain) {
			notFound(w, r)
			return

		}
		name := strings.TrimSuffix(acct, "@"+domain)
		if prof, err := actorsdb.GetPageActor(name); err != nil {
			notFound(w, r)
			return
		} else {
			val, err := json.Marshal(struct {
				Subject string   `json:"subject"`
				Aliases []string `json:"aliases"`
				Links   []struct {
					Rel  string `json:"rel"`
					Type string `json:"type"`
					Href string `json:"href"`
				} `json:"links"`
			}{
				resource,
				[]string{prof.Id},
				[]struct {
					Rel  string `json:"rel"`
					Type string `json:"type"`
					Href string `json:"href"`
				}{
					{
						Rel:  "self",
						Type: "application/activity+json",
						Href: prof.Id,
					},
				},
			})
			if err != nil {
				log.Println(err)
				w.WriteHeader(500)
				return
			}
			w.Header().Set("Content-Type", "application/jrd+json")
			w.WriteHeader(200)
			fmt.Fprintf(w, "%s", string(val))
		}
	}
}
