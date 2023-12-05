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

func getCurrentRank(ranks []int) int {
	return ranks[len(ranks)-1]
}

var io_mutex sync.RWMutex

type RuntimeError struct {
	Code    int
	Message string
	Log     string
}

const DefaultRank int = 400

func catch(w http.ResponseWriter) {

	if r := recover(); r != nil {

		w.Header().Set("Content-Type", "application/json")

		error := struct {
			What string `json:"error"`
		}{}

		if rt, ok := r.(RuntimeError); ok {
			w.WriteHeader(rt.Code)
			error.What = rt.Message
			log.Fatalln(rt.Log)
		} else {
			error.What = fmt.Sprintf("%v", r)
			w.WriteHeader(http.StatusBadRequest)
			log.Fatalln(error.What)
		}

		json.NewEncoder(w).Encode(error)
	}
}

func readKeyFile(path string) UserRanks {
	file, err := os.Open(path)
	if err != nil {
		panic(RuntimeError{
			Code:    http.StatusNotFound,
			Message: fmt.Sprintf(`key is invalid: "%s"`, path),
			Log:     fmt.Sprintf(`[ERROR] readKeyFile::Open "%s" failed: %v`, path, err),
		})
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		panic(RuntimeError{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf(`key is corrupted: "%s"`, path),
			Log:     fmt.Sprintf(`[ERROR] readKeyFile::ReadAll "%s" failed: %v`, path, err),
		})
	}

	if 0 == len(data) {
		return make(UserRanks)
	}

	var users map[string][]int
	err = json.Unmarshal(data, &users)
	if err != nil {
		panic(RuntimeError{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf(`key is corrupted: "%s"`, path),
			Log:     fmt.Sprintf(`[ERROR] readKeyFile::Unmarshal "%s" failed: %v`, path, err),
		})
	}

	return users
}

func writeKeyFile(path string, users UserRanks) {
	file, err := os.Create(path)
	if err != nil {
		panic(RuntimeError{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf(`key is corrupted: "%s"`, path),
			Log:     fmt.Sprintf(`[ERROR] writeKeyFile::Create "%s" failed: %v`, path, err),
		})
	}
	defer file.Close()

	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		panic(RuntimeError{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf(`key is corrupted: "%s"`, path),
			Log:     fmt.Sprintf(`[ERROR] writeKeyFile::MarshalIndent "%s" failed: %v`, path, err),
		})
	}

	_, err = file.Write(data)
	if err != nil {
		panic(RuntimeError{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf(`key is corrupted: "%s"`, path),
			Log:     fmt.Sprintf(`[ERROR] writeKeyFile::Write "%s" failed: %v`, path, err),
		})
	}
}

func postPlayer(w http.ResponseWriter, r *http.Request) {
	io_mutex.Lock()
	defer func() {
		io_mutex.Unlock()
	}()

	defer catch(w)

	vars := mux.Vars(r)
	path := vars["key"]
	name := vars["name"]

	key_file := readKeyFile(path)
	if _, ok := key_file[name]; !ok {
		key_file[name] = []int{DefaultRank}
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

	defer catch(w)

	vars := mux.Vars(r)
	path := vars["key"]
	name := vars["name"]

	key_file := readKeyFile(path)
	if ranks, ok := key_file[name]; !ok {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		data := struct {
			User string `json:"user"`
			Rank int    `json:"rank"`
		}{
			User: name,
			Rank: getCurrentRank(ranks),
		}

		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			panic(RuntimeError{
				Code:    http.StatusInternalServerError,
				Message: `json encoding failed`,
				Log:     fmt.Sprintf(`[ERROR] getPlayerRank::Marshal failed: %v`, err),
			})
		}
	}
}

func getPlayerHistory(w http.ResponseWriter, r *http.Request) {
	io_mutex.RLock()
	defer func() {
		io_mutex.RUnlock()
	}()

	defer catch(w)

	vars := mux.Vars(r)
	path := vars["key"]
	name := vars["name"]

	key_file := readKeyFile(path)
	if ranks, ok := key_file[name]; !ok {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		data := struct {
			User  string `json:"user"`
			Ranks []int  `json:"ranks"`
		}{
			User:  name,
			Ranks: ranks,
		}

		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			panic(RuntimeError{
				Code:    http.StatusInternalServerError,
				Message: `json encoding failed`,
				Log:     fmt.Sprintf(`[ERROR] getPlayerHistory::Marshal failed: %v`, err),
			})
		}
	}
}

