package httpapi

import (
	"net/http"
	"path"
	"path/filepath"
)

// spa serves the built frontend from dir. Existing files are served as-is;
// any other path gets index.html so client-side routes (/body, /trends) work
// on reload.
func spa(dir string) http.Handler {
	root := http.Dir(dir)
	files := http.FileServer(root)
	index := filepath.Join(dir, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f, err := root.Open(path.Clean("/" + r.URL.Path)); err == nil {
			st, statErr := f.Stat()
			f.Close()
			if statErr == nil && !st.IsDir() {
				files.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, index)
	})
}
