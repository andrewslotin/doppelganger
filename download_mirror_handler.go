package main

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/andrewslotin/doppelganger/git"
)

type DownloadMirrorHandler struct {
	mirroredRepos *git.MirroredRepositories
}

func NewDownloadMirrorHandler(mirroredRepos *git.MirroredRepositories) *DownloadMirrorHandler {
	return &DownloadMirrorHandler{
		mirroredRepos: mirroredRepos,
	}
}

func (handler *DownloadMirrorHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	repoName, ok := FetchRepoFromRequest(req)
	if !ok {
		WriteNotFoundPage(w, "No such repository", "")
		return
	}

	clonePath, err := ioutil.TempDir(os.TempDir(), "cloned")
	if err != nil {
		WriteErrorPage(w, err, http.StatusInternalServerError)
		return
	}

	if err := os.MkdirAll(clonePath, 0755); err != nil {
		WriteErrorPage(w, err, http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(clonePath)

	err = handler.mirroredRepos.Clone(repoName, clonePath)
	if err != nil {
		WriteErrorPage(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	_ = compressDir(w, clonePath)
}

func compressDir(w io.Writer, dirPath string) error {
	gz := gzip.NewWriter(w)
	defer gz.Close()

	tarball := tar.NewWriter(gz)
	defer tarball.Close()

	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}

		header.Name = relPath
		if err := tarball.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		fd, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fd.Close()

		_, err = io.Copy(tarball, fd)

		return err
	})

	return nil
}
