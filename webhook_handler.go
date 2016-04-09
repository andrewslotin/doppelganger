package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/andrewslotin/doppelganger/git"
)

// WebhookHandler is a type that implements http.Handler interface and is used by HTTP server to handle GitHub webhooks sent to "/apihook".
// Currently "ping" and "push" events are supported.
//
// For more details on webhooks see https://developer.github.com/webhooks/.
type WebhookHandler struct {
	mirroredRepos git.MirrorService
}

// NewWebhookHandler creates and initializes an instance of WebhookHandler.
func NewWebhookHandler(mirroredRepos git.MirrorService) *WebhookHandler {
	return &WebhookHandler{
		mirroredRepos: mirroredRepos,
	}
}

func (handler *WebhookHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()
	defer req.Body.Close()

	switch event := req.Header.Get("X-Github-Event"); event {
	case "ping":
		fmt.Fprint(w, "PONG")
	case "push":
		switch repo, err := handler.UpdateRepo(req); err {
		case nil:
			log.Printf("updated %s [%s]", repo.FullName, time.Since(startTime))
			fmt.Fprint(w, "OK")
		case git.ErrorNotFound, git.ErrorNotMirrored:
			http.Error(w, "Not found", http.StatusNotFound)
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	default:
		http.Error(w, fmt.Sprintf("Unsupported event %q", event), http.StatusBadRequest)
	}
}

// UpdateRepo handles "push" event and updates existing GitHub repository mirrors synchronizing it with remote.
func (handler *WebhookHandler) UpdateRepo(req *http.Request) (repo *git.Repository, err error) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("failed to read request body (%s)", err)
		return nil, err
	}

	var updateEvent struct {
		Ref        string `json:"ref"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}

	if err := json.Unmarshal(body, &updateEvent); err != nil {
		log.Printf("failed to parse push event payload %s", string(body))
		return nil, err
	}

	repo, err = handler.mirroredRepos.Get(updateEvent.Repository.FullName)
	if err != nil {
		log.Printf("failed to find mirrored copy of %s (%s)", updateEvent.Repository.FullName, err)
		return nil, err
	}

	updatedBranch := strings.TrimPrefix(updateEvent.Ref, "refs/heads/")
	if repo.Master != updatedBranch {
		log.Printf("skip push event to %s (mirrored ref %s, received %s)", repo.FullName, repo.Master, updatedBranch)
		return repo, nil
	}

	return repo, handler.mirroredRepos.Update(repo.FullName)
}