func deletePlayer(w http.ResponseWriter, r *http.Request) {
	io_mutex.Lock()
	defer func() {
		io_mutex.Unlock()
	}()

	defer catch(w)

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

	defer catch(w)

	vars := mux.Vars(r)
	path := vars["key"]

	key_file := readKeyFile(path)

	elo_sys := elo.NewElo()
	score := 0.5
	wname := ""
	lname := ""

	if r.URL.Query().Has("winner") {
		if !r.URL.Query().Has("loser") {
			panic("missing argument: loser=<name>.")
		}

		wname = r.URL.Query().Get("winner")
		lname = r.URL.Query().Get("loser")
		score = 1.0

	} else if r.URL.Query().Has("draw") {
		players := strings.Split(r.URL.Query().Get("draw"), ",")
		if 2 != len(players) {
			panic("draw requires two names: draw=<name>,<name>.")
		}

		wname = players[0]
		lname = players[1]
		score = 0.5
	} else {
		panic("provide either: ?draw=<name>,<name> or ?winner=<name>&loser=<name>.")
	}

	wranks, ok := key_file[wname]
	if !ok {
		panic(RuntimeError{Message: fmt.Sprintf("unknown user: %s", wname), Code: http.StatusNotFound, Log: fmt.Sprintf(`[ERROR] postGame::wname user "%s" not found.`, wname)})
	}

	lranks, ok := key_file[lname]
	if !ok {
		panic(RuntimeError{Message: fmt.Sprintf("unknown user: %s", lname), Code: http.StatusNotFound, Log: fmt.Sprintf(`[ERROR] postGame::wname user "%s" not found.`, lname)})
	}

	wrank := getCurrentRank(wranks)
	lrank := getCurrentRank(lranks)

	woutcome, loutcome := elo_sys.Outcome(wrank, lrank, score)

	wranks = append(wranks, woutcome.Rating)
	lranks = append(lranks, loutcome.Rating)

	writeKeyFile(path, key_file)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	result := []struct {
		User  string `json:"user"`
		Rank  int    `json:"rank"`
		Delta int    `json:"delta"`
	}{
		{User: wname, Rank: woutcome.Rating, Delta: woutcome.Delta},
		{User: lname, Rank: loutcome.Rating, Delta: loutcome.Delta},
	}

	err := json.NewEncoder(w).Encode(result)
	if err != nil {
		panic(RuntimeError{
			Code:    http.StatusInternalServerError,
			Message: `json encoding failed`,
			Log:     fmt.Sprintf(`[ERROR] postGame::Marshal failed: %v`, err),
		})
	}
}

func getRanks(w http.ResponseWriter, r *http.Request) {
	io_mutex.RLock()
	defer func() {
		io_mutex.RUnlock()
	}()

	defer catch(w)

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
		data.Ranks = append(data.Ranks, Entry{User: key, Rank: getCurrentRank(ranks)})
	}

	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		panic(RuntimeError{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf(`key is corrupted: "%s"`, path),
			Log:     fmt.Sprintf(`[ERROR] getRanks::Marshal "%s" failed: %v`, path, err),
		})
	}
}

func getHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/{key}/player/{name}", postPlayer).Methods("POST")
	r.HandleFunc("/{key}/player/{name}/rank", getPlayerRank).Methods("GET")
	r.HandleFunc("/{key}/player/{name}/history", getPlayerHistory).Methods("GET")
	r.HandleFunc("/{key}/player/{name}", deletePlayer).Methods("DELETE")
	r.HandleFunc("/{key}/game", postGame).Methods("POST") // ?winner=<name>&loser=<name> or ?draw=<name>,<name>
	r.HandleFunc("/{key}/ranks", getRanks).Methods("GET")
	r.HandleFunc("/health", getHealth).Methods("GET")

	srv := &http.Server{
		Handler:      r,
		Addr:         "127.0.0.1:8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
