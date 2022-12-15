package main

import (
"os"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
    "net/http/cgi"
	"regexp"
	"strings"

	"fediwiki/session"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

const pagesRoot = "/pages/"
const frontPage = "FrontPage"

var pageTemplate, editTemplate, loggedInHeader *template.Template

type PageTemplateData struct {
	Title   string
	Header  template.HTML
	Content template.HTML
}

func notFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(404)
	io.WriteString(w, "Not found\n")
}

func notImplemented(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(501)
	io.WriteString(w, "Not implemented\n")
}

func getHeader(s *session.Session) template.HTML {
	if user := s.Get("OAuthAuthenticatedUsername"); user != "" {
		var b bytes.Buffer
		if err := loggedInHeader.Execute(&b, user); err != nil {
			panic(err)
		}
		return template.HTML(b.Bytes())
	}
	var b bytes.Buffer
	if err := loginTemplate.Execute(&b, ""); err != nil {
		panic(err)
	}
	return template.HTML(b.Bytes())
}

func createPage(session *session.Session, page string, db PagePersister, w http.ResponseWriter, r *http.Request) {
	var b bytes.Buffer
	if err := editTemplate.Execute(&b, Page{}); err != nil {
		w.WriteHeader(500)
		return
	}
	pageTemplate.Execute(
		w,
		PageTemplateData{
			Title:   page,
			Header:  getHeader(session),
			Content: template.HTML(b.Bytes()),
		},
	)
	return
}

func hasEditPermission(session *session.Session) bool {
    if session == nil {
        return false;
    }
	return session.Get("OAuthAuthenticatedUsername") != "" 
}
func wikipage(session *session.Session, pagename string, db PagePersister, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		page, err := db.GetPage(pagename)
		if err != nil {
			w.WriteHeader(404)
			createPage(session, pagename, db, w, r)
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
					Header:  getHeader(session),
					Content: template.HTML(string(b.Bytes())),
				},
			)
			return

		}
		contentparser := parser.NewWithExtensions(parser.CommonExtensions)
		summaryrenderer := html.NewRenderer(html.RendererOptions{Flags: html.CommonFlags | html.SkipHTML})
		contentrenderer := html.NewRenderer(html.RendererOptions{Flags: html.CommonFlags | html.SkipHTML | html.TOC})

		federatedLink := regexp.MustCompile(`\[\[([[:alpha:]]+)@([[:alpha:]\.]+)\]\]`)
		internalLink := regexp.MustCompile(`\[\[([[:alpha:]]+)\]\]`)

		content := string(markdown.ToHTML([]byte(page.Content), contentparser, contentrenderer))
		content = federatedLink.ReplaceAllString(content, `<a href="https://$2/` + pagesRoot + ` "/$1">$1 ($2)</a>`)
		content = internalLink.ReplaceAllString(content, `<a href="` + pagesRoot + `/$1">$1</a>`)
        var summary string
        if page.Summary != "" {
            summaryparser := parser.NewWithExtensions(parser.CommonExtensions)
            summary = string(markdown.ToHTML([]byte(page.Summary), summaryparser, summaryrenderer))
            summary = federatedLink.ReplaceAllString(summary, `<a href="https://$2/` + pagesRoot + ` "/$1">$1 ($2)</a>`)
            summary = internalLink.ReplaceAllString(summary, `<a href="` + pagesRoot + `/$1">$1</a>`)
        }
		w.WriteHeader(200)
		// We don't check the error of execute because we've already written to ResponseWriter
		// s it's too late to change it to a 500 error

		pageTemplate.Execute(
			w,
			PageTemplateData{
				Title:   page.Title,
				Header:  getHeader(session),
				Content: template.HTML(summary + content),
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
		page := Page{
			PageName: pagename,
			Title:    r.Form.Get("title"),
			Summary:  r.Form.Get("summary"),
			Content:  r.Form.Get("content"),
		}
		fmt.Println("Saving page", pagename)
		if err := db.SavePage(page); err != nil {
			log.Println(err)
			w.WriteHeader(500)
			io.WriteString(w, "Internal server error")
		}
		http.Redirect(w, r, pagesRoot+page.PageName, 303)
	default:
		w.WriteHeader(405)
		// Should be PUT, but html is stupid and doesn't let us send a PUT request from a form
		// without javascript
		w.Header().Add("Allow", "GET,POST")
		io.WriteString(w, "Invalid method")
	}
}

func rootPage(actordb ActorPersister, pagedb PagePersister, sessionDB session.Store, prefix string) func(w http.ResponseWriter, r *http.Request) {
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
			wikipage(sess, frontPage, pagedb, w, r)
			return
		case 1:
			if urlPieces[0] == "" {
				wikipage(sess, frontPage, pagedb, w, r)
				return
			}
			wikipage(sess, urlPieces[0], pagedb, w, r)
			return
		case 2:
			switch urlPieces[1] {
			case "id":
				act, err := actordb.GetPageActor(urlPieces[0])
				if err != nil {
					// FIXME: Be smarter about error
					notImplemented(w, r)
					return
				}
				val, err := json.Marshal(act)
				if err != nil {
					// FIXME: Be smarter about error
					notImplemented(w, r)
					return
				}
				w.Write(val)
				return
			case "inbox":
				notImplemented(w, r)
				return
			case "outbox":
				notImplemented(w, r)
				return
			case "history":
				notImplemented(w, r)
				return
			case "talk":
				notImplemented(w, r)
				return
			default:
				notFound(w, r)
			}
		default:
			notFound(w, r)
		}
	}
}

func redirectToPagesRoot(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, pagesRoot + r.URL.Path, http.StatusSeeOther)
}
func main() {
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
    }

    header nav.actions, header nav.actions ul {
        justify-content: right;
        gap: 1em;
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
                    <li><a href="` + pagesRoot + `">Home</a></li>
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
                    <li><a href="` + pagesRoot + `">Home</a></li>
                </ul>
            </nav>
            <div>Logged in as {{.}}</div>
            <nav class="actions">
                <ul>
                    <li><a href="?edit=true">Edit Page</li>
                    <li><a href="/logout">Logout</a></li>
                </ul>
            </nav>
        </header>
    `))
	db := FileSystemDB{"/home/driusan/testwiki"}
	http.HandleFunc(pagesRoot, rootPage(&db, &db, &db, pagesRoot))
	http.HandleFunc("/login/", loginHandler(&db, &db))
	http.HandleFunc("/logout", logoutHandler(&db))
    http.HandleFunc("/", redirectToPagesRoot)

    if os.Getenv("FEDIWIKI_CGI") == "true" {
        if err := cgi.Serve(nil); err != nil {
            log.Println(err)
        }
    } else {
        log.Println("Starting server")
            if err := http.ListenAndServe(":3333", nil); err != nil {
                log.Fatal(err)
            }
    }
}
