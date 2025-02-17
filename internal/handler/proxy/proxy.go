package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"

	"github.com/rs/zerolog/log"

	"github.com/ix64/ollama-registry-proxy/internal/worker/cache_worker"
)

func Handler(p *httputil.ReverseProxy, cacheDir string, upstream url.URL) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Host = upstream.Host
		r.URL.Host = upstream.Host
		r.URL.Scheme = upstream.Scheme

		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			// Serve from upstream. Log
			log.Info().Str("component", "handler").Msgf("Method %s not supported, serving from upstream", r.Method)
			p.ServeHTTP(w, r)
			return
		}

		// Check if file with r.Path exists in cachedir. If yes, serve it
		cachePath := path.Join(cacheDir, r.URL.Path)
		if _, err := os.Stat(cachePath); err == nil {
			// File does exist. Serve from cache
			log.Info().Str("component", "handler").Str("source", "CACHE").Msgf("%s %s", r.Method, r.URL.Path)

			http.ServeFile(w, r, cachePath)
			return
		}

		// File does not exist in cache. Queue the download & serve from upstream
		log.Info().Str("component", "handler").Str("source", "ORIGIN").Msgf("%s %s", r.Method, r.URL.Path)

		cache_worker.QueueFileForDownload(cachePath)
		p.ServeHTTP(w, r)
	}
}
