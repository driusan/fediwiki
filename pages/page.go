package pages

import (
	"fmt"
	"os"
	"time"

	"fediwiki/activitypub"
)

const Root = "/pages/"

type Page struct {
	PageName string
	Title    string
	Summary  string
	Content  string
}

type Revision struct {
	PageName   string
	RevisionID string
	Editor     string
	EditTime   *time.Time
}

type Persister interface {
	GetPage(pagename string) (*Page, error)
	SavePage(page Page, pageactor activitypub.Actor, editor string) (*Revision, error)
	GetPageRevisions(pagename string) ([]Revision, error)
	GetPageRevision(pagename, revisionid string) (*Page, error)
	GetPageRevisionParent(pagename, revisionid string) (*Page, error)
}

func (r Revision) DiffNote(diff string) activitypub.Note {
	id := fmt.Sprintf("https://%s%s%s/history/%s/diff", os.Getenv("fediwikidomain"), Root, r.PageName, r.RevisionID)
	summary := fmt.Sprintf("Page Changes for %v", r.PageName)
	note := activitypub.Note{
		BaseProperties: activitypub.BaseProperties{
			Context: []interface{}{"https://www.w3.org/ns/activitystreams"},
			Id:      id,
			Type:    "Note",
		},
		Summary:      &summary,
		Published:    r.EditTime,
		Url:          id,
		MediaType:    "text/plain",
		Content:      diff,
		AttributedTo: fmt.Sprintf("https://%s%s%s/actor", os.Getenv("fediwikidomain"), Root, r.PageName),
		To:           []string{"https://www.w3.org/ns/activitystreams#Public"},
		Cc:           []string{fmt.Sprintf("https://%s%s%s/followers", os.Getenv("fediwikidomain"), Root, r.PageName)},
	}
	return note
}
