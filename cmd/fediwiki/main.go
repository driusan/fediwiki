package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/http/cgi"
	"os"
	"regexp"
	"sort"
	"strings"

	"fediwiki/activitypub"
	"fediwiki/filesystemdb"
	"fediwiki/httpsig"
	"fediwiki/inbox"
	"fediwiki/pages"
	"fediwiki/session"

	"golang.org/x/crypto/acme/autocert"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

const frontPage = "FrontPage"

var pageTemplate, editTemplate, loggedInHeader *template.Template

type PageTemplateData struct {
	Title   string
	Header  template.HTML
	Content template.HTML
}

func internalError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(500)
	io.WriteString(w, "Internal Server error\n")
}
func badRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(400)
	io.WriteString(w, "Bad Request\n")
}

func notFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(404)
	io.WriteString(w, "Not found\n")
}

func notImplemented(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(501)
	io.WriteString(w, "Not implemented\n")
}

func getHeader(s *session.Session, pagename string) template.HTML {
	var b bytes.Buffer
	if s != nil {
		if user := s.Get("OAuthAuthenticatedUsername"); user != "" {
			if err := loggedInHeader.Execute(&b, struct{ PageName, Username string }{pagename, user}); err != nil {
				panic(err)
			}
			return template.HTML(b.Bytes())
		}
	}
	if err := loginTemplate.Execute(&b, struct{ PageName string }{pagename}); err != nil {
		panic(err)
	}
	return template.HTML(b.Bytes())
}

func createPage(session *session.Session, page string, adb pages.PagesDatabase, db pages.Persister, w http.ResponseWriter, r *http.Request) {
	var b bytes.Buffer
	if err := editTemplate.Execute(&b, pages.Page{}); err != nil {
		w.WriteHeader(500)
		return
	}
	pageTemplate.Execute(
		w,
		PageTemplateData{
			Title:   page,
			Header:  getHeader(session, page),
			Content: template.HTML(b.Bytes()),
		},
	)
	return
}

func hasEditPermission(session *session.Session) bool {
	if session == nil {
		return false
	}
	return session.Get("OAuthAuthenticatedUsername") != ""
}

func pagehistory(session *session.Session, pagename string, historydb pages.Persister, w http.ResponseWriter, r *http.Request) {
	revs, err := historydb.GetPageRevisions(pagename)
	if err != nil {
		notFound(w, r)
		return
	}

	var b bytes.Buffer
	fmt.Fprintf(&b, "<ul>\n")
	sort.Slice(revs, func(i, j int) bool {
		return revs[i].EditTime.After(*(revs[j].EditTime))
	})
	for _, rev := range revs {
		fmt.Fprintf(&b, "<li><a href=\"%s%s/history/%s\">%v</a>: edited by %v</li>\n", pages.Root, pagename, rev.RevisionID, rev.EditTime, rev.Editor)
	}
	fmt.Fprintf(&b, "</ul>")
	pageTemplate.Execute(
		w,
		PageTemplateData{
			Title:   "History of " + pagename,
			Header:  getHeader(session, pagename),
			Content: template.HTML(string(b.Bytes())),
		},
	)
}

