package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/nxadm/tail"
)

const DIR_NAME = "logs"

func errOut(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprint(w, err.Error())
}

func ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Ack")
}

func put(w http.ResponseWriter, r *http.Request) {
	runId := r.PathValue("runId")
	path := filepath.Join(DIR_NAME, runId)

	log, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		errOut(w, err)
		return
	}

	_, err = io.Copy(log, r.Body)
	if err != nil {
		errOut(w, err)
		return
	}

	log.Close()

	fmt.Fprint(w, "Ok")
}

func del(w http.ResponseWriter, r *http.Request) {
	log := filepath.Join(DIR_NAME, r.PathValue("runId"))

	os.Remove(log)

	fmt.Fprint(w, "Ok")
}

func get(w http.ResponseWriter, r *http.Request) {
	runId := r.PathValue("runId")
	log := filepath.Join(DIR_NAME, runId)

	if _, err := os.Stat(log); errors.Is(err, os.ErrNotExist) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "Run not found: "+runId)
		return
	}

	t, err := tail.TailFile(log, tail.Config{Follow: true, ReOpen: true})
	if err != nil {
		errOut(w, err)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		errOut(w, errors.New("Cannot make a HTTP flusher"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Connection", "Keep-Alive")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	for line := range t.Lines {
		fmt.Fprintln(w, line.Text)
		flusher.Flush()
	}
}

func main() {
	port, exists := os.LookupEnv("PORT")
	if !exists {
		port = "8002"
	}
	mux := http.NewServeMux()
	path := "/bob_logs/runs/{runId}"

	mux.HandleFunc("GET /ping", ping)
	mux.HandleFunc("PUT "+path, put)
	mux.HandleFunc("DELETE "+path, del)
	mux.HandleFunc("GET "+path, get)

	os.Mkdir(DIR_NAME, os.ModePerm)

	http.ListenAndServe(":"+port, mux)
}
