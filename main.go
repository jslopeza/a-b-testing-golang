package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	_ "github.com/lib/pq"
)

type variant struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description sql.NullString `json:"description"`
	Percent     int32          `json:"percent"`
}

type user struct {
	ID        string         `json:"id"`
	UserID    sql.NullString `json:"user_id"`
	VariantID string         `json:"variant_id"`
	Variant   variant
}

func runMigrations(db *sql.DB) {
	log.Println("Running migrations")
	_, err := db.Query(`CREATE EXTENSION IF NOT EXISTS pgcrypto;`)
	if err != nil {
		log.Fatal(err)
	}

	_, err2 := db.Query(`CREATE TABLE "variant" (
		id uuid NOT NULL DEFAULT gen_random_uuid(),
		name text NOT NULL,
		description text,
		percent int NOT NULL,
		PRIMARY KEY(id)
	);`)
	if err2 != nil {
		log.Fatal(err)
	}

	_, err3 := db.Query(`CREATE TABLE "user" (
		id uuid NOT NULL DEFAULT gen_random_uuid(),
		user_id text,
		variant_id uuid NOT NULL,
		PRIMARY KEY (id),
		FOREIGN KEY (variant_id) REFERENCES variant(id)
	);`)
	if err3 != nil {
		log.Fatal(err)
	}
}

func getVariant(db *sql.DB) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		var v variant
		err := db.QueryRow(`SELECT * FROM "variant" WHERE id = $1;`, vars["id"]).Scan(&v.ID, &v.Name, &v.Description, &v.Percent)
		if err != nil {
			log.Fatal(err)
			fmt.Fprintf(w, "Not found")
		}

		json.NewEncoder(w).Encode(v)
	}

	return http.HandlerFunc(fn)
}

func postVariant(db *sql.DB) http.HandlerFunc {
	var newVariant variant
	fn := func(w http.ResponseWriter, r *http.Request) {
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Fprintf(w, "JSON yo body")
		}

		json.Unmarshal(reqBody, &newVariant)

		var v variant
		err2 := db.QueryRow(`INSERT INTO "variant" (name, description, percent) VALUES ($1, $2, $3) RETURNING *`, newVariant.Name, newVariant.Description, newVariant.Percent).Scan(&v.ID, &v.Name, &v.Description, &v.Percent)
		if err2 != nil {
			log.Fatal(err2)
			fmt.Fprintf(w, "Error inserting into DB")
		}

		json.NewEncoder(w).Encode(v)

	}

	return http.HandlerFunc(fn)
}

func getUser(db *sql.DB) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		var u user
		err := db.QueryRow(`
		SELECT u.id          as id,
				u.user_id     as user_ud,
				u.variant_id  as variant_id,
				v.name        as variant_name,
				v.description as variant_description,
				v.percent as variant_percent
		FROM "user" as u
					LEFT JOIN "variant" as v on u.variant_id = v.id
		WHERE u.id = $1;
		`, vars["id"]).Scan(&u.ID, &u.UserID, &u.VariantID, &u.Variant.Name, &u.Variant.Description, &u.Variant.Percent)
		if err != nil {
			log.Fatal(err)
			fmt.Fprintf(w, "Not found")
		}

		json.NewEncoder(w).Encode(u)
	}

	return http.HandlerFunc(fn)
}

func postUser(db *sql.DB) http.HandlerFunc {
	var newUser user
	fn := func(w http.ResponseWriter, r *http.Request) {
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Fprintf(w, "JSON yo body")
		}

		json.Unmarshal(reqBody, &newUser)

		var u user
		err2 := db.QueryRow(`INSERT INTO "user" (user_id, variant_id) VALUES ($1, $2) RETURNING id, user_id, variant_id;`, newUser.UserID, newUser.VariantID).Scan(&u.ID, &u.UserID, &u.VariantID)
		if err2 != nil {
			log.Fatal(err2)
			fmt.Fprintf(w, "Error inserting into DB")
		}

		json.NewEncoder(w).Encode(u)

	}

	return http.HandlerFunc(fn)
}

func main() {
	connStr := "postgres://postgres@localhost:5432/a_b_tester?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	// Don't run this multiple times, TODO move to migrations file blah blah
	// runMigrations(db)

	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter().StrictSlash(true)
	// Variant
	r.HandleFunc("/api/variant/{id}", getVariant(db)).Methods("GET")
	r.HandleFunc("/api/variant", postVariant(db)).Methods("POST")

	// User
	r.HandleFunc("/api/user/{id}", getUser(db)).Methods("GET")
	r.HandleFunc("/api/user", postUser(db)).Methods("POST")

	log.Fatal(http.ListenAndServe(":8081", r))
}

// Client usage (maybe?)

// client.getVariant(id)
// new user visit, coin toss
// client.createUser(user)
// repeat user, check cookie
// client.getUser(user)
