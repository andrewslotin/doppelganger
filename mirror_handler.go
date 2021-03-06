package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andrewslotin/doppelganger/git"
	"github.com/andrewslotin/doppelganger/git/gitssh"
	"golang.org/x/net/context"
)

var (
	privateRepoAccessTemplate = template.Must(template.ParseFiles("templates/layout.html.template", "templates/mirror/private_repo_access.html.template"))
)

// MirrorHandler is a type that implements http.Handler interface and is used to handle requests to "/mirror".
// An action that needs to be executed is defined by "action" form variable. Target repository is specified by its name passed in
// request parameter "name".
//
//   // Create a new mirror of andrewslotin/doppelganger
//   curl http://doppelganger/mirror?action=create&name=andrewslotin/doppelganger
//   // Update an existing mirror of andrewslotin/doppelganger
//   curl http://doppelganger/mirror?action=update&name=andrewslotin/doppelganger
//   // Set up tracking of changes in andrewslotin/doppelganger
//   curl http://doppelganger/mirror?action=track&name=andrewslotin/doppelganger
type MirrorHandler struct {
	githubRepos      git.RepositoryService
	mirroredRepos    git.MirrorService
	trackRepoService git.TrackingService
}

// NewMirrorHandler creates and initializes a new handler.
func NewMirrorHandler(githubRepos git.RepositoryService, mirroredRepos git.MirrorService, trackingService git.TrackingService) *MirrorHandler {
	return &MirrorHandler{
		githubRepos:      githubRepos,
		mirroredRepos:    mirroredRepos,
		trackRepoService: trackingService,
	}
}

func (handler *MirrorHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()
	ctx := req.Context()

	repoName, ok := handler.fetchRepoFromRequest(req)
	if !ok {
		WriteErrorPage(w, UserError{Message: "Missing source repository name", BackURL: req.Referer()}, http.StatusBadRequest)
		return
	}

	switch action := strings.ToLower(req.FormValue("action")); action {
	case "create":
		if err := handler.CreateMirror(ctx, w, repoName); err != nil {
			if err == git.ErrorNotFound {
				err = handler.ShowPrivateRepoAccessPage(w, repoName, action)
				if err == nil {
					return
				}

				log.Printf("failed to obtain public key: %s", err)
			} else {
				log.Printf("failed to create mirror %s: %s", repoName, err)
			}

			userErr := UserError{
				Message:       "Internal server error",
				BackURL:       req.Referer(),
				OriginalError: err,
			}
			WriteErrorPage(w, userErr, http.StatusInternalServerError)

			return
		}

		if req.FormValue("notrack") == "" && handler.trackRepoService != nil {
			if err := handler.SetupChangeTracking(ctx, w, req, repoName); err != nil {
				if err == git.ErrorNotMirrored {
					WriteNotFoundPage(w, fmt.Sprintf("Repository %s was not mirrored yet", repoName), "/src/"+repoName)
				} else {
					log.Printf("failed to track changes for mirror %s: %s", repoName, err)
					userErr := UserError{
						Message:       "Failed to set up push web hook, please check logs for details",
						BackURL:       req.Referer(),
						OriginalError: err,
					}
					WriteErrorPage(w, userErr, http.StatusInternalServerError)
				}

				return
			}
		}

		log.Printf("mirrored %s [%s]", repoName, time.Since(startTime))
		handler.redirectToRepository(w, req, repoName)
	case "update":
		if err := handler.UpdateMirror(ctx, w, repoName); err != nil {
			if err == git.ErrorNotMirrored {
				WriteNotFoundPage(w, fmt.Sprintf("Repository %s was not mirrored yet", repoName), "/src/"+repoName)
			} else {
				log.Printf("failed to update mirror %s: %s", repoName, err)
				userErr := UserError{
					Message:       "Internal server error",
					BackURL:       req.Referer(),
					OriginalError: err,
				}
				WriteErrorPage(w, userErr, http.StatusInternalServerError)
			}

			return
		}

		log.Printf("updated mirror %s [%s]", repoName, time.Since(startTime))
		handler.redirectToRepository(w, req, repoName)
	case "track":
		if handler.trackRepoService == nil {
			WriteErrorPage(w, UserError{Message: "Tracking changes not supported", BackURL: req.Referer()}, http.StatusNotImplemented)
			return
		}

		if err := handler.SetupChangeTracking(ctx, w, req, repoName); err != nil {
			if err == git.ErrorNotMirrored {
				WriteNotFoundPage(w, fmt.Sprintf("Repository %s was not mirrored yet", repoName), "/src/"+repoName)
			} else {
				log.Printf("failed to track changes for mirror %s: %s", repoName, err)
				userErr := UserError{
					Message:       "Failed to set up push web hook, please check logs for details",
					BackURL:       req.Referer(),
					OriginalError: err,
				}
				WriteErrorPage(w, userErr, http.StatusInternalServerError)
			}

			return
		}

		log.Printf("set up push changes hook for %s [%s]", repoName, time.Since(startTime))
		handler.redirectToRepository(w, req, repoName)
	default:
		WriteErrorPage(w, UserError{Message: fmt.Sprintf("Unsupported action %q", action), BackURL: req.Referer()}, http.StatusBadRequest)
	}
}

