package api

import (
	"io/fs"
	"net/http"

	"pipeline-horn/web"

	"github.com/gin-gonic/gin"
)

// spaHandler serves the embedded Vite build. Requests for existing static
// assets are served directly; all other paths fall back to index.html so
// that the React SPA can handle client-side routing.
func spaHandler() gin.HandlerFunc {
	subFS, err := fs.Sub(web.StaticFiles, "dist")
	if err != nil {
		panic("web: failed to sub into dist: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(subFS))

	return func(c *gin.Context) {
		path := c.Request.URL.Path[1:] // strip leading /
		if path == "" {
			path = "index.html"
		}

		f, err := subFS.Open(path)
		if err != nil {
			// File not found → serve index.html for SPA routing
			c.Request.URL.Path = "/"
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}
		_ = f.Close()
		fileServer.ServeHTTP(c.Writer, c.Request)
	}
}
