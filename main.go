package main

import (
	"fmt"
	"log"
	"net/http"

	"bank-api/router"
)

func main() {
	app := router.NewApp()

	http.HandleFunc("POST /user", app.CreateUser())
	http.HandleFunc("POST /account", app.CreateAccount())
	http.HandleFunc("POST /transaction", app.CreateTransaction())

	http.HandleFunc("GET /user/{userId}", app.GetUser())
	http.HandleFunc("GET /account/{userId}", app.GetAccounts())
	http.HandleFunc("GET /transaction/in/{userId}", app.GetInTransactions())
	http.HandleFunc("GET /transaction/out/{userId}", app.GetOutTransactions())

	servePort := 8080
	log.Printf("Server started at http://localhost:%d", servePort)

	err := http.ListenAndServe(fmt.Sprintf(":%d", servePort), nil)
	if err != nil {
		if err == http.ErrServerClosed {
			log.Println("Server closed")
		} else {
			log.Fatal(err)
		}
	}
}
