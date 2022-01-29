package main

import (
	"encoding/json"
	sitemap "github.com/dinofizz/sitemapper/sitemapper/internal"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"time"
)

type API struct {
	router *mux.Router
	nats   *sitemap.NATS
	CassDB *sitemap.AstraDB
}

func (a *API) initRoutes() {
	a.router.HandleFunc("/live", a.health).Methods("GET")
	a.router.HandleFunc("/ready", a.health).Methods("GET")
	a.router.HandleFunc("/sitemap", a.createSitemap).Methods("POST")
	a.router.HandleFunc("/sitemap/{id}", a.getSitemapResults).Methods("GET")
}

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	if err := a.CassDB.HealthCheck(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error occurred")
	}
	w.WriteHeader(http.StatusNoContent)
}

type SitemapCreateRequest struct {
	URL      string
	MaxDepth int
}
type SitemapCreateResponse struct {
	SitemapCreateRequest
	SitemapID string
}

func (a *API) createSitemap(w http.ResponseWriter, r *http.Request) {
	var scr SitemapCreateRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&scr); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	sitemapID, err := uuid.NewUUID()
	if err != nil {
		log.Print(err)
		respondWithError(w, http.StatusInternalServerError, "Unable to create new UUID")
		return
	}

	err = a.nats.SendStartMessage(sitemapID, scr.URL, scr.MaxDepth)
	if err != nil {
		log.Print(err)
		respondWithError(w, http.StatusInternalServerError, "Unable to send start message")
		return
	}

	response := SitemapCreateResponse{SitemapID: sitemapID.String(), SitemapCreateRequest: scr}
	respondWithJSON(w, 200, response)
}

func (a *API) getSitemapResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sid := vars["id"]
	if sid == "" {
		respondWithError(w, http.StatusBadRequest, "No sitemap ID in path")
		return
	}

	sitemapID, err := uuid.Parse(sid)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Sitemap ID invalid")
		return
	}

	smDetails, err := a.CassDB.GetSitemapDetails(sitemapID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	results, err := a.CassDB.GetSitemapResults(sitemapID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := struct {
		Count     int
		SitemapID string
		MaxDepth  int
		URL       string
		Results   *[]sitemap.Result
	}{
		Count:     len(*results),
		SitemapID: smDetails.SitemapID,
		URL:       smDetails.URL,
		MaxDepth:  smDetails.MaxDepth,
		Results:   results,
	}

	response.SitemapID = smDetails.SitemapID

	respondWithJSON(w, http.StatusOK, response)

}

func main() {
	router := mux.NewRouter()
	nm := sitemap.NewNATSManager()
	app := &API{CassDB: sitemap.NewAstraDB(), router: router, nats: nm}
	app.initRoutes()

	address := os.Getenv("API_ADDRESS")
	log.Printf("Starting web server on %s\n", address)
	srv := &http.Server{
		Handler:      router,
		Addr:         address,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err := w.Write(response)
	if err != nil {
		log.Print(err)
	}
}
