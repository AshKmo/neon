package main

import (
	"fmt"
	"net/http"
	"log"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"errors"
	"math/rand"
	"github.com/joho/godotenv"
	"encoding/hex"
)

func generateToken() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	addr := os.Getenv("HOST_ADDRESS")

	const dbPath = "./database.db"

	// check if the database existed before we opened it
	_, err = os.Stat(dbPath)
	setupDatabase := errors.Is(err, os.ErrNotExist)

	// open the database
	fmt.Println("Loading database file...")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// if the database didn't pre-exist, initialise it
	if (setupDatabase) {
		fmt.Println("Initialising new database...")
		_, err = db.Exec(`
		create table users (userName text primary key, password text, userData text);
		create table invites (code text primary key);
		`)
		if err != nil {
			log.Fatal(err)
		}
	}

	mux := http.NewServeMux()
	httpServer := &http.Server{
		Addr: addr,
		Handler: mux,
	}

	mux.Handle("/", http.FileServer(http.Dir("./site")))

	mux.HandleFunc("/api/invite", func(w http.ResponseWriter, r *http.Request) {
	})

	fmt.Println("Server started at " + addr)

	log.Fatal(httpServer.ListenAndServe())
}
