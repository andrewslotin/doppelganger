package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andrewslotin/doppelganger/git"
)

type MirrorHandler struct {
	githubRepos      git.RepositoryService
	mirroredRepos    git.MirrorService
	trackRepoService git.TrackingService
}

func NewMirrorHandler(githubRepos git.RepositoryService, mirroredRepos git.MirrorService, trackingService git.TrackingService) *MirrorHandler {
	return &MirrorHandler{
		githubRepos:      githubRepos,
		mirroredRepos:    mirroredRepos,
		trackRepoService: trackingService,
	}
}

func (handler *MirrorHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()

	repoName := req.FormValue("repo")
	if repoName == "" {
		http.Error(w, "Missing source repository name", http.StatusBadRequest)
		return
	}

	switch action := strings.ToLower(req.FormValue("action")); action {
	case "create":
		if err := handler.CreateMirror(w, repoName); err != nil {
			log.Printf("failed to create mirror %s: %s", repoName, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		log.Printf("mirrored %s [%s]", repoName, time.Since(startTime))
		if req.FormValue("notrack") != "" {
			handler.redirectToRepository(w, req, repoName)
			return
		}

		// Redirect to /mirror?action=track&repo=<repoName> to set up webhook
		q := req.Form
		q.Set("action", "track")
		req.URL.RawQuery = q.Encode()
		http.Redirect(w, req, req.URL.String(), http.StatusSeeOther)
	case "update":
		if err := handler.UpdateMirror(w, repoName); err != nil {
			log.Printf("failed to update mirror %s: %s", repoName, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		log.Printf("updated mirror %s [%s]", repoName, time.Since(startTime))
		handler.redirectToRepository(w, req, repoName)
	case "track":
		if handler.trackRepoService == nil {
			http.Error(w, "Tracking changes not supported", http.StatusNotImplemented)
			return
		}

		if err := handler.SetupChangeTracking(w, req, repoName); err != nil {
			log.Printf("failed to track changes for mirror %s: %s", repoName, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		log.Printf("set up push changes hook for %s [%s]", repoName, time.Since(startTime))
		handler.redirectToRepository(w, req, repoName)
	default:
		http.Error(w, fmt.Sprintf("Unsupported action %q", action), http.StatusBadRequest)
	}
}

func (handler *MirrorHandler) CreateMirror(w http.ResponseWriter, repoName string) error {
	switch repo, err := handler.githubRepos.Get(repoName); err {
	case nil:
		return handler.mirroredRepos.Create(repo.FullName, repo.GitURL)
	case git.ErrorNotFound:
		http.Error(w, "Source repository not found", http.StatusNotFound)
		return nil
	default:
		return err
	}
}

func (handler *MirrorHandler) SetupChangeTracking(w http.ResponseWriter, req *http.Request, repoName string) error {
	switch repo, err := handler.mirroredRepos.Get(repoName); err {
	case nil:
		return handler.trackRepoService.Track(repo.FullName, apiHookURL(req.Host, req.TLS != nil).String())
	case git.ErrorNotMirrored:
		http.Error(w, "Repository not mirrored", http.StatusNotFound)
		return nil
	default:
		return err
	}
}

func (handler *MirrorHandler) UpdateMirror(w http.ResponseWriter, repoName string) error {
	switch repo, err := handler.mirroredRepos.Get(repoName); err {
	case nil:
		return handler.mirroredRepos.Update(repo.FullName)
	case git.ErrorNotMirrored:
		http.Error(w, "Source repository not mirrored", http.StatusNotFound)
		return nil
	default:
		return err
	}
}

func (handler *MirrorHandler) redirectToRepository(w http.ResponseWriter, req *http.Request, repoName string) {
	http.Redirect(w, req, fmt.Sprintf("/mirrored?repo=%s", url.QueryEscape(repoName)), http.StatusSeeOther)
}

func apiHookURL(host string, isSSL bool) *url.URL {
	scheme := "http"
	if isSSL {
		scheme = "https"
	}

	return &url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   "/apihook",
	}
}
