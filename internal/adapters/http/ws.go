package httpserver

import (
	"bytes"
	"context"
	"net/http"

	"github.com/OliveiraNt/kdash/internal/adapters/http/ui/templates/pages"
	"github.com/OliveiraNt/kdash/internal/domain"
	"github.com/OliveiraNt/kdash/internal/registry"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	// TODO: tighten CORS/origin as needed. For now allow all to simplify local usage.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// wsStreamTopic upgrades to WebSocket and streams Kafka messages from the given topic to the client.
// On client disconnect, the Kafka consumption is canceled via context.
func (s *Server) wsStreamTopic(w http.ResponseWriter, r *http.Request) {
	clusterName := chi.URLParam(r, "clusterName")
	topicName := chi.URLParam(r, "topicName")

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		registry.Logger.Error("websocket upgrade failed", "cluster", clusterName, "topic", topicName, "err", err)
		http.Error(w, "websocket upgrade failed", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	conn.SetCloseHandler(func(code int, text string) error {
		registry.Logger.Info("websocket close handler triggered", "cluster", clusterName, "topic", topicName, "code", code)
		cancel()
		return nil
	})

	msgs := make(chan domain.Message, 256)

	go func() {
		defer func() {
			registry.Logger.Info("consumer goroutine stopping", "cluster", clusterName, "topic", topicName)
			cancel()
		}()
		if err := s.topicService.StreamMessages(ctx, clusterName, topicName, msgs); err != nil {
			registry.Logger.Error("stream messages failed", "cluster", clusterName, "topic", topicName, "err", err)
		}
		registry.Logger.Info("stream stopped", "cluster", clusterName, "topic", topicName)
		close(msgs)
	}()

	go func() {
		defer cancel()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				registry.Logger.Info("websocket client disconnected",
					"cluster", clusterName,
					"topic", topicName,
					"err", err,
				)
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case m, ok := <-msgs:
			if !ok {
				registry.Logger.Info("message channel closed", "cluster", clusterName, "topic", topicName)
				return
			}

			var buf bytes.Buffer
			err := pages.Message(m).Render(r.Context(), &buf)
			if err != nil {
				registry.Logger.Error("failed to render message", "err", err)
				continue
			}
			if err := conn.WriteMessage(websocket.TextMessage, buf.Bytes()); err != nil {
				registry.Logger.Info("websocket write failed, stopping stream", "cluster", clusterName, "topic", topicName, "err", err)
				return
			}
		}
	}
}
