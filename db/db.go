package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type FailedTransactionsLimitError struct{}

func (err *FailedTransactionsLimitError) Error() string {
	return "Limit of failed transactions per day (3) has been reached"
}

type SQLiteDb struct {
	client *sql.DB
}

func NewSQLiteDb() (SQLiteDb, error) {
	db := SQLiteDb{}
	if err := db.InitDatabase(); err != nil {
		return db, err
	}

	return db, nil
}

func (sqlite *SQLiteDb) InitDatabase() error {
	if sqlite.client == nil {
		var err error
		sqlite.client, err = sql.Open("sqlite3", "./bank.db")
		if err != nil {
			return err
		}
		sqlite.createTables()
	}

	return nil
}

func (sqlite *SQLiteDb) createTables() {
	createUsersTable := `CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT
    );`

	createAccountsTable := `CREATE TABLE IF NOT EXISTS accounts (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER,
        balance REAL,
        FOREIGN KEY (user_id) REFERENCES users(id)
    );`

	createTransactionsTable := `CREATE TABLE IF NOT EXISTS transactions (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        from_account_id INTEGER,
        to_account_id INTEGER,
        amount REAL,
        succeeded INTEGER,
        timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (from_account_id) REFERENCES accounts(id)
        FOREIGN KEY (to_account_id) REFERENCES accounts(id)
    );`

	_, err := sqlite.client.Exec(createUsersTable)
	if err != nil {
		log.Fatal(err)
	}

	_, err = sqlite.client.Exec(createAccountsTable)
	if err != nil {
		log.Fatal(err)
	}

	_, err = sqlite.client.Exec(createTransactionsTable)
	if err != nil {
		log.Fatal(err)
	}
}

func (sqlite *SQLiteDb) CreateUser(userName string) (User, error) {
	sqlite.InitDatabase()
	var user User

	result, err := sqlite.client.Exec("INSERT INTO users (name) VALUES (?)", userName)
	if err != nil {
		return user, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return user, err
	}

	user.Name = userName
	user.ID = int(id)

	return user, nil
}

func (sqlite *SQLiteDb) CreateAccount(userId int, balance float64) (Account, error) {
	sqlite.InitDatabase()
	var account Account

	result, err := sqlite.client.Exec("INSERT INTO accounts (user_id, balance) VALUES (?, ?)", userId, balance)
	if err != nil {
		return account, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return account, err
	}

	account.ID = int(id)
	account.UserID = userId
	account.Balance = balance

	return account, nil
}

func (sqlite *SQLiteDb) CreateTransaction(fromAccountId int, toAccountId int, amount float64) (Transaction, error) {
	sqlite.InitDatabase()
	var transaction Transaction

	user, err := sqlite.GetUserByAccountId(fromAccountId)
	if err != nil {
		return transaction, err
	}

	pastTransactions, err := sqlite.GetTransactions(fmt.Sprint(user.ID), false)
	if err != nil {
		return transaction, err
	}

	past24Hours := time.Now().Add(-24 * time.Hour)
	failedTransactionsCnt := 0
	for _, transaction := range pastTransactions {
		if transaction.Timestamp.After(past24Hours) && transaction.Succeeded == 0 {
			failedTransactionsCnt += 1
		}
		if failedTransactionsCnt >= 3 {
			return transaction, &FailedTransactionsLimitError{}
		}
	}

	tx, err := sqlite.client.Begin()
	if err != nil {
		return transaction, err
	}

	var fromBalance float64
	err = tx.QueryRow("SELECT balance FROM accounts WHERE id = ?", fromAccountId).Scan(&fromBalance)
	if err != nil {
		tx.Rollback()
		return transaction, err
	}

	var toBalance float64
	err = tx.QueryRow("SELECT balance FROM accounts WHERE id = ?", toAccountId).Scan(&toBalance)
	if err != nil {
		tx.Rollback()
		return transaction, err
	}

	transactionSucceeded := 1
	if fromBalance < amount {
		transactionSucceeded = 0
	}
	newFromBalance := fromBalance - amount
	newToBalance := toBalance + amount

	if transactionSucceeded == 1 {
		_, err = tx.Exec("UPDATE accounts SET balance = ? WHERE id = ?", newFromBalance, fromAccountId)
		if err != nil {
			tx.Rollback()
			return transaction, err
		}
	}

	if transactionSucceeded == 1 {
		_, err = tx.Exec("UPDATE accounts SET balance = ? WHERE id = ?", newToBalance, toAccountId)
		if err != nil {
			tx.Rollback()
			return transaction, err
		}
	}

	result, err := tx.Exec("INSERT INTO transactions (from_account_id, to_account_id, amount, timestamp, succeeded) VALUES (?, ?, ?, ?, ?)", fromAccountId, toAccountId, amount, time.Now(), transactionSucceeded)
	if err != nil {
		tx.Rollback()
		return transaction, err
	}

	err = tx.Commit()
	if err != nil {
		return transaction, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return transaction, err
	}

	transaction.ID = int(id)
	transaction.FromAccountID = fromAccountId
	transaction.ToAccountID = toAccountId
	transaction.Amount = amount
	transaction.Timestamp = time.Now()
	transaction.Succeeded = transactionSucceeded

	return transaction, nil
}

