package do

import (
	"context"

	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

// NewClient builds and returns a new DO client for a given access token string.
func NewClient(ctx context.Context, accessToken string) *godo.Client {
	tokenSource := &tokenSource{
		AccessToken: accessToken,
	}
	oauthClient := oauth2.NewClient(ctx, tokenSource)

	return godo.NewClient(oauthClient)
}

// tokenSource implements an interface for retrieving credentials for an oauth2 client.
type tokenSource struct {
	AccessToken string
}

// Token builds and returns the oauth2 Token structure for this TokenSource.
func (t *tokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}
