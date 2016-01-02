package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/andrewslotin/doppelganger/git"
)

var (
	addr      = flag.String("addr", "", "Listen address")
	port      = flag.Int("port", 8081, "Listen port")
	mirrorDir = flag.String("mirror", filepath.Join(os.Getenv("GOPATH"), "src", "github.com"), "Mirrored repositories directory")
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

	repositoryService := git.NewGithubRepositories(token)
	mirroredRepositoryService := git.NewMirroredRepositories(*mirrorDir)

	http.Handle("/", NewReposHandler(repositoryService))
	http.Handle("/mirrored", NewReposHandler(mirroredRepositoryService))
	http.Handle("/mirror", NewMirrorHandler(repositoryService, mirroredRepositoryService, repositoryService))
	http.Handle("/apihook", NewWebhookHandler(mirroredRepositoryService))

	*addr = fmt.Sprintf("%s:%d", *addr, *port)
	log.Printf("doppelganger is listening on %s", *addr)
	http.ListenAndServe(*addr, nil)
}
