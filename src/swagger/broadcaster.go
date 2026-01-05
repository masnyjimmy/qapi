package swagger

import (
	"net/http"
	"slices"
	"sync"
)

type broadcaster struct {
	m       sync.Mutex
	clients []chan<- string
}

func NewBroadcaster() *broadcaster {
	return &broadcaster{
		clients: make([]chan<- string, 0),
	}
}

func (b *broadcaster) AddClient(ch chan<- string) {
	b.m.Lock()
	b.clients = append(b.clients, ch)
	b.m.Unlock()
}

func (b *broadcaster) RemoveClient(ch chan<- string) {
	b.m.Lock()
	defer b.m.Unlock()

	idx := slices.Index(b.clients, ch)

	if idx == -1 {
		panic("Unable to remove client channel, not found")
	}
	close(b.clients[idx])
	b.clients = slices.Delete(b.clients, idx, idx+1)
}

func (b *broadcaster) Broadcast(msg string) {
	b.m.Lock()
	for _, ch := range b.clients {
		select {
		case ch <- msg:
		default:
		}
	}
	b.m.Unlock()
}

func (b *broadcaster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

var _ http.Handler = (*broadcaster)(nil)
