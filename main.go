package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"

	"golang.org/x/net/context"

	"github.com/andrewslotin/doppelganger/git"
	"github.com/andrewslotin/doppelganger/server"
	"github.com/bmizerany/pat"
)

var (
	// Version is used in help message and logs and set by Makefile
	Version = "n/a"
	// BuildDate is used in help message and set by Makefile
	BuildDate = "n/a"
	// PrivateKeyPath is a path to the SSH used by git+ssh
	PrivateKeyPath = path.Join(os.Getenv("HOME"), ".ssh", "id_rsa")

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

	mux.Get("/:owner/:repo.tar.gz", NewDownloadMirrorHandler(mirroredRepositoryService))
	mux.Get("/:owner/:repo", NewRepoHandler(mirroredRepositoryService))
	mux.Get("/", NewReposHandler(mirroredRepositoryService, true))
	mux.Get("/src/:owner/:repo", NewRepoHandler(repositoryService))
	mux.Get("/src/", NewReposHandler(repositoryService, false))
	mux.Post("/mirror", NewMirrorHandler(repositoryService, mirroredRepositoryService, repositoryService))
	mux.Get("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))

	// GitHub webhooks
	mux.Post("/apihook", NewWebhookHandler(mirroredRepositoryService))

	srv := server.New(args.addr, args.port)
	if err := srv.Run(mux); err != nil {
		log.Panic(err)
	}
	log.Printf("doppelganger %s is listening on %s", Version, srv.Addr)

	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	select {
	case <-signals:
		log.Println("shutdown signal received, terminating...")
		if err := srv.Shutdown(); err != nil {
			log.Fatal(err)
		}
	}
}
