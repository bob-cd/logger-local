/*
* Copyright 2025- Rahul De
*
* Use of this source code is governed by an MIT-style
* license that can be found in the LICENSE file or at
* https://opensource.org/licenses/MIT.
 */

package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

const DIR_NAME = "logs"

func errOut(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprint(w, err.Error())
}

func streamDiff(
	file *os.File,
	currentPos *int64,
	w http.ResponseWriter,
	flusher http.Flusher,
	ctx context.Context,
) {
	fileInfo, err := file.Stat()
	if err != nil {
		return
	}

	if fileInfo.Size() < *currentPos {
		// file truncated
		*currentPos = 0
		file.Seek(0, io.SeekStart)
	}

	if fileInfo.Size() > *currentPos {
		// file has new content
		file.Seek(*currentPos, io.SeekStart)
		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				fmt.Fprintln(w, scanner.Text()) // newline is needed to signal the end of chunk
				flusher.Flush()
			}
		}

		*currentPos = fileInfo.Size()
	}
}

func ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Ack")
}

func put(w http.ResponseWriter, r *http.Request) {
	if err := os.MkdirAll(DIR_NAME, os.ModePerm); err != nil {
		errOut(w, err)
		return
	}

	path := filepath.Join(DIR_NAME, r.PathValue("runId"))

	log, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		errOut(w, err)
		return
	}
	defer log.Close()

	data, err := io.ReadAll(r.Body)
	if err != nil {
		errOut(w, err)
		return
	}

	info, err := log.Stat()
	if err != nil {
		errOut(w, err)
		return
	}

	toWrite := strings.TrimSpace(string(data))
	if info.Size() > 0 {
		toWrite = "\n" + toWrite
	}

	_, err = log.WriteString(toWrite)
	if err != nil {
		errOut(w, err)
		return
	}

	fmt.Fprint(w, "Ok")
}

func del(w http.ResponseWriter, r *http.Request) {
	log := filepath.Join(DIR_NAME, r.PathValue("runId"))

	os.Remove(log)

	fmt.Fprint(w, "Ok")
}

func get(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("runId")
	log := filepath.Join(DIR_NAME, path)

	if _, err := os.Stat(log); errors.Is(err, os.ErrNotExist) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "Run not found")
		return
	}

	if r.URL.Query().Get("follow") != "true" {
		http.ServeFile(w, r, log)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	flusher, ok := w.(http.Flusher)
	if !ok {
		errOut(w, errors.New("Streaming unsupported"))
		return
	}

	var startPos int64 = 0
	file, err := os.Open(log)
	if err != nil {
		errOut(w, err)
		return
	}
	defer file.Close()

	_, err = file.Seek(startPos, io.SeekStart)
	if err != nil {
		errOut(w, err)
		return
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
			fmt.Fprintln(w, scanner.Text())
			flusher.Flush()
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		errOut(w, err)
		return
	}
	defer watcher.Close()

	if err = watcher.Add(log); err != nil {
		errOut(w, err)
		return
	}

	currentPos := startPos

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-watcher.Events:
			if event.Name == log {
				if event.Op&fsnotify.Remove == fsnotify.Remove {
					return
				}

				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					streamDiff(file, &currentPos, w, flusher, ctx)
				}
			}
		case err := <-watcher.Errors:
			slog.Error("Watcher error", "err", err)
		}
	}
}

func main() {
	port, exists := os.LookupEnv("PORT")
	if !exists {
		port = "8002"
	}
	mux := http.NewServeMux()
	path := "/bob_logs/{runId}"

	mux.HandleFunc("GET /ping", ping)
	mux.HandleFunc("PUT "+path, put)
	mux.HandleFunc("DELETE "+path, del)
	mux.HandleFunc("GET "+path, get)

	os.Mkdir(DIR_NAME, os.ModePerm)

	http.ListenAndServe(":"+port, mux)
}
