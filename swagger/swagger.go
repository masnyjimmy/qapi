package swagger

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/masnyjimmy/qapi/compilation"
	"github.com/masnyjimmy/qapi/docs"
	"github.com/masnyjimmy/qapi/validation"
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

func readAPI(filename string) ([]byte, error) {
	bytes, err := os.ReadFile(filename)

	if err != nil {
		return nil, fmt.Errorf("unable to read file %v: %w", filename, err)
	}

	if err := validation.Validate(bytes); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	var document docs.Document

	if err := yaml.Unmarshal(bytes, &document); err != nil {
		return nil, fmt.Errorf("unable to decode document: %w", err)
	}

	result, err := compilation.CompileToJSON(&document)

	if err != nil {
		return nil, fmt.Errorf("unable to compile document: %w", err)
	}

	return result, nil
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

func NewFromFile(filename string, opt Options) (*Swagger, error) {
	result, err := readAPI(filename)

	if err != nil {
		return nil, fmt.Errorf("unable to read api: %w", err)
	}

	return New(result, opt)
}

var ErrWatcher = errors.New("unable to create watcher")

func NewWithWatcher(filename string, ctx context.Context, opt Options) (*Swagger, error) {

	swagger, err := NewFromFile(filename, opt)

	if err != nil {
		return nil, err
	}

	watcher, err := WatchFile(filename, opt.DebounceTime)

	if err != nil {
		return swagger, fmt.Errorf("%w: %w", ErrWatcher, err)
	}

	go swagger.watchHandler(watcher, filename, ctx)

	return swagger, nil
}

func (s *Swagger) watchHandler(w *Watcher, filename string, ctx context.Context) {
	for {
		select {
		case err := <-w.Update:
			if err != nil {
				slog.Error("Swagger watcher fatal error", slog.String("details", err.Error()))
				w.Close()
				return
			}

			result, err := readAPI(filename)

			if err != nil {
				slog.Warn("Document update failed", slog.String("details", err.Error()))
			}

			s.SetDocument(result)
			slog.Info("Document updated")
		case <-ctx.Done():
			w.Close()
			return
		}
	}
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
