package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/net/context"

	"github.com/andrewslotin/doppelganger/git"
)

// Version and BuildDate are used in help message and set by Makefile
const (
	Version   = "n/a"
	BuildDate = "n/a"
)

var (
	addr      = flag.String("addr", "", "Listen address")
	port      = flag.Int("port", 8081, "Listen port")
	mirrorDir = flag.String("mirror", filepath.Join(os.Getenv("GOPATH"), "src", "github.com"), "Mirrored repositories directory")
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Doppelganger, version %s, build date %s\n\nUsage: %s [OPTIONS]\n\nOptions:\n", Version, BuildDate, os.Args[0])
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

	repositoryService, err := git.NewGithubRepositories(context.WithValue(context.Background(), git.GithubToken, token))
	if err != nil {
		log.Panic(err)
	}
	mirroredRepositoryService := git.NewMirroredRepositories(*mirrorDir)

	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./assets/favicon.ico")
	})

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