func renderPage(page pages.Page) template.HTML {
	contentparser := parser.NewWithExtensions(parser.CommonExtensions)
	summaryrenderer := html.NewRenderer(html.RendererOptions{Flags: html.CommonFlags | html.SkipHTML})
	contentrenderer := html.NewRenderer(html.RendererOptions{Flags: html.CommonFlags | html.SkipHTML | html.TOC})

	federatedLink := regexp.MustCompile(`\[\[([[:alpha:]]+)@([[:alpha:]\.]+)\]\]`)
	internalLink := regexp.MustCompile(`\[\[([[:alpha:]]+)\]\]`)

	content := string(markdown.ToHTML([]byte(page.Content), contentparser, contentrenderer))
	content = federatedLink.ReplaceAllString(content, `<a href="https://$2/`+pages.Root+` "/$1">$1 ($2)</a>`)
	content = internalLink.ReplaceAllString(content, `<a href="`+pages.Root+`/$1">$1</a>`)
	var summary string
	if page.Summary != "" {
		summaryparser := parser.NewWithExtensions(parser.CommonExtensions)
		summary = string(markdown.ToHTML([]byte(page.Summary), summaryparser, summaryrenderer))
		summary = federatedLink.ReplaceAllString(summary, `<a href="https://$2/`+pages.Root+` "/$1">$1 ($2)</a>`)
		summary = internalLink.ReplaceAllString(summary, `<a href="`+pages.Root+`/$1">$1</a>`)
	}
	return template.HTML(summary + content)
}
func wikipagerev(session *session.Session, pagename, rev string, db pages.Persister, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		page, err := db.GetPageRevision(pagename, rev)
		if err != nil {
			notFound(w, r)
			return
		}
		content := renderPage(*page)
		pageTemplate.Execute(
			w,
			PageTemplateData{
				Title:   page.Title,
				Header:  getHeader(session, pagename),
				Content: content,
			},
		)
	default:
		w.WriteHeader(405)
		w.Header().Add("Allow", "GET")
		io.WriteString(w, "Invalid method")
	}
}
func wikipage(session *session.Session, pagename string, adb pages.PagesDatabase, db pages.Persister, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		page, err := db.GetPage(pagename)
		if err != nil {
			w.WriteHeader(404)
			createPage(session, pagename, adb, db, w, r)
			return
		}
		if r.URL.Query().Get("edit") == "true" {
			var b bytes.Buffer
			if err := editTemplate.Execute(&b, page); err != nil {
				w.WriteHeader(500)
				io.WriteString(w, err.Error())
				return
			}
			pageTemplate.Execute(
				w,
				PageTemplateData{
					Title:   page.Title,
					Header:  getHeader(session, pagename),
					Content: template.HTML(string(b.Bytes())),
				},
			)
			return

		}
		w.WriteHeader(200)
		content := renderPage(*page)

		pageTemplate.Execute(
			w,
			PageTemplateData{
				Title:   page.Title,
				Header:  getHeader(session, pagename),
				Content: content,
			},
		)
		return
	case "POST":
		if hasEditPermission(session) != true {
			w.WriteHeader(403)
			io.WriteString(w, "Permission denied")
			return
		}
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(400)
			io.WriteString(w, "Invalid form data")
			return
		}
		page := pages.Page{
			PageName: pagename,
			Title:    r.Form.Get("title"),
			Summary:  r.Form.Get("summary"),
			Content:  r.Form.Get("content"),
		}
		pageactor, err := adb.GetPageActor(pagename)
		if err == filesystemdb.NotFound {
			key, err := rsa.GenerateKey(rand.Reader, 4096)
			if err != nil {
				log.Println(err)
				w.WriteHeader(500)
				io.WriteString(w, "Internal server error")
				return
			}
			a2, err := adb.NewPageActor(page, r.Host, key, &key.PublicKey)
			if err != nil {
				log.Println(err)
				w.WriteHeader(500)
				io.WriteString(w, "Internal server error")
				return
			}
			pageactor = a2
		}
		if err := db.SavePage(page, *pageactor, session.Get("OAuthAuthenticatedUsername")); err != nil {
			log.Println(err)
			w.WriteHeader(500)
			io.WriteString(w, "Internal server error")
		}
		http.Redirect(w, r, pages.Root+page.PageName, 303)
	default:
		w.WriteHeader(405)
		// Should be PUT, but html is stupid and doesn't let us send a PUT request from a form
		// without javascript
		w.Header().Add("Allow", "GET,POST")
		io.WriteString(w, "Invalid method")
	}
}