// CreateMirror searches for a repository in githubRepos and creates its mirror.
func (handler *MirrorHandler) CreateMirror(ctx context.Context, w http.ResponseWriter, repoName string) error {
	repo, err := handler.githubRepos.Get(ctx, repoName)
	if err != nil {
		return err
	}

	return handler.mirroredRepos.Create(ctx, repo.FullName, repo.GitURL)
}

// SetupChangeTracking searches for a repository in githubRepos and sets up changes tracker using trackingService.Track().
func (handler *MirrorHandler) SetupChangeTracking(ctx context.Context, w http.ResponseWriter, req *http.Request, repoName string) error {
	repo, err := handler.mirroredRepos.Get(ctx, repoName)
	if err != nil {
		return err
	}

	return handler.trackRepoService.Track(ctx, repo.FullName, apiHookURL(req.Host, req.TLS != nil).String())
}

// UpdateMirror updates an existing mirror synchronizing its with source.
func (handler *MirrorHandler) UpdateMirror(ctx context.Context, w http.ResponseWriter, repoName string) error {
	repo, err := handler.mirroredRepos.Get(ctx, repoName)
	if err != nil {
		return err
	}

	return handler.mirroredRepos.Update(ctx, repo.FullName)
}

// ShowPrivateRepoAccessPage renders a page with public SSH key that can be used for GitHub authentication.
func (handler *MirrorHandler) ShowPrivateRepoAccessPage(w http.ResponseWriter, repoName, action string) error {
	pubkey, err := handler.getPublicKey()
	if err != nil {
		return err
	}

	values := struct {
		PublicKey string
		FullName  string
		Action    string
	}{
		PublicKey: string(pubkey),
		FullName:  "andrewslotin/lol",
		Action:    "create",
	}

	return privateRepoAccessTemplate.Execute(w, values)
}

func (handler *MirrorHandler) getPublicKey() ([]byte, error) {
	pkey, err := gitssh.ReadPrivateRSAKey(PrivateKeyPath)
	if err != nil {
		log.Printf("writing a new RSA key to %s", PrivateKeyPath)
		pkey, err = gitssh.CreatePrivateRSAKey(PrivateKeyPath)
		if err != nil {
			return nil, err
		}
	}

	pubkey, err := gitssh.AuthorizedRSAKey(pkey)
	if err != nil {
		return nil, err
	}

	return pubkey, nil
}

func (handler *MirrorHandler) redirectToRepository(w http.ResponseWriter, req *http.Request, repoName string) {
	http.Redirect(w, req, "/"+repoName, http.StatusSeeOther)
}

func (handler *MirrorHandler) fetchRepoFromRequest(req *http.Request) (string, bool) {
	fullRepoName := req.FormValue("repo")
	return fullRepoName, fullRepoName != ""
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
