package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	elo "github.com/kortemy/elo-go"
)

type UserRanks map[string][]int

var io_mutex sync.RWMutex

func readKeyFile(path string) UserRanks {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}

	if 0 == len(data) {
		return make(UserRanks)
	}

	var users map[string][]int
	err = json.Unmarshal(data, &users)
	if err != nil {
		log.Fatal(err)
	}

	return users
}

func writeKeyFile(path string, users UserRanks) {
	file, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	_, err = file.Write(data)
	if err != nil {
		log.Fatal(err)
	}
}

func postPlayer(w http.ResponseWriter, r *http.Request) {
	io_mutex.Lock()
	defer func() {
		io_mutex.Unlock()
	}()

	vars := mux.Vars(r)
	path := vars["key"]
	name := vars["name"]

	key_file := readKeyFile(path)
	if _, ok := key_file[name]; !ok {
		key_file[name] = []int{400}
		writeKeyFile(path, key_file)
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func getPlayerRank(w http.ResponseWriter, r *http.Request) {
	io_mutex.RLock()
	defer func() {
		io_mutex.RUnlock()
	}()

	vars := mux.Vars(r)
	path := vars["key"]
	name := vars["name"]

	key_file := readKeyFile(path)
	if _, ok := key_file[name]; !ok {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		data := struct {
			User string `json:"user"`
			Rank int    `json:"rank"`
		}{
			User: name,
			Rank: key_file[name][len(key_file[name])-1],
		}

		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func getPlayerHistory(w http.ResponseWriter, r *http.Request) {
	io_mutex.RLock()
	defer func() {
		io_mutex.RUnlock()
	}()

	vars := mux.Vars(r)
	path := vars["key"]
	name := vars["name"]

	key_file := readKeyFile(path)
	if _, ok := key_file[name]; !ok {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		data := struct {
			User  string `json:"user"`
			Ranks []int  `json:"ranks"`
		}{
			User:  name,
			Ranks: key_file[name],
		}

		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func deletePlayer(w http.ResponseWriter, r *http.Request) {
	io_mutex.Lock()
	defer func() {
		io_mutex.Unlock()
	}()

	vars := mux.Vars(r)
	path := vars["key"]
	name := vars["name"]

	key_file := readKeyFile(path)
	if _, ok := key_file[name]; !ok {
		w.WriteHeader(http.StatusNotFound)
	} else {
		delete(key_file, name)
		writeKeyFile(path, key_file)
		w.WriteHeader(http.StatusOK)
	}
}

func postGame(w http.ResponseWriter, r *http.Request) {
	io_mutex.Lock()
	defer func() {
		io_mutex.Unlock()
	}()

	defer func() {
		if r := recover(); r != nil {
			w.WriteHeader(http.StatusBadRequest)
		}
	}()

	vars := mux.Vars(r)
	path := vars["key"]

	key_file := readKeyFile(path)

	elo_sys := elo.NewElo()
	score := 0.5
	wname := ""
	lname := ""

	if r.URL.Query().Has("winner") {
		if !r.URL.Query().Has("loser") {
			panic("No loser")
		}

		wname = r.URL.Query().Get("winner")
		lname = r.URL.Query().Get("loser")
		score = 1.0

	} else if r.URL.Query().Has("draw") {
		players := strings.Split(r.URL.Query().Get("draw"), ",")
		if 2 != len(players) {
			panic("there must be two players")
		}

		wname = players[0]
		lname = players[1]
		score = 0.5
	} else {
		panic("missing arguments")
	}

	if _, ok := key_file[wname]; !ok {
		panic("unknown user")
	}
	if _, ok := key_file[lname]; !ok {
		panic("unknown user")
	}

	wrank := key_file[wname][len(key_file[wname])-1]
	lrank := key_file[lname][len(key_file[lname])-1]

	woutcome, loutcome := elo_sys.Outcome(wrank, lrank, score)

	key_file[wname] = append(key_file[wname], woutcome.Rating)
	key_file[lname] = append(key_file[lname], loutcome.Rating)

	writeKeyFile(path, key_file)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s: %d\n%s: %d\n", wname, woutcome.Delta, lname, loutcome.Delta)
}

func getRanks(w http.ResponseWriter, r *http.Request) {
	io_mutex.RLock()
	defer func() {
		io_mutex.RUnlock()
	}()

	vars := mux.Vars(r)
	path := vars["key"]

	key_file := readKeyFile(path)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	type Entry struct {
		User string `json:"user"`
		Rank int    `json:"rank"`
	}

	data := struct {
		Ranks []Entry `json:"ranks"`
	}{}

	for key, ranks := range key_file {
		data.Ranks = append(data.Ranks, Entry{User: key, Rank: ranks[len(ranks)-1]})
	}

	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/{key}/player/{name}", postPlayer).Methods("POST")
	r.HandleFunc("/{key}/player/{name}/rank", getPlayerRank).Methods("GET")
	r.HandleFunc("/{key}/player/{name}/history", getPlayerHistory).Methods("GET")
	r.HandleFunc("/{key}/player/{name}", deletePlayer).Methods("DELETE")
	r.HandleFunc("/{key}/game", postGame).Methods("POST")
	r.HandleFunc("/{key}/ranks", getRanks).Methods("GET")

	srv := &http.Server{
		Handler:      r,
		Addr:         "127.0.0.1:8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
