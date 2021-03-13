package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"simplepaste/middleware"
	"simplepaste/util"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Path to the SQlite database
	dbPath := util.EnvDefault("SIMPLEPASTE_DB_PATH", "simplepaste.db")
	// Max paste size in bytes
	pasteMaxSize := util.EnvDefaultInt64("SIMPLEPASTE_MAX_PASTE_SIZE", 50*1024) // 50KB
	// The address at which to listen to
	listenAddress := util.EnvDefault("SIMPLEPASTE_LISTEN_ADDRESS", "127.0.0.1:8080")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Panicf("Failed to open SQLite database: %v", err)
	}

	if _, err := db.Exec(`

	create table if not exists pastes ( id      integer not null primary key autoincrement
	                                  , content text    not null
	                                  )

	`); err != nil {
		panic(err)
	}

	getPasteStmt, err := db.Prepare(`

	select content from pastes
	where id = ?
	limit 1

	`)
	if err != nil {
		panic(err)
	}

	insertPasteStmt, err := db.Prepare(`

	insert into pastes(content)
	values( ? )

	`)
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", middleware.LogHTTP(middleware.ServeFile("www/index.html")))
	http.HandleFunc("/paste/",
		middleware.LogHTTP(
			middleware.SwitchMethod(map[string]http.HandlerFunc{
				"GET": func(rw http.ResponseWriter, r *http.Request) {
					rw.Header().Set("Content-Type", "text/plain") // make sure that the client doesn't interpret the paste as html

					var id int64
					if _, err := fmt.Sscanf(r.URL.Path, "/paste/%d", &id); err != nil {
						rw.WriteHeader(http.StatusBadRequest)
						_, _ = fmt.Fprintf(rw, "Unable to parse %s because: %v", r.URL.Path, err)
						return
					}

					rows, err := getPasteStmt.Query(id)
					if err != nil {
						rw.WriteHeader(http.StatusInternalServerError)
						_, _ = io.WriteString(rw, "Internal Server Error")
						log.Printf("[ERROR] Error querying for paste id %d: %v", id, err)
						return
					}
					defer rows.Close()

					if !rows.Next() {
						// paste doesn't exist
						rw.WriteHeader(http.StatusNotFound)
						_, _ = fmt.Fprintf(rw, "Paste id %d not found", id)
						return
					}

					var content []byte
					if err := rows.Scan(&content); err != nil {
						rw.WriteHeader(http.StatusInternalServerError)
						_, _ = io.WriteString(rw, "Internal Server Error")
						log.Printf("[ERROR] Error scanning paste content for paste id %d: %v", id, err)
						return
					}

					_, _ = rw.Write(content)
				},
				"POST": func(rw http.ResponseWriter, r *http.Request) {
					if r.Body == nil {
						rw.WriteHeader(http.StatusBadRequest)
						_, _ = io.WriteString(rw, "Request Body is missing")
						return
					}
					defer r.Body.Close()

					content, err := io.ReadAll(io.LimitReader(r.Body, pasteMaxSize))
					if err != nil {
						rw.WriteHeader(http.StatusServiceUnavailable) // for the lack of a better status code
						_, _ = io.WriteString(rw, "Error reading request body")
						return
					}

					res, err := insertPasteStmt.Exec(content)
					if err != nil {
						rw.WriteHeader(http.StatusInternalServerError)
						_, _ = io.WriteString(rw, "Internal Server Error")
						log.Printf("[ERROR] Error inserting content: %v", err)
						return
					}

					lastInsertID, err := res.LastInsertId()
					if err != nil {
						rw.WriteHeader(http.StatusInternalServerError)
						_, _ = io.WriteString(rw, "Internal Server Error")
						log.Printf("[ERROR] Error fetching last insert ID: %v", err)
						return
					}

					_, _ = fmt.Fprintf(rw, "%d", lastInsertID)
				},
			})))

	log.Printf("[INFO] Listening on %s", listenAddress)
	panic(http.ListenAndServe(listenAddress, nil)) // http.ListenAndServe always returns an error
}
