package httpserver

import (
	"net/http"

	"github.com/OliveiraNt/maned-scout/internal/adapters/http/ui/templates/pages"
	"github.com/OliveiraNt/maned-scout/internal/application"
	"github.com/OliveiraNt/maned-scout/internal/utils"
	"github.com/go-chi/chi/v5"
)

func (s *Server) apiListConsumerGroup(w http.ResponseWriter, r *http.Request) {
	clusterName := chi.URLParam(r, "clusterName")

	service := application.NewConsumerGroupsService(s.clusterService)

	cgs, err := service.ListConsumerGroupsWithLagFromTopic(r.Context(), clusterName, "")

	if err != nil {
		utils.Logger.Error("api get consumer groups failed", "cluster", clusterName, "err", err)
		http.Error(w, err.Error(), mapErrorToHTTPStatus(err))
		return
	}

	if err := pages.ConsumerGroupsListFragment(clusterName, cgs, service).Render(r.Context(), w); err != nil {
		utils.Logger.Error("render consumer groups list view failed", "err", err)
		http.Error(w, "failed to render consumer groups list view", 500)
		return
	}
}
