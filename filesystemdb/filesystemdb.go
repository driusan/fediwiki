package filesystemdb

import (
	"crypto"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"path/filepath"

	"fediwiki/activitypub"
	"fediwiki/httpsig"
	"fediwiki/oauth"
	"fediwiki/pages"
	"fediwiki/session"

	"github.com/mischief/ndb"
)

var NotFound error = errors.New("Not Found")
var BadId error = errors.New("Bad id")

type FileSystemDB struct {
	FSRoot string
}

func (db *FileSystemDB) GetPage(pagename string) (*pages.Page, error) {
	var p pages.Page
	if pagename == "" {
		return nil, fmt.Errorf("No page name")
	}
	filesdir := filepath.Join(db.FSRoot, "pages", pagename)
	if !strings.HasPrefix(filesdir, db.FSRoot+"/pages") {
		return nil, fmt.Errorf("Invalid page name")
	}
	latest, err := os.ReadFile(filepath.Join(filesdir, "latest"))
	if err == nil {
		filesdir = filepath.Join(filesdir, "history", string(latest))
	}
	if _, err := os.Stat(filepath.Join(filesdir, "content.md")); errors.Is(err, os.ErrNotExist) {
		return nil, NotFound
	}
	content, err := os.ReadFile(filepath.Join(filesdir, "content.md"))
	if err != nil {
		return nil, err
	}
	p.Content = string(content)

	if _, err := os.Stat(filepath.Join(filesdir, "title.txt")); err == nil {
		content, err := os.ReadFile(filepath.Join(filesdir, "title.txt"))
		if err != nil {
			return nil, err
		}
		p.Title = string(content)
	}
	if _, err := os.Stat(filepath.Join(filesdir, "summary.md")); err == nil {
		content, err := os.ReadFile(filepath.Join(filesdir, "summary.md"))
		if err != nil {
			return nil, err
		}
		p.Summary = string(content)
	}

	return &p, nil
}
func (db *FileSystemDB) GetPageRevision(pagename, revision string) (*pages.Page, error) {
	var p pages.Page
	if pagename == "" {
		return nil, fmt.Errorf("No page name")
	}
	filesdir := filepath.Join(db.FSRoot, "pages", pagename, "history", revision)
	if !strings.HasPrefix(filesdir, db.FSRoot+"/pages") {
		return nil, fmt.Errorf("Invalid page name")
	}
	if _, err := os.Stat(filepath.Join(filesdir, "content.md")); errors.Is(err, os.ErrNotExist) {
		return nil, NotFound
	}
	content, err := os.ReadFile(filepath.Join(filesdir, "content.md"))
	if err != nil {
		return nil, err
	}
	p.Content = string(content)

	if _, err := os.Stat(filepath.Join(filesdir, "title.txt")); err == nil {
		content, err := os.ReadFile(filepath.Join(filesdir, "title.txt"))
		if err != nil {
			return nil, err
		}
		p.Title = string(content)
	}
	if _, err := os.Stat(filepath.Join(filesdir, "summary.md")); err == nil {
		content, err := os.ReadFile(filepath.Join(filesdir, "summary.md"))
		if err != nil {
			return nil, err
		}
		p.Summary = string(content)
	}

	return &p, nil
}
func (db *FileSystemDB) GetPageRevisionParent(pagename, revision string) (*pages.Page, error) {
	return nil, fmt.Errorf("Not implemented")
}
func (db *FileSystemDB) SavePage(p pages.Page, prof activitypub.Actor, editor string) (*pages.Revision, error) {
	if p.PageName == "" {
		return nil, fmt.Errorf("No page name")
	}
	basedir := filepath.Join(db.FSRoot, "pages", p.PageName)

	var idrand [36]byte
	if _, err := rand.Read(idrand[:]); err != nil {
		return nil, err
	}
	savedir := filepath.Join(basedir, "history", base64.URLEncoding.EncodeToString(idrand[:]))

	if err := os.MkdirAll(savedir, 0777); err != nil {
		return nil, err
	}
	content := string(p.Content)
	content = strings.Replace(content, "\r\n", "\n", -1)
	content = strings.Replace(content, "\n\r", "\n", -1)
	content = strings.Replace(content, "\r", "\n", -1)
	if err := os.WriteFile(savedir+"/content.md", []byte(content), 0664); err != nil {
		return nil, err
	}
	content = string(p.Summary)
	content = strings.Replace(content, "\r\n", "\n", -1)
	content = strings.Replace(content, "\n\r", "\n", -1)
	content = strings.Replace(content, "\r", "\n", -1)
	if err := os.WriteFile(savedir+"/summary.md", []byte(content), 0664); err != nil {
		return nil, err
	}
	if err := os.WriteFile(savedir+"/title.txt", []byte(p.Title), 0664); err != nil {
		return nil, err
	}
	var parentstring string
	if bytes, err := os.ReadFile(filepath.Join(basedir, "latest")); err == nil {
		parentstring = string(bytes)
		if err := os.WriteFile(filepath.Join(savedir, "parentversion"), bytes, 0664); err != nil {
			return nil, err
		}
	}

	f, err := os.OpenFile(filepath.Join(basedir, "revisions.db"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	savetime := time.Now()
	if parentstring != "" {
		fmt.Fprintf(f, "id=%s time=%s editor=%s parent=%s pagename=%s\n", base64.URLEncoding.EncodeToString(idrand[:]), savetime.Format(time.RFC3339), editor, parentstring, p.PageName)
	} else {
		fmt.Fprintf(f, "id=%s time=%s editor=%s pagename=%s\n", base64.URLEncoding.EncodeToString(idrand[:]), savetime.Format(time.RFC3339), editor, p.PageName)
	}

	if err := os.WriteFile(basedir+"/latest", []byte(base64.URLEncoding.EncodeToString(idrand[:])), 0664); err != nil {
		return nil, err
	}
	return &pages.Revision{
		PageName:   p.PageName,
		RevisionID: base64.URLEncoding.EncodeToString(idrand[:]),
		Editor:     editor,
		EditTime:   &savetime,
	}, nil
}

func (db *FileSystemDB) GetClient(hostname string) (oauth.Client, error) {
	oauthdb, err := ndb.Open(db.FSRoot + "/oauthclients.db")
	if err != nil {
		return oauth.Client{}, fmt.Errorf("No clients registered")
	}

	records := oauthdb.Search("hostname", hostname)
	if len(records) > 1 {
		panic("too many hostnames")
	}
	if len(records) == 0 {
		return oauth.Client{}, fmt.Errorf("No client for %s", hostname)
	}
	var client oauth.Client
	for _, tuple := range records[0] {
		switch tuple.Attr {
		case "remoteid":
			client.Id = tuple.Val
		case "remotename":
			client.Name = tuple.Val
		case "website":
			client.Website = tuple.Val
		case "redirect_uri":
			client.RedirectURI = tuple.Val
		case "client_id":
			client.ClientId = tuple.Val
		case "client_secret":
			client.ClientSecret = tuple.Val

		}
	}
	return client, nil
}

func (db *FileSystemDB) StoreClient(hostname string, c oauth.Client) error {
	if _, err := db.GetClient(hostname); err == nil {
		return fmt.Errorf("%s already registered", hostname)
	}
	filename := db.FSRoot + "/oauthclients.db"
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "hostname=%s remoteid=%s remotename=%s website=%s redirect_uri=%s client_id=%s client_secret=%s\n", hostname, c.Id, c.Name, c.Website, c.RedirectURI, c.ClientId, c.ClientSecret); err != nil {
		return err
	}
	return nil
}

func (db *FileSystemDB) GetSession(id string) (*session.Session, error) {
	dir := db.FSRoot + "/sessions/" + id
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return nil, NotFound
	}

	sess := &session.Session{Id: id, Values: make(map[string]string)}
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		bytes, err := os.ReadFile(dir + "/" + file.Name())
		if err != nil {
			return nil, err
		}
		sess.Values[file.Name()] = string(bytes)
	}
	return sess, nil
}

