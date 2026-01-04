/*
Copyright © 2026 NAME HERE
*/
package cmd

import (
	_ "embed"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/goccy/go-yaml"
	"github.com/masnyjimmy/qapi/src/compilation"
	"github.com/masnyjimmy/qapi/src/docs"
	"github.com/rs/cors"
	"github.com/spf13/cobra"
)

//go:embed swagger.html
var swaggerUI []byte

// ==================== Broadcaster ====================

type Broadcaster struct {
	m       sync.Mutex
	clients []chan<- string
}

func NewBroadcaster() Broadcaster {
	return Broadcaster{
		clients: make([]chan<- string, 0),
	}
}

func (b *Broadcaster) AddClient(ch chan<- string) {
	b.m.Lock()
	b.clients = append(b.clients, ch)
	b.m.Unlock()
}

func (b *Broadcaster) RemoveClient(ch chan<- string) {
	b.m.Lock()
	defer b.m.Unlock()

	idx := slices.Index(b.clients, ch)

	if idx == -1 {
		panic("Unable to remove client channel, not found")
	}
	close(b.clients[idx])
	b.clients = slices.Delete(b.clients, idx, idx+1)
}

func (b *Broadcaster) Broadcast(msg string) {
	b.m.Lock()
	for _, ch := range b.clients {
		select {
		case ch <- msg:
		default:
		}
	}
	b.m.Unlock()
}

func (b *Broadcaster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)

	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	msgCh := make(chan string, 1)
	b.AddClient(msgCh)
	defer b.RemoveClient(msgCh)

	notify := r.Context().Done()

	w.Write([]byte(":ok\n\n"))
	flusher.Flush()

	for {
		select {
		case <-notify:
			return
		case msg := <-msgCh:
			w.Write([]byte("event: update\n"))
			w.Write([]byte("data: " + msg + "\n\n"))
			flusher.Flush()
		}
	}
}

var _ http.Handler = (*Broadcaster)(nil)

// ==================== Cobra Command ====================

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve OpenAPI documentation with Redoc",
	RunE: func(cmd *cobra.Command, _ []string) error {
		input, err := cmd.Flags().GetString("input")
		if err != nil {
			return err
		}
		return Serve(input)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringP(
		"input",
		"i",
		"openapi.yaml",
		"OpenAPI 3.1 YAML file to watch",
	)
}

// ==================== Swagger Storage ====================

type Swagger struct {
	data  []byte
	mutex sync.RWMutex
}

func NewSwagger() Swagger {
	return Swagger{}
}

func (s *Swagger) UpdateData(data []byte) {
	s.mutex.Lock()
	s.data = append([]byte(nil), data...)
	s.mutex.Unlock()
}

func (s *Swagger) serverJSON() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		s.mutex.RLock()
		defer s.mutex.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		if len(s.data) == 0 {
			http.Error(w, "spec not ready", http.StatusServiceUnavailable)
			return
		}
		w.Write(s.data)
	}
}

func (s *Swagger) handleIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(swaggerUI))
	}
}

// ==================== API Watcher ====================

type SafeDocument struct {
	mu       sync.RWMutex
	document *compilation.Document
}

func NewSafeDocument() *SafeDocument {
	return &SafeDocument{
		document: compilation.NewDocument(),
	}
}
func (s *SafeDocument) lockWrite() {
	s.mu.Lock()
}
func (s *SafeDocument) unlockWrite() {
	s.mu.Unlock()
}

func (s *SafeDocument) Read(reader func(*compilation.Document)) {
	s.mu.RLock()
	reader(s.document)
	s.mu.RUnlock()
}

type APIWatcher struct {
	watcher    *fsnotify.Watcher
	filename   string
	document   docs.Document
	outDoc     *SafeDocument
	compiler   *compilation.Compiler
	timer      *time.Timer
	updateChan chan error
}

func NewAPIWatcher(filename string, outDoc *SafeDocument) (*APIWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if err := watcher.Add(filename); err != nil {
		return nil, err
	}

	api := &APIWatcher{
		watcher:    watcher,
		filename:   filename,
		document:   docs.Document{},
		outDoc:     outDoc,
		updateChan: make(chan error, 1),
	}

	api.compiler = compilation.New(&api.document, outDoc.document)

	// Initial load
	if err := api.reload(); err != nil {
		api.updateChan <- err
	} else {
		api.updateChan <- nil
	}

	go api.run()
	return api, nil
}

func (a *APIWatcher) run() {
	for {
		select {
		case err, ok := <-a.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)

		case ev, ok := <-a.watcher.Events:
			if !ok {
				return
			}
			if ev.Has(fsnotify.Write) {
				a.debounceReload()
			}
		}
	}
}

func (a *APIWatcher) debounceReload() {
	const debounce = 100 * time.Millisecond

	if a.timer != nil {
		a.timer.Stop()
	}

	a.timer = time.AfterFunc(debounce, func() {
		if err := a.reload(); err != nil {
			a.updateChan <- err
			return
		}
		a.updateChan <- nil
	})
}

func (a *APIWatcher) reload() error {
	bytes, err := os.ReadFile(a.filename)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(bytes, &a.document); err != nil {
		return err
	}

	a.outDoc.lockWrite()
	defer a.outDoc.unlockWrite()

	return a.compiler.Parse()
}

func (a *APIWatcher) OnUpdate() <-chan error {
	return a.updateChan
}

func (a *APIWatcher) Close() {
	a.watcher.Close()
	close(a.updateChan)
}

// ==================== Server ====================

func Serve(input string) error {
	output := NewSafeDocument()

	watcher, err := NewAPIWatcher(input, output)
	if err != nil {
		return err
	}
	defer watcher.Close()

	swagger := NewSwagger()
	broadcaster := NewBroadcaster()

	go func() {
		for err := range watcher.OnUpdate() {
			if err != nil {
				log.Printf("update error: %v", err)
				continue
			}
			var bytes []byte
			var err error

			output.Read(func(d *compilation.Document) {
				bytes, err = json.Marshal(*d)
			})

			if err != nil {
				log.Printf("marshal error: %v", err)
				continue
			}

			swagger.UpdateData(bytes)
			broadcaster.Broadcast("reload")
			log.Println("swagger updated")
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/", swagger.handleIndex())
	mux.Handle("/events", &broadcaster)
	mux.HandleFunc("/openapi.json", swagger.serverJSON())

	log.Println("Serving docs at http://localhost:8080")
	return http.ListenAndServe(":8080", cors.Default().Handler(mux))
}
