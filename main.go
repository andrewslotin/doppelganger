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
	"github.com/bmizerany/pat"
)

var (
	// Version and BuildDate are used in help message and set by Makefile
	Version   = "n/a"
	BuildDate = "n/a"

	args struct {
		version   bool
		addr      string
		port      int
		mirrorDir string
	}
)

func init() {
	flag.BoolVar(&args.version, "version", false, "Print version and exit")
	flag.StringVar(&args.addr, "addr", "", "Listen address")
	flag.IntVar(&args.port, "port", 8081, "Listen port")
	flag.StringVar(&args.mirrorDir, "mirror", filepath.Join(os.Getenv("GOPATH"), "src", "github.com"), "Mirrored repositories directory")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(2)
	}
}

func main() {
	flag.Parse()
	if args.version {
		fmt.Printf("Doppelganger, version %s, build date %s\n", Version, BuildDate)
		os.Exit(0)
	}

	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags)

	token := os.Getenv("DOPPELGANGER_GITHUB_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "Missing GitHub access token (set DOPPELGANGER_GITHUB_TOKEN environment variable)")
		os.Exit(-1)
	}

	repositoryService, err := git.NewGithubRepositories(context.WithValue(context.Background(), git.GithubToken, token))
	if err != nil {
		log.Fatal(err)
	}

	gitCmd, err := git.SystemGit()
	if err != nil {
		log.Fatal(err)
	}
	mirroredRepositoryService := git.NewMirroredRepositories(args.mirrorDir, gitCmd)

	mux := pat.New()
	mux.Get("/favicon.ico", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./assets/favicon.ico")
	}))

	mux.Get("/", NewReposHandler(repositoryService))
	mux.Get("/:owner/:repo", NewRepoHandler(repositoryService))
	mux.Get("/mirror", NewReposHandler(mirroredRepositoryService))
	mux.Post("/mirror", NewMirrorHandler(repositoryService, mirroredRepositoryService, repositoryService))
	mux.Get("/mirror/:owner/:repo", NewRepoHandler(mirroredRepositoryService))
	mux.Get("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))

	// GitHub webhooks
	mux.Post("/apihook", NewWebhookHandler(mirroredRepositoryService))

	addr := fmt.Sprintf("%s:%d", args.addr, args.port)
	log.Printf("doppelganger is listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Panic(err)
	}
}
