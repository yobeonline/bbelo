package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

type UserRanks map[string][]int

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
	w.WriteHeader(http.StatusNotImplemented)
}

func getRanks(w http.ResponseWriter, r *http.Request) {
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
