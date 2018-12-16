package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var indexBaseName = "index" + templateFileNameSuffix

type fileHandler struct {
	route  string
	path   string
	render func(string) (string, error)
}

func (f *fileHandler) serveStatus(w http.ResponseWriter, r *http.Request, status int) {
	w.WriteHeader(status)
	w.Write([]byte(http.StatusText(status)))
}

func (f *fileHandler) serveDir(w http.ResponseWriter, r *http.Request, dirPath string) {
	indexPath := filepath.Join(dirPath, indexBaseName)
	f.serveTemplate(w, r, indexPath)
}

func (f *fileHandler) serveTemplate(w http.ResponseWriter, r *http.Request, path string) {
	template, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		log.Printf("read %q: %v", path, err)
		f.serveStatus(w, r, http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("read %q: %v", path, err)
		f.serveStatus(w, r, http.StatusInternalServerError)
		return
	}
	rendered, err := f.render(string(template))
	if err != nil {
		log.Printf("render %q: %v", path, err)
		f.serveStatus(w, r, http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, filepath.Base(path), time.Time{}, strings.NewReader(rendered))
}

// ServeHTTP is http.Handler.ServeHTTP
func (f *fileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	path = strings.TrimPrefix(path, f.route)
	path = strings.TrimPrefix(path, "/"+f.route)
	path = filepath.Clean(path)
	path = filepath.Join(f.path, path)
	info, err := os.Stat(path)
	switch {
	case os.IsNotExist(err):
		f.serveStatus(w, r, http.StatusNotFound)
	case os.IsPermission(err):
		f.serveStatus(w, r, http.StatusForbidden)
	case err != nil:
		f.serveStatus(w, r, http.StatusInternalServerError)
	case info.IsDir():
		f.serveDir(w, r, path)
	case strings.HasSuffix(path, templateFileNameSuffix):
		f.serveTemplate(w, r, path)
	default:
		http.ServeFile(w, r, path)
	}
}