func (sqlite *SQLiteDb) GetUser(userId string) (User, error) {
	sqlite.InitDatabase()
	var user User
	err := sqlite.client.QueryRow("SELECT id, name FROM users WHERE id = ?", userId).Scan(&user.ID, &user.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return user, nil
		} else {
			return user, err
		}
	}
	return user, nil
}

func (sqlite *SQLiteDb) GetUserByAccountId(accountId int) (User, error) {
	sqlite.InitDatabase()
	var user User

	query := `
    SELECT users.id, users.name
    FROM users
    INNER JOIN accounts ON users.id = accounts.user_id
    WHERE accounts.id = ?
    `

	err := sqlite.client.QueryRow(query, accountId).Scan(&user.ID, &user.Name)
	if err != nil {
		return user, err
	}

	return user, nil
}

func (sqlite *SQLiteDb) GetAccounts(userId string) ([]Account, error) {
	sqlite.InitDatabase()
	accounts := []Account{}

	rows, err := sqlite.client.Query("SELECT id, user_id, balance FROM accounts WHERE user_id = ?", userId)
	if err != nil {
		if err == sql.ErrNoRows {
			return accounts, nil
		} else {
			return nil, err
		}
	}
	defer rows.Close()

	for rows.Next() {
		var account Account
		if err := rows.Scan(&account.ID, &account.UserID, &account.Balance); err != nil {
			return accounts, err
		}

		accounts = append(accounts, account)
	}

	return accounts, nil
}

// incoming is true to get all transactions into the account
// incoming is false to get all transactions out of the account (outgoing transactions)
func (sqlite *SQLiteDb) GetTransactions(userId string, incoming bool) ([]Transaction, error) {
	sqlite.InitDatabase()
	transactions := []Transaction{}

	accounts, err := sqlite.GetAccounts(userId)
	if err != nil {
		return transactions, err
	}

	where_clause := "from_account_id"
	if incoming {
		where_clause = "to_account_id"
	}

	for _, account := range accounts {
		rows, err := sqlite.client.Query("SELECT id, from_account_id, to_account_id, amount, timestamp FROM transactions WHERE "+where_clause+" = ?", account.ID)
		if err != nil {
			if err == sql.ErrNoRows {
				return transactions, nil
			} else {
				return nil, err
			}
		}
		defer rows.Close()

		for rows.Next() {
			var transaction Transaction
			if err := rows.Scan(&transaction.ID, &transaction.FromAccountID, &transaction.ToAccountID, &transaction.Amount, &transaction.Timestamp); err != nil {
				return nil, err
			}

			transactions = append(transactions, transaction)
		}
	}

	return transactions, nil
}
