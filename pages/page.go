package pages

import (
	"fmt"
	"os"
	"regexp"
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
	GetPageNotes(pagename string) ([]activitypub.Note, error)
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

func GetPageNameFromActorId(url string) (string, error) {
	re := regexp.MustCompile("https://" + os.Getenv("fediwikidomain") + "/pages/(.+)/actor")
	matches := re.FindStringSubmatch(url)
	if matches == nil {
		return "", fmt.Errorf("Unknown page %s", url)

	}
	return matches[1], nil

}
