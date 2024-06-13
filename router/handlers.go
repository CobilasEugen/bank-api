package router

import (
	"bank-api/db"
	"encoding/json"
	"log"
	"net/http"
)

func (app *App) CreateUser() http.HandlerFunc {
	handler := func(w http.ResponseWriter, r *http.Request) {
		var user db.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, "Could not decode user data", http.StatusBadRequest)
			return
		}

		user, err := app.Db.CreateUser(user.Name)
		if err != nil {
			log.Println("[ERROR] " + err.Error())
			http.Error(w, "Could not create user", http.StatusInternalServerError)
			return
		}

		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, "Could not decode user data", http.StatusBadRequest)
			return
		}

		log.Printf("created new user: %d", user.ID)
	}

	return app.RateLimit(handler, "ip")
}

func (app *App) CreateAccount() http.HandlerFunc {
	handler := func(w http.ResponseWriter, r *http.Request) {
		var account db.Account
		if err := json.NewDecoder(r.Body).Decode(&account); err != nil {
			http.Error(w, "Could not decode account data", http.StatusBadRequest)
			return
		}

		account, err := app.Db.CreateAccount(account.UserID, account.Balance)
		if err != nil {
			log.Println("[ERROR] " + err.Error())
			http.Error(w, "Could not create account", http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(account); err != nil {
			http.Error(w, "Could not encode account data", http.StatusInternalServerError)
			return
		}

		log.Printf("created new account: %d", account.ID)
	}

	return app.RateLimit(handler, "ip")
}

func (app *App) CreateTransaction() http.HandlerFunc {
	handler := func(w http.ResponseWriter, r *http.Request) {
		var transaction db.Transaction
		if err := json.NewDecoder(r.Body).Decode(&transaction); err != nil {
			http.Error(w, "Could not decode transaction data", http.StatusBadRequest)
			return
		}

		transaction, err := app.Db.CreateTransaction(transaction.FromAccountID, transaction.ToAccountID, transaction.Amount)
		if err != nil {
			if _, ok := err.(*db.FailedTransactionsLimitError); ok {
				http.Error(w, "Rate Limit Exceeded", http.StatusTooManyRequests)
			} else {
				log.Println("[ERROR] " + err.Error())
				http.Error(w, "Could not execute transaction", http.StatusInternalServerError)
			}
			return
		}

		if err := json.NewEncoder(w).Encode(transaction); err != nil {
			http.Error(w, "Could not encode transaction data", http.StatusInternalServerError)
			return
		}

		log.Printf("created new transaction: %d", transaction.ID)
	}

	return app.RateLimit(handler, "ip")
}

func (app *App) GetUser() http.HandlerFunc {
	handler := func(w http.ResponseWriter, r *http.Request) {
		userId := r.PathValue("userId")

		user, err := app.Db.GetUser(userId)
		if err != nil {
			log.Println("[ERROR] " + err.Error())
			http.Error(w, "Could not read user data", http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(user); err != nil {
			http.Error(w, "Could not encode user data", http.StatusInternalServerError)
			return
		}

		log.Printf("read user %d", user.ID)
	}

	return app.RateLimit(app.RateLimit(handler, "ip"), "user")
}

func (app *App) GetAccounts() http.HandlerFunc {
	handler := func(w http.ResponseWriter, r *http.Request) {
		userId := r.PathValue("userId")

		accounts, err := app.Db.GetAccounts(userId)
		if err != nil {
			log.Println("[ERROR] " + err.Error())
			http.Error(w, "Could not read account data", http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(accounts); err != nil {
			http.Error(w, "Could not encode account data", http.StatusInternalServerError)
			return
		}

		log.Printf("read accounts of user %s", userId)
	}

	return app.RateLimit(app.RateLimit(handler, "ip"), "user")
}

func (app *App) GetInTransactions() http.HandlerFunc {
	handler := func(w http.ResponseWriter, r *http.Request) {
		userId := r.PathValue("userId")

		transactions, err := app.Db.GetTransactions(userId, true)
		if err != nil {
			log.Println("[ERROR] " + err.Error())
			http.Error(w, "Could not get transactions", http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(transactions); err != nil {
			http.Error(w, "Could not encode transaction data", http.StatusInternalServerError)
			return
		}

		log.Printf("read incoming transactions for user %s", userId)
	}

	return app.RateLimit(app.RateLimit(handler, "ip"), "user")
}

func (app *App) GetOutTransactions() http.HandlerFunc {
	handler := func(w http.ResponseWriter, r *http.Request) {
		userId := r.PathValue("userId")

		transactions, err := app.Db.GetTransactions(userId, false)
		if err != nil {
			log.Println("[ERROR] " + err.Error())
			http.Error(w, "Could not get transactions", http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(transactions); err != nil {
			http.Error(w, "Could not encode transaction data", http.StatusInternalServerError)
			return
		}

		log.Printf("read outgoing transactions for user %s", userId)
	}

	return app.RateLimit(app.RateLimit(handler, "ip"), "user")
}
