package db

import (
	"fmt"
	"time"
)

type MockDb struct {
	users        []User
	accounts     []Account
	transactions []Transaction
}

func NewMockDb() (MockDb, error) {
	db := MockDb{}
	if err := db.init(); err != nil {
		return db, err
	}

	return db, nil
}

func (mock *MockDb) init() error {
	mock.users = []User{
		{ID: 0, Name: "Alice"},
		{ID: 1, Name: "Bob"},
		{ID: 2, Name: "Charlie"},
	}

	mock.accounts = []Account{
		{ID: 0, UserID: 0, Balance: 400},
		{ID: 1, UserID: 1, Balance: 900},
		{ID: 2, UserID: 2, Balance: 200},
		{ID: 3, UserID: 2, Balance: 300},
	}

	// use future dates, as the backend will consider these to be transactions from today
	setTime, _ := time.Parse("2006-01-02 15:04:05 -0700", "2030-10-07 12:44:22 +0530")
	mock.transactions = []Transaction{
		{ID: 0, FromAccountID: 0, ToAccountID: 1, Amount: 600, Succeeded: 1, Timestamp: setTime},
		{ID: 1, FromAccountID: 0, ToAccountID: 1, Amount: 500, Succeeded: 0, Timestamp: setTime},
		{ID: 2, FromAccountID: 0, ToAccountID: 2, Amount: 500, Succeeded: 0, Timestamp: setTime},
		{ID: 3, FromAccountID: 2, ToAccountID: 3, Amount: 300, Succeeded: 1, Timestamp: setTime},
	}

	return nil
}

func (mock *MockDb) CreateUser(userName string) (User, error) {
	userId := len(mock.users)
	user := User{ID: userId, Name: userName}
	mock.users = append(mock.users, user)

	return user, nil
}

func (mock *MockDb) CreateAccount(userId int, balance float64) (Account, error) {
	accountId := len(mock.accounts)
	account := Account{ID: accountId, UserID: userId, Balance: balance}
	mock.accounts = append(mock.accounts, account)

	return account, nil
}

func (mock *MockDb) CreateTransaction(fromAccountId int, toAccountId int, amount float64) (Transaction, error) {
	var transaction Transaction

	user, err := mock.GetUserByAccountId(fromAccountId)
	if err != nil {
		return transaction, err
	}

	pastTransactions, err := mock.GetTransactions(fmt.Sprint(user.ID), false)
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

	var fromBalance float64
	for _, account := range mock.accounts {
		if account.ID == fromAccountId {
			fromBalance = account.Balance
		}
	}

	var toBalance float64
	for _, account := range mock.accounts {
		if account.ID == toAccountId {
			fromBalance = account.Balance
		}
	}

	transactionSucceeded := 1
	if fromBalance < amount {
		transactionSucceeded = 0
	}
	newFromBalance := fromBalance - amount
	newToBalance := toBalance + amount

	for i, account := range mock.accounts {
		if account.ID == fromAccountId {
			mock.accounts[i].Balance = newFromBalance
		}
	}

	for i, account := range mock.accounts {
		if account.ID == toAccountId {
			mock.accounts[i].Balance = newToBalance
		}
	}

	transaction.ID = len(mock.transactions)
	transaction.FromAccountID = fromAccountId
	transaction.ToAccountID = toAccountId
	transaction.Amount = amount
	setTime, _ := time.Parse("2006-01-02 15:04:05 -0700", "2030-10-07 12:44:22 +0530")
	transaction.Timestamp = setTime
	transaction.Succeeded = transactionSucceeded

	mock.transactions = append(mock.transactions, transaction)

	return transaction, nil
}

func (mock *MockDb) GetUser(userId string) (User, error) {
	for _, user := range mock.users {
		if fmt.Sprint(user.ID) == userId {
			return user, nil
		}
	}
	return User{}, fmt.Errorf("could not find user %s", userId)
}

func (mock *MockDb) GetUserByAccountId(accountId int) (User, error) {
	for _, account := range mock.accounts {
		if account.ID == accountId {
			return mock.GetUser(fmt.Sprint(account.UserID))
		}
	}
	return User{}, fmt.Errorf("could not find user with account %d", accountId)
}

func (mock *MockDb) GetAccounts(userId string) ([]Account, error) {
	accounts := []Account{}
	for _, account := range mock.accounts {
		if fmt.Sprintf("%d", account.UserID) == userId {
			accounts = append(accounts, account)
		}
	}
	return accounts, nil
}

// incoming is true to get all transactions into the account
// incoming is false to get all transactions out of the account (outgoing transactions)
func (mock *MockDb) GetTransactions(userId string, incoming bool) ([]Transaction, error) {
	transactions := []Transaction{}

	accounts, err := mock.GetAccounts(userId)
	if err != nil {
		return transactions, err
	}

	for _, account := range accounts {
		for _, transaction := range mock.transactions {

			if (incoming && transaction.ToAccountID == account.ID) || (!incoming && transaction.FromAccountID == account.ID) {
				transactions = append(transactions, transaction)
			}
		}
	}

	return transactions, nil
}
