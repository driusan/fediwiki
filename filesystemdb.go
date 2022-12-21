package main

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"path/filepath"

	"fediwiki/session"

	"github.com/mischief/ndb"
)

var NotFound error = errors.New("Not Found")
var BadId error = errors.New("Bad id")

type FileSystemDB struct {
	FSRoot string
}

func (db *FileSystemDB) GetPageActor(page string) (*Profile, error) {
	filename := filepath.Join(db.FSRoot, pagesRoot, page, "actor.json")
	if !strings.HasPrefix(filename, db.FSRoot+pagesRoot) {
		// They were trying to use ../.. or something to escape the
		// filesystem
		return nil, fmt.Errorf("Invalid page name")
	}
	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		return nil, NotFound
	}
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var p Profile
	if err := json.Unmarshal(bytes, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (db *FileSystemDB) NewPageActor(p Page, domain string, private crypto.PrivateKey, public crypto.PublicKey) (*Profile, error) {
	pageurl := "https://" + domain + pagesRoot + p.PageName
	id := pageurl + "/actor"
	keybytes, err := x509.MarshalPKIXPublicKey(public)
	if err != nil {
		return nil, err
	}
	block := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: keybytes})
	prof := &Profile{
		Context:           JsonLDContext{"https://www.w3.org/ns/activitystreams", "https://w3id.org/security/v1"},
		Id:                id,
		Type:              "Service",
		PreferredUsername: p.PageName,
		Title:             p.Title,
		Summary:           p.Summary,
		Inbox:             pageurl + "/inbox",
		Outbox:            pageurl + "/outbox",
		PublicKey: PublicKey{
			Id:           id + "#main-key",
			Owner:        id,
			PublicKeyPem: string(block),
		},
	}
	filedir := filepath.Join(db.FSRoot, pagesRoot, p.PageName)
	if !strings.HasPrefix(filedir, db.FSRoot+pagesRoot) {
		// Make sure no ones trying to escape with a ../../ or something
		return nil, fmt.Errorf("Unknown error creating directory")
	}
	if err := os.MkdirAll(filedir, 0775); err != nil {
		return nil, err
	}

	filename := filepath.Join(filedir, "actor.json")

	log.Println(filename)
	bytes, err := json.Marshal(prof)
	if err != nil {
		return nil, err
	}
	privkeybytes, err := x509.MarshalPKIXPublicKey(private)
	if err := os.WriteFile(filepath.Join(filedir, "private.pem"), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privkeybytes}), 0400); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filename, bytes, 0664); err != nil {
		return nil, err
	}
	return prof, nil
}

func (db *FileSystemDB) GetPage(pagename string) (*Page, error) {
	var p Page
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
func (db *FileSystemDB) GetPageRevision(pagename, revision string) (*Page, error) {
	var p Page
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
func (db *FileSystemDB) SavePage(p Page, prof Profile, editor string) error {
	if p.PageName == "" {
		return fmt.Errorf("No page name")
	}
	basedir := filepath.Join(db.FSRoot, "pages", p.PageName)

	var idrand [36]byte
	if _, err := rand.Read(idrand[:]); err != nil {
		return err
	}
	savedir := filepath.Join(basedir, "history", base64.URLEncoding.EncodeToString(idrand[:]))

	if err := os.MkdirAll(savedir, 0777); err != nil {
		return err
	}
	content := string(p.Content)
	content = strings.Replace(content, "\r\n", "\n", -1)
	content = strings.Replace(content, "\n\r", "\n", -1)
	content = strings.Replace(content, "\r", "\n", -1)
	if err := os.WriteFile(savedir+"/content.md", []byte(content), 0664); err != nil {
		return err
	}
	content = string(p.Summary)
	content = strings.Replace(content, "\r\n", "\n", -1)
	content = strings.Replace(content, "\n\r", "\n", -1)
	content = strings.Replace(content, "\r", "\n", -1)
	if err := os.WriteFile(savedir+"/summary.md", []byte(content), 0664); err != nil {
		return err
	}
	if err := os.WriteFile(savedir+"/title.txt", []byte(p.Title), 0664); err != nil {
		return err
	}
	var parentstring string
	if bytes, err := os.ReadFile(filepath.Join(basedir, "latest")); err == nil {
		parentstring = string(bytes)
		if err := os.WriteFile(filepath.Join(savedir, "parentversion"), bytes, 0664); err != nil {
			return err
		}
	}

	f, err := os.OpenFile(filepath.Join(basedir, "revisions.db"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		return err
	}
	defer f.Close()

	if parentstring != "" {
		fmt.Fprintf(f, "id=%s time=%s editor=%s parent=%s pagename=%s\n", base64.URLEncoding.EncodeToString(idrand[:]), time.Now().Format(time.RFC3339), editor, parentstring, p.PageName)
	} else {
		fmt.Fprintf(f, "id=%s time=%s editor=%s pagename=%s\n", base64.URLEncoding.EncodeToString(idrand[:]), time.Now().Format(time.RFC3339), editor, p.PageName)
	}

	if err := os.WriteFile(basedir+"/latest", []byte(base64.URLEncoding.EncodeToString(idrand[:])), 0664); err != nil {
		return err
	}
	return nil
}

func (db *FileSystemDB) GetClient(hostname string) (OAuthClient, error) {
	oauthdb, err := ndb.Open(db.FSRoot + "/oauthclients.db")
	if err != nil {
		return OAuthClient{}, fmt.Errorf("No clients registered")
	}

	records := oauthdb.Search("hostname", hostname)
	if len(records) > 1 {
		panic("too many hostnames")
	}
	if len(records) == 0 {
		return OAuthClient{}, fmt.Errorf("No client for %s", hostname)
	}
	var client OAuthClient
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

func (db *FileSystemDB) StoreClient(hostname string, c OAuthClient) error {
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
func (db *FileSystemDB) GetPageRevisions(pagename string) ([]Revision, error) {
	pagesdb, err := ndb.Open(filepath.Join(db.FSRoot, "pages", pagename, "revisions.db"))
	if err != nil {
		return nil, err
	}
	var result []Revision = nil

	pages := pagesdb.Search("pagename", pagename)
	for _, rowtuple := range pages {
		var rev Revision
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
func (d *FileSystemDB) SaveObject(id, body string) error {
	if !strings.HasPrefix(id, "https://") {
		return BadId
	}
	path := strings.TrimPrefix(id, "https://")
	dir := filepath.Join(d.FSRoot, "objects", filepath.Dir(path))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(d.FSRoot, "objects", path), []byte(body), 0644); err != nil {
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
	return parsePemKey(keyid, owner, data, nil)
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
