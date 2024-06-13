package router

import (
	"bank-api/db"
	"log"
	"net/http"
	"strings"
)

type AppInterface interface {
	CreateUser() http.HandlerFunc
	CreateAccount() http.HandlerFunc
	CreateTransaction() http.HandlerFunc
	GetUser() http.HandlerFunc
	GetAccounts() http.HandlerFunc
	GetInTransactions() http.HandlerFunc
	GetOutTransactions() http.HandlerFunc
}

type App struct {
	Db         db.DbInterface
	AppHandler AppInterface
	Limiters   map[string]*Limiter
}

func NewApp() App {
	app := App{}
	db, err := db.NewSQLiteDb()
	if err != nil {
		log.Fatal("[ERROR] " + err.Error())
	}
	app.Db = &db

	userLimiter := NewLimiter(5, func(r *http.Request) string { return r.PathValue("userId") })
	ipLimiter := NewLimiter(166, func(r *http.Request) string { // 10.000 requests per minute = 166 requests per second
		fullAddress := r.RemoteAddr
		lastIndex := strings.LastIndex(fullAddress, ":")
		return fullAddress[:lastIndex] // only look at IP, remove port
	})

	app.Limiters = map[string]*Limiter{
		"ip":   ipLimiter,
		"user": userLimiter,
	}

	return app
}

func (app *App) RateLimit(handler http.HandlerFunc, name string) http.HandlerFunc {
	limiter, ok := app.Limiters[name]
	if !ok {
		log.Fatalf("limiter %s does not exist", name)
	}
	return RateLimit(handler, limiter)
}
