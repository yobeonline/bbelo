package bbelo

import (
	"net/http"

	"github.com/gorilla/mux"
)

func postPlayer(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func getPlayerRank(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func getPlayerHistory(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func deletePlayer(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func postGame(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func getRanks(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/{key}/player/{name}", postPlayer).Methods("POST")
	r.HandleFunc("/{key}/player/{name}/rank", getPlayerRank).Methods("GET")
	r.HandleFunc("/{key}/player/{name}/history", getPlayerHistory).Methods("GET")
	r.HandleFunc("/{key}/player/{name}", deletePlayer).Methods("DELETE")
	r.HandleFunc("/{key}/game", postGame).Methods("POST")
	r.HandleFunc("/{key}/ranks", getRanks).Methods("GET")

	http.Handle("/", r)
}
