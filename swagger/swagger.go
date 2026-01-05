package swagger

import (
	_ "embed"
	"net/http"
	"path"
	"slices"
	"strings"
	"sync"
	"time"
)

//go:embed swagger.html
var swaggerUIBase string

func buildSwaggerUI(documentUrl, eventsUrl string) []byte {
	replacer := strings.NewReplacer(
		"%OPENAPI_DOCUMENT_URL%", documentUrl,
		"%EVENTS_URL%", eventsUrl,
	)

	return []byte(replacer.Replace(swaggerUIBase))
}

type Options struct {
	DebounceTime time.Duration
	BaseUrl      string
}

func DefaultOptions() Options {
	return Options{
		DebounceTime: DEFAULT_DEBOUNCE_TIME,
		BaseUrl:      "/",
	}
}

type urls struct {
	UI       string
	Document string
	Events   string
}

func makeUrls(base string) urls {
	return urls{
		UI:       path.Clean(base),
		Document: path.Join(base, "openapi.json"),
		Events:   path.Join(base, "events"),
	}
}

type Swagger struct {
	options Options

	broadcaster *broadcaster
	urls        urls
	mu          sync.RWMutex
	document    []byte
}

func New(document []byte, opt Options) (*Swagger, error) {

	broadcaster := NewBroadcaster()

	out := &Swagger{
		options:     opt,
		broadcaster: broadcaster,
		urls:        makeUrls(opt.BaseUrl),
		document:    document,
	}

	return out, nil
}

func (s *Swagger) Handler(h http.Handler) http.Handler {

	swaggerUI := buildSwaggerUI(s.urls.Document, s.urls.Events)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		switch path {
		case s.urls.UI:
			w.Header().Set("Content-Type", "text/html")
			w.Write(swaggerUI)
		case s.urls.Document:
			w.Header().Set("Content-Type", "application/json")
			s.mu.RLock()
			defer s.mu.RUnlock()
			w.Write(s.document)
		default:
			if path == s.urls.Events {
				s.broadcaster.ServeHTTP(w, r)
			} else {
				if h != nil {
					h.ServeHTTP(w, r)
				}
			}
		}
	})
}

func (s *Swagger) SetDocument(document []byte) {
	s.mu.Lock()
	s.document = slices.Clone(document)
	s.mu.Unlock()
	s.broadcaster.Broadcast("reload")
}
