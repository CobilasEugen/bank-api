package db

import (
	"time"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Account struct {
	ID      int     `json:"id"`
	UserID  int     `json:"user_id"`
	Balance float64 `json:"balance"`
}

type Transaction struct {
	ID            int       `json:"id"`
	FromAccountID int       `json:"from_account_id"`
	ToAccountID   int       `json:"to_account_id"`
	Amount        float64   `json:"amount"`
	Timestamp     time.Time `json:"timestamp"`
	Succeeded     int       `json:"succeeded"`
}