func (db *FileSystemDB) SaveSession(s *session.Session) error {
	if s == nil {
		return fmt.Errorf("No session")
	}
	dir := db.FSRoot + "/sessions/" + s.Id
	if err := os.MkdirAll(dir, 0700); err != nil {
		log.Println(err)
		return err
	}
	for key, value := range s.Values {
		dir := db.FSRoot + "/sessions/" + s.Id
		if err := os.WriteFile(dir+"/"+key, []byte(value), 0600); err != nil {
			return err
		}
	}
	return nil
}

func (db *FileSystemDB) DestroySession(s *session.Session) error {
	dir := filepath.Join(db.FSRoot, "sessions", s.Id)
	if !strings.HasPrefix(dir, db.FSRoot+"/sessions/") || dir == db.FSRoot+"/sessions" || dir == db.FSRoot+"/sessions/" {
		return fmt.Errorf("Invalid session ID")
	}
	return os.RemoveAll(dir)
}
func (db *FileSystemDB) GetPageRevisions(pagename string) ([]pages.Revision, error) {
	pagesdb, err := ndb.Open(filepath.Join(db.FSRoot, "pages", pagename, "revisions.db"))
	if err != nil {
		return nil, err
	}
	var result []pages.Revision = nil

	ndbpages := pagesdb.Search("pagename", pagename)
	for _, rowtuple := range ndbpages {
		var rev pages.Revision
		for _, tuple := range rowtuple {
			switch tuple.Attr {
			case "id":
				rev.RevisionID = tuple.Val
			case "time":
				if t, err := time.Parse(time.RFC3339, tuple.Val); err != nil {
					log.Println(err)
				} else {
					rev.EditTime = &t
				}
			case "editor":
				rev.Editor = tuple.Val
			case "pagename":
				rev.PageName = tuple.Val
			}

		}
		result = append(result, rev)
	}
	return result, nil
}
func (d *FileSystemDB) HasObject(id string) bool {
	odb, err := ndb.Open(filepath.Join(d.FSRoot, "objects", "objects.db"))
	if err != nil {
		return false
	}
	records := odb.Search("id", id)
	return len(records) > 0
}