func rootPage(pagesdb pages.PagesDatabase, pagedb pages.Persister, sessionDB session.Store, keystore httpsig.KeyStore, objectDB activitypub.ObjectDatabase, actorDb activitypub.ActorDatabase, activityDb activitypub.ActivityDatabase, prefix string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.URL.Path)
		sess, err := session.Start(sessionDB, w, r)
		if err != nil {
			log.Println(err)
		}
		// 1: to get rid of the leading slash.
		urlPieces := strings.Split(strings.TrimPrefix(r.URL.Path, prefix), "/")
		switch len(urlPieces) {
		case 0:
			wikipage(sess, frontPage, pagesdb, pagedb, w, r)
			return
		case 1:
			if urlPieces[0] == "" {
				wikipage(sess, frontPage, pagesdb, pagedb, w, r)
				return
			}
			wikipage(sess, urlPieces[0], pagesdb, pagedb, w, r)
			return
		case 2:
			switch urlPieces[1] {
			case "actor":
				act, err := pagesdb.GetPageActor(urlPieces[0])
				if err != nil {
					log.Println(err)
					// FIXME: Be smarter about error
					notImplemented(w, r)
					return
				}
				val, err := json.Marshal(act)
				if err != nil {
					log.Println(err)
					// FIXME: Be smarter about error
					notImplemented(w, r)
					return
				}
				accept := strings.Split(r.Header.Get("Accept"), ",")
				log.Println(r.Header.Get("Accept"))
				log.Println(accept)
				setContent := false
				for _, val := range accept {
					switch strings.TrimSpace(val) {
					case "application/activity+json":
						w.Header().Set("Content-Type", `application/activity+json`)
						log.Println("Set activity+json")
						setContent = true
					case `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`:
						w.Header().Set("Content-Type", `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`)
						setContent = true
					case "application/ld+json":
						w.Header().Set("Content-Type", `application/ld+json`)
						log.Println("Set ld+json")
						setContent = true
					case "*/*":
						w.Header().Set("Content-Type", `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`)
						setContent = true
					}
					if setContent {
						break
					}
				}
				if setContent == false {
					w.Header().Set("Content-Type", `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`)
				}
				w.Write(val)
				return
			case "inbox":
				switch r.Method {
				case "GET":
					_, err := pagesdb.GetPageActor(urlPieces[0])

					if err != nil {
						if err == filesystemdb.NotFound {
							notFound(w, r)
						} else {
							log.Println(w, r)
							internalError(w, r)
						}
						return
					}
				case "POST":
					if err := httpsig.Validate(r, keystore); err != nil {
						log.Println(err)
						badRequest(w, r)
						fmt.Fprintf(w, "Could not validate http signature\n")
						return
					}
					bytes, err := io.ReadAll(r.Body)
					if err != nil {
						log.Println(err)
						internalError(w, r)
						return
					}

					var inbound activitypub.Object
					if err := json.Unmarshal(bytes, &inbound); err != nil {
						log.Println(err)
						badRequest(w, r)
						return
					}
					inbound.RawBytes = bytes
					if err := inbox.Process(objectDB, pagesdb, actorDb, activityDb, inbound); err != nil {
						if err == filesystemdb.BadId {
							badRequest(w, r)
							return
						}
						log.Println(err)
						internalError(w, r)
						return
					}
					w.WriteHeader(201)
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprintf(w, `{ "okay" : "accepted" }`)
				}

				return
			case "outbox":
				notImplemented(w, r)
				return
			case "history":
				pagehistory(sess, urlPieces[0], pagedb, w, r)
				return
			case "talk":
				notImplemented(w, r)
				return
			default:
				notFound(w, r)
			}
		case 3:
			if urlPieces[1] != "history" {
				notFound(w, r)
				return
			}
			wikipagerev(sess, urlPieces[0], urlPieces[2], pagedb, w, r)
		default:
			notFound(w, r)
		}
	}
}

func redirectToPagesRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, pages.Root+r.URL.Path, http.StatusSeeOther)
}
func main() {
	mux := http.NewServeMux()
	pageTemplate = template.Must(template.New("MainPage").Parse(`
    <html>
    <title>{{.Title}}</title>
    <style>
    header {
        display: flex;
        justify-content: space-around;
    }

    header div {
        flex: 3;
    }
    header nav {
        display: flex;
        flex: 1;
    }
    header nav ul {
        list-style: none;
        display: flex;
        flex: 1;
        marign: 0;
        padding: 0;
        gap: 1em;
    }

    header nav.actions, header nav.actions ul {
        justify-content: right;
    }
    main {
        margin-left: 5em;
    }
    main h1 {
        text-align: center;
    }

    main nav ul:before {
        content: "Jump to section:";
    }
 
    main nav ul {
        list-style: none;
        padding-left: 1ex;
        margin: 0;
    }

    main nav ul li {
        list-style: none;
        padding-left: 1em;
    }


   main nav {
        background: rgb(230, 230, 230);
        width: 50%;
        padding: 1ex;
        border: thin solid black;
        border-radius: 1ex;

    }
    </style>
    <body>
        {{.Header}}
        <main>
            <h1>{{.Title}}</h1>
            <article>
                {{.Content}}
            </article>
        </main>
    </body>
    </html>
    `))
	editTemplate = template.Must(template.New("EditPage").Parse(`
        <form method="post">
            <fieldset>
                <div>
                    <h2>Page Title</h2>
                    <input name="title" value="{{.Title}}" />
                </div>
                <div>
                    <h2>Page Summary</h2>
                    <p>(A sentence or up to a paragraph describing this page before the table of contents.)</p>
                    <textarea cols="80" rows="10" name="summary">{{.Summary}}</textarea>
                </div>
                <div>
                    <h2>Content</h2>
                    <p>(The rest of the content to display after the summary.)</p>
                    <textarea cols="80" rows="24" name="content">{{.Content}}</textarea>
                </div>
                <div>
                <input type="Submit" value="Save" />
                </div>
            </fieldset>
        </form>
    `))
	loginTemplate = template.Must(template.New("LoginForm").Parse(`
        <header>
            <nav>
                <ul>
                    <li><a href="` + pages.Root + `">Home</a></li>
                    <li><a href="` + pages.Root + `{{.PageName}}/history">Page history</a></li>
                </ul>
            </nav>

            <div>
                <form method="post" action="/login/">
                    <fieldset>
                        Not logged in. You can use an existing fediverse account on a server compatible with the Mastodon OAuth API to login.
                        Username: <input name="username" placeholder="@example@example.com" />
                    </fieldset>
                </form>
            </div>
        </header>

    `))
	loggedInHeader = template.Must(template.New("LoggedInHeader").Parse(`
        <header>
            <nav>
                <ul>
                    <li><a href="` + pages.Root + `">Home</a></li>
                    <li><a href="` + pages.Root + `{{.PageName}}/history">Page history</a></li>
                </ul>
            </nav>
            <div>Logged in as {{.Username}}</div>
            <nav class="actions">
                <ul>
                    <li><a href="?edit=true">Edit Page</li>
                    <li><a href="/logout">Logout</a></li>
                </ul>
            </nav>
        </header>
    `))
	var db filesystemdb.FileSystemDB
	if root := os.Getenv("fediwikiroot"); root != "" {
		db.FSRoot = root
	} else {
		log.Fatal("Missing fediwikiroot")
	}
	domain := os.Getenv("fediwikidomain")
	if domain == "" {
		log.Fatal("Missing fediwikidomain")
	}
	mux.HandleFunc("/.well-known/webfinger", webFingerHandler(&db))
	mux.HandleFunc(pages.Root, rootPage(&db, &db, &db, &db, &db, &db, &db, pages.Root))
	mux.HandleFunc("/login/", loginHandler(&db, &db))
	mux.HandleFunc("/logout", logoutHandler(&db))
	mux.HandleFunc("/", redirectToPagesRoot)

	if os.Getenv("FEDIWIKI_CGI") == "true" {
		if err := cgi.Serve(nil); err != nil {
			log.Println(err)
		}
	} else {
		log.Println("Starting server")
		log.Fatal(http.Serve(autocert.NewListener(domain), mux))
	}
}
