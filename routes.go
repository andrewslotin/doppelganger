package main

import "net/http"

func FetchRepoFromRequest(req *http.Request) (string, bool) {
	owner, repo := req.URL.Query().Get(":owner"), req.URL.Query().Get(":repo")
	if owner == "" || repo == "" {
		return "", false
	}

	return owner + "/" + repo, true
}
