package web

import (
	"net/http"
	"path"
	"strings"
)

const defaultMafiaMiniAppDir = "web/mafia-mini-app/dist"

func (s *server) mafiaMiniApp(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/mafia-mini-app" {
		http.Redirect(w, r, "/mafia-mini-app/", http.StatusMovedPermanently)
		return
	}
	dir := strings.TrimSpace(s.cfg.Web.MafiaMiniAppDir)
	if dir == "" {
		dir = defaultMafiaMiniAppDir
	}

	fs := http.Dir(dir)
	requested := strings.TrimPrefix(r.URL.Path, "/mafia-mini-app/")
	if requested == "" {
		requested = "index.html"
	}
	requested = strings.TrimPrefix(path.Clean("/"+requested), "/")
	if serveMafiaMiniAppFile(w, r, fs, requested) {
		return
	}
	if serveMafiaMiniAppFile(w, r, fs, "index.html") {
		return
	}
	http.NotFound(w, r)
}

func serveMafiaMiniAppFile(w http.ResponseWriter, r *http.Request, fs http.FileSystem, name string) bool {
	file, err := fs.Open(name)
	if err != nil {
		return false
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		return false
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	return true
}
