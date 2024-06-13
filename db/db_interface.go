package db

type DbInterface interface {
	InitDatabase() error

	CreateUser(userName string) (User, error)
	CreateAccount(userId int, balance float64) (Account, error)
	CreateTransaction(fromAccountId int, toAccountId int, amount float64) (Transaction, error)

	GetUser(userId string) (User, error)
	GetUserByAccountId(accountID int) (User, error)
	GetAccounts(userId string) ([]Account, error)
	GetTransactions(userId string, incoming bool) ([]Transaction, error)
}
