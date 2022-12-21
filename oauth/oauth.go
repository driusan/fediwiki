package oauth

type Client struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	Website      string `json:"website"`
	RedirectURI  string `json:"redirect_uri"`
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type ClientStore interface {
	GetClient(hostname string) (Client, error)
	StoreClient(hostname string, c Client) error
}