func (d *FileSystemDB) UpdateObject(obj activitypub.Object) error {
	if !d.HasObject(obj.Id) {
		return d.SaveObject(obj)
	}
	odb, err := ndb.Open(filepath.Join(d.FSRoot, "objects", "objects.db"))
	if err != nil {
		return err
	}
	records := odb.Search("id", obj.Id)
	if len(records) != 1 {
		return fmt.Errorf("Could not update object")
	}
	for _, tuple := range records[0] {
		if tuple.Attr == "cachepath" {
			if err := os.WriteFile(filepath.Join(d.FSRoot, "objects", tuple.Val), obj.RawBytes, 0644); err != nil {
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("Could not find object cachepath")
}

func (d *FileSystemDB) GetObject(id string) (*activitypub.Object, error) {
	odb, err := ndb.Open(filepath.Join(d.FSRoot, "objects", "objects.db"))
	if err != nil {
		return nil, err
	}
	record := odb.Search("id", id)
	if len(record) != 1 {
		return nil, fmt.Errorf("Wrong number of records. Want 1 got %v", len(record))
	}
	var rv activitypub.Object
	for _, tuple := range record[0] {
		switch tuple.Attr {
		case "id":
			rv.Id = tuple.Val
		case "type":
			rv.Type = tuple.Val
		case "cachepath":
			bytes, err := os.ReadFile(filepath.Join(d.FSRoot, "objects", tuple.Val))
			if err != nil {
				return nil, err
			}
			rv.RawBytes = bytes
		}
	}
	return &rv, nil

}
func (d *FileSystemDB) SaveObject(obj activitypub.Object) error {
	if !strings.HasPrefix(obj.Id, "https://") {
		return BadId
	}
	path := base64.URLEncoding.EncodeToString([]byte(obj.Id))

	if err := os.MkdirAll(filepath.Join(d.FSRoot, "objects", filepath.Dir(path)), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(d.FSRoot, "objects", path), obj.RawBytes, 0644); err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(d.FSRoot, "objects", "objects.db"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		return err
	}
	defer f.Close()

	record := fmt.Sprintf("\nid=%s type=%s cachepath=%s\n", obj.Id, obj.Type, path)
	if _, err := f.WriteString(record); err != nil {
		return err
	}
	return nil
}

func (d *FileSystemDB) GetKey(keyid string) (crypto.PublicKey, error) {
	ndbDir := filepath.Join(d.FSRoot, "keys")
	ndb, err := ndb.Open(filepath.Join(ndbDir, "knownkeys.db"))
	if err != nil {
		return nil, err
	}

	records := ndb.Search("keyid", keyid)
	if len(records) == 0 {
		return nil, fmt.Errorf("No records found")
	}

	var data []byte
	var owner string
	for _, record := range records {
		for _, tuple := range record {
			switch tuple.Attr {
			case "owner":
				owner = tuple.Val
			case "cachepath":
				filedata, err := os.ReadFile(filepath.Join(ndbDir, tuple.Val))
				if err != nil {
					return nil, err
				}
				data = filedata
			default:
			}
		}
		if owner != "" && data != nil {
			break
		}
	}
	return httpsig.ParsePemKey(keyid, owner, data, nil)
}

func (d *FileSystemDB) SaveKey(keyid, owner string, pembytes []byte) error {
	ndbDir := filepath.Join(d.FSRoot, "keys")
	if err := os.MkdirAll(ndbDir, 0775); err != nil {
		return err
	}
	f, err := os.OpenFile(
		filepath.Join(ndbDir, "knownkeys.db"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return err
	}
	defer f.Close()
	keyfilename := base64.URLEncoding.EncodeToString([]byte(keyid))
	fullkeyfilename := filepath.Join(ndbDir, keyfilename)
	if err := os.WriteFile(fullkeyfilename, pembytes, 0644); err != nil {
		return err
	}
	record := fmt.Sprintf("\nkeyid=%s owner=%s cachepath=%s\n", keyid, owner, keyfilename)

	if _, err := f.WriteString(record); err != nil {
		return err
	}
	return nil
}

func (d *FileSystemDB) GetForeignActor(id string) (*activitypub.Actor, error) {
	actordb, err := ndb.Open(filepath.Join(d.FSRoot, "actors.db"))
	if err != nil {
		return nil, err
	}
	records := actordb.Search("id", id)
	if len(records) == 0 {
		return nil, NotFound
	}
	if len(records) > 1 {
		return nil, fmt.Errorf("Too many records in database")
	}
	var cachepath string
	// The tuple has most of the things we need, but it doesn't have the key
	// so we just look for cachepath and parse the whole thing
	for _, tuple := range records[0] {
		switch tuple.Attr {
		case "cachepath":
			cachepath = tuple.Val
			break
		}
	}
	if cachepath == "" {
		return nil, NotFound
	}
	f, err := os.Open(filepath.Join(d.FSRoot, "actors", cachepath))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	bytes, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	var actor activitypub.Actor
	if err := json.Unmarshal(bytes, &actor); err != nil {
		return nil, err
	}
	log.Println("Got actor from filesystem")
	return &actor, nil
}

func (d *FileSystemDB) StoreActor(actor activitypub.Actor, raw []byte) error {
	filename := filepath.Join(d.FSRoot, "actors.db")
	cachedir := filepath.Join(d.FSRoot, "actors")
	if err := os.MkdirAll(cachedir, 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		return err
	}
	defer f.Close()

	record := fmt.Sprintf("\nid=%s type=%s\n", actor.Id, actor.Type)
	record = fmt.Sprintf("%s\tinbox=%s outbox=%s\n", record, actor.Inbox, actor.Outbox)
	record = fmt.Sprintf("%s\tfollowing=%s followers=%s\n", record, actor.Following, actor.Followers)
	record = fmt.Sprintf("%s\tpreferredUsername=%s\n", record, actor.PreferredUsername)
	record = fmt.Sprintf("%s\tname=%s\n", record, actor.Name)
	if actor.ProfileIcon != "" {
		record = fmt.Sprintf("%s\tprofileIcon=%s\n", record, actor.ProfileIcon)
	}
	fname := base64.URLEncoding.EncodeToString([]byte(actor.Id))
	record = fmt.Sprintf("%s\tcachepath=%s", record, fname)

	if err := os.WriteFile(filepath.Join(cachedir, fname), raw, 0644); err != nil {
		return err
	}
	if _, err := f.WriteString(record); err != nil {
		return err
	}
	return nil
}
