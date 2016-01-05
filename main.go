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

	VERSION    = "n/a"
	BUILD_DATE = "n/a"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Doppelganger, version %s, build date %s\n\nUsage: %s [OPTIONS]\n\nOptions:\n", VERSION, BUILD_DATE, os.Args[0])
		flag.PrintDefaults()
		os.Exit(2)
	}
}

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
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))

	*addr = fmt.Sprintf("%s:%d", *addr, *port)
	log.Printf("doppelganger is listening on %s", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Panic(err)
	}
}
