package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var (
	addr = flag.String("addr", "", "Listen address.")
	port = flag.Int("port", 8081, "Listen port.")
)

func main() {
	flag.Parse()

	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags)

	token := os.Getenv("DOPPELGANGER_GITHUB_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "Missing GitHub access token (set DOPPELGANGER_GITHUB_TOKEN environment variable)")
		os.Exit(-1)
	}

	*addr = fmt.Sprintf("%s:%d", *addr, *port)
	log.Printf("doppelganger is listening on %s", *addr)
	http.ListenAndServe(*addr, nil)
}

func newGithubClient(token string) *github.Client {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token,
	})
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)

	return github.NewClient(oauthClient)
}
