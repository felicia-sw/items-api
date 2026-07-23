package main

import (

	"database/sql"

	"encoding/json"
	"fmt"
	"log"

	"net/http"
	"os"
	"strconv"

	"strings"

	_ "github.com/lib/pq"

)

type Item struct {

	ID          int    `json:"id"`

	Name        string `json:"name"`

	Description string `json:"description"`

}

var db *sql.DB

func main() {

	var err error

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getenv("DB_HOST", "localhost"),
		getenv("DB_PORT", "5432"),
		getenv("DB_USER", "feli"),
		os.Getenv("DB_PASSWORD"),
		getenv("DB_NAME", "itemsdb"),
	)

	db, err = sql.Open("postgres", connStr)

	if err != nil {

		log.Fatal("cannot open db: ", err)

	}

	if err = db.Ping(); err != nil {

		log.Fatal("cannot reach db: ", err)

	}

	log.Println("connected to database")

	http.HandleFunc("/items", itemsHandler)  // GET all, POST create

	http.HandleFunc("/items/", itemHandler)  // PUT update, DELETE one

	log.Println("listening on :8080 - auto-deployed v2")

	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}

func itemsHandler(w http.ResponseWriter, r *http.Request) {

	switch r.Method {

	case http.MethodGet:

		getItems(w, r)

	case http.MethodPost:

		createItem(w, r)

	default:

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

	}

}

func itemHandler(w http.ResponseWriter, r *http.Request) {

	idStr := strings.TrimPrefix(r.URL.Path, "/items/")

	id, err := strconv.Atoi(idStr)

	if err != nil {

		http.Error(w, "invalid id", http.StatusBadRequest)

		return

	}

	switch r.Method {

	case http.MethodPut:

		updateItem(w, r, id)

	case http.MethodDelete:

		deleteItem(w, id)

	default:

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

	}

}

func getItems(w http.ResponseWriter, r *http.Request) {

	rows, err := db.Query("SELECT id, name, COALESCE(description, '') FROM items ORDER BY id")

	if err != nil {

		http.Error(w, err.Error(), http.StatusInternalServerError)

		return

	}

	defer rows.Close()

	items := []Item{}

	for rows.Next() {

		var it Item

		if err := rows.Scan(&it.ID, &it.Name, &it.Description); err != nil {

			http.Error(w, err.Error(), http.StatusInternalServerError)

			return

		}

		items = append(items, it)

	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(items)

}

func createItem(w http.ResponseWriter, r *http.Request) {

	var it Item

	if err := json.NewDecoder(r.Body).Decode(&it); err != nil {

		http.Error(w, "invalid JSON", http.StatusBadRequest)

		return

	}

	err := db.QueryRow(

		"INSERT INTO items (name, description) VALUES ($1, $2) RETURNING id",

		it.Name, it.Description,

	).Scan(&it.ID)

	if err != nil {

		http.Error(w, err.Error(), http.StatusInternalServerError)

		return

	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(it)

}

func updateItem(w http.ResponseWriter, r *http.Request, id int) {

	var it Item

	if err := json.NewDecoder(r.Body).Decode(&it); err != nil {

		http.Error(w, "invalid JSON", http.StatusBadRequest)

		return

	}

	res, err := db.Exec(

		"UPDATE items SET name = $1, description = $2 WHERE id = $3",

		it.Name, it.Description, id,

	)

	if err != nil {

		http.Error(w, err.Error(), http.StatusInternalServerError)

		return

	}

	n, _ := res.RowsAffected()

	if n == 0 {

		http.Error(w, "item not found", http.StatusNotFound)

		return

	}

	it.ID = id

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(it)

}

func deleteItem(w http.ResponseWriter, id int) {

	res, err := db.Exec("DELETE FROM items WHERE id = $1", id)

	if err != nil {

		http.Error(w, err.Error(), http.StatusInternalServerError)

		return

	}

	n, _ := res.RowsAffected()

	if n == 0 {

		http.Error(w, "item not found", http.StatusNotFound)

		return

	}

	w.WriteHeader(http.StatusNoContent)

}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
