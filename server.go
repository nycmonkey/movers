package movers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type server struct {
	g      Getter
	router *mux.Router
}

func (s *server) routes() {
	s.router.HandleFunc("/gainers/{year:20[0-9]{2}}-{month:[01]?[0-9]}-{day:[0-3]?[0-9]}", s.handleGainers())
	s.router.HandleFunc("/losers/{year:20[0-9]{2}}-{month:[01]?[0-9]}-{day:[0-3]?[0-9]}", s.handleLosers())
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// NewHandler returns an http handler that responds to requests for top stock gainers and losers by date
func NewHandler(router *mux.Router) http.Handler {
	s := server{
		g:      NewGetter(),
		router: router,
	}
	s.routes()
	return &s
}

func (s *server) handleGainers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		year, err := strconv.Atoi(vars[`year`])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		month, err := strconv.Atoi(vars[`month`])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		day, err := strconv.Atoi(vars[`day`])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		var d Date
		d, err = NewDate(year, time.Month(month), day)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		stocks, err := s.g.Get(USCompositeGainers, d)
		if err != nil {
			http.Error(w, err.Error(), http.StatusFailedDependency)
		}
		w.Header().Set(`content-type`, `application/json`)
		json.NewEncoder(w).Encode(&stocks)
	}
}

func (s *server) handleLosers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		year, err := strconv.Atoi(vars[`year`])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		month, err := strconv.Atoi(vars[`month`])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		day, err := strconv.Atoi(vars[`day`])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		var d Date
		d, err = NewDate(year, time.Month(month), day)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		stocks, err := s.g.Get(USCompositeLosers, d)
		if err != nil {
			http.Error(w, err.Error(), http.StatusFailedDependency)
		}
		w.Header().Set(`content-type`, `application/json`)
		json.NewEncoder(w).Encode(&stocks)
	}
}
