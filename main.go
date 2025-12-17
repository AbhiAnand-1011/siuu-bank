package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func seedAccount(store Storage, fname, lname, pw string) *Account {
	acc, err := NewAccount(fname, lname, pw)
	if err != nil {
		log.Fatal(err)
	}

	if err := store.CreateAccount(acc); err != nil {
		log.Fatal(err)
	}

	fmt.Println("seeded account =>", acc.Number)
	return acc
}

func seedAccounts(store Storage) {
	seedAccount(store, "abhi", "anand", "siuu")
}

func main() {
	seed := flag.Bool("seed", false, "seed the database with dummy data")
	flag.Parse()

	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET environment variable must be set")
	}

	store, err := NewPostgresStore()
	if err != nil {
		log.Fatal(err)
	}

	if err := store.Init(); err != nil {
		log.Fatal(err)
	}

	if *seed {
		log.Println("seeding database")
		seedAccounts(store)
	}

	addr := os.Getenv("SERVER_ADDR")
	if addr == "" {
		addr = ":3000"
	}

	server := NewAPIServer(addr, store)
	server.Run()
}
