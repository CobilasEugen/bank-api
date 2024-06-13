package main

import (
	"bank-api/db"
	"bank-api/router"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newMockApp() router.App {
	app := router.App{}
	db, _ := db.NewMockDb()
	app.Db = &db

	userLimiter := router.NewLimiter(5, func(r *http.Request) string { return r.PathValue("userId") })
	ipLimiter := router.NewLimiter(166, func(r *http.Request) string {
		fullAddress := r.RemoteAddr
		lastIndex := strings.LastIndex(fullAddress, ":")
		return fullAddress[:lastIndex] // only look at IP, remove port
	})

	app.Limiters = map[string]*router.Limiter{
		"ip":   ipLimiter,
		"user": userLimiter,
	}

	return app
}

func testRequest(t *testing.T, rr *httptest.ResponseRecorder, expectedCode int, expectedBody string) {
	if rr.Code != expectedCode {
		t.Errorf("handler returned wrong status code: got %v want %v",
			rr.Code, http.StatusOK)
	}

	if body := strings.TrimSpace(rr.Body.String()); body != expectedBody {
		t.Errorf("handler returned unexpected body:\ngot : %s\nwant: %s\n",
			body, expectedBody)
	}
}

func TestIpRateLimiting(t *testing.T) {
	log.SetOutput(io.Discard)
	app := newMockApp()

	userHandler := http.HandlerFunc(app.GetUser())
	userReq, _ := http.NewRequest("GET", "/user/1", nil)
	userReq.SetPathValue("userId", "1")
	userReq.RemoteAddr = "127.0.0.1:8080"

	// make 5 requests with userId 1
	for range 5 {
		rr := httptest.NewRecorder()
		userHandler.ServeHTTP(rr, userReq)
		testRequest(t, rr, http.StatusOK, `{"id":1,"name":"Bob"}`)
	}

	// make 161 requests that do not use the userId, but are from the same ip
	createHandler := http.HandlerFunc(app.CreateUser())
	for i := range 161 {
		reader := strings.NewReader(`{"name": "Dan"}`)
		createReq, _ := http.NewRequest("POST", "/user/", reader)
		createReq.Header.Set("Content-Type", "application/json")
		createReq.RemoteAddr = "127.0.0.1:8080"

		rr := httptest.NewRecorder()
		createHandler.ServeHTTP(rr, createReq)
		log.Println(rr.Body.String())
		testRequest(t, rr, http.StatusOK, `{"id":`+fmt.Sprint(i+3)+`,"name":"Dan"}`)
	}

	// the 167th request fails
	rr := httptest.NewRecorder()
	userHandler.ServeHTTP(rr, userReq)
	testRequest(t, rr, http.StatusTooManyRequests, `Rate Limit Exceeded`)
}

func TestUserRateLimiting(t *testing.T) {
	log.SetOutput(io.Discard)
	app := newMockApp()

	userReq, _ := http.NewRequest("GET", "/user/2", nil)
	userReq.SetPathValue("userId", "2")
	userReq.RemoteAddr = "localhost:8080"
	accReq, _ := http.NewRequest("GET", "/account/2", nil)
	accReq.SetPathValue("userId", "2")
	accReq.RemoteAddr = "localhost:8080"

	// make 3 requests to one endpoint with userId 2
	userHandler := http.HandlerFunc(app.GetUser())
	for range 3 {
		rr := httptest.NewRecorder()
		userHandler.ServeHTTP(rr, userReq)
		testRequest(t, rr, http.StatusOK, `{"id":2,"name":"Charlie"}`)
	}
	// make 2 requests to another endpoint with the same userId 2
	accountHandler := http.HandlerFunc(app.GetAccounts())
	for range 2 {
		rr := httptest.NewRecorder()
		accountHandler.ServeHTTP(rr, accReq)
		testRequest(t, rr, http.StatusOK, `[{"id":2,"user_id":2,"balance":200},{"id":3,"user_id":2,"balance":300}]`)
	}

	// 6th request fails
	rr := httptest.NewRecorder()
	userHandler.ServeHTTP(rr, userReq)
	testRequest(t, rr, http.StatusTooManyRequests, `Rate Limit Exceeded`)

	// request with other userId goes through
	tranReq, _ := http.NewRequest("GET", "/transaction/in/1", nil)
	tranReq.SetPathValue("userId", "1")
	tranReq.RemoteAddr = "localhost:8080"

	tranHandler := http.HandlerFunc(app.GetInTransactions())
	rr = httptest.NewRecorder()
	tranHandler.ServeHTTP(rr, tranReq)
	testRequest(t, rr, http.StatusOK, `[{"id":0,"from_account_id":0,"to_account_id":1,"amount":600,"timestamp":"2030-10-07T12:44:22+05:30","succeeded":1},{"id":1,"from_account_id":0,"to_account_id":1,"amount":500,"timestamp":"2030-10-07T12:44:22+05:30","succeeded":0}]`)

	// if we wait a bit, we can once again make requests with the initial userId
	time.Sleep(time.Millisecond * 500)
	rr = httptest.NewRecorder()
	accountHandler.ServeHTTP(rr, accReq)
	testRequest(t, rr, http.StatusOK, `[{"id":2,"user_id":2,"balance":200},{"id":3,"user_id":2,"balance":300}]`)
}

func TestFailedTransactionsRateLimiting(t *testing.T) {
	log.SetOutput(io.Discard)
	app := newMockApp()

	createHandler := http.HandlerFunc(app.CreateTransaction())

	// two failed transactions are already present in the db
	// good transactions goes through
	reader := strings.NewReader(`{
		"from_account_id": 0,
		"to_account_id": 3,
		"amount": 100.0
	}`)
	createReq, _ := http.NewRequest("POST", "/transaction/", reader)
	createReq.Header.Set("Content-Type", "application/json")
	createReq.RemoteAddr = "127.0.0.1:8080"
	rr := httptest.NewRecorder()
	createHandler.ServeHTTP(rr, createReq)
	testRequest(t, rr, http.StatusOK, `{"id":4,"from_account_id":0,"to_account_id":3,"amount":100,"timestamp":"2030-10-07T12:44:22+05:30","succeeded":1}`)

	// bad transaction leads to third failure
	reader = strings.NewReader(`{
		"from_account_id": 0,
		"to_account_id": 3,
		"amount": 300.0
	}`)
	createReq, _ = http.NewRequest("POST", "/transaction/", reader)
	createReq.Header.Set("Content-Type", "application/json")
	createReq.RemoteAddr = "127.0.0.1:8080"
	rr = httptest.NewRecorder()
	createHandler.ServeHTTP(rr, createReq)

	// three bad transactions have been made in the past day, so we get rate limited
	reader = strings.NewReader(`{
		"from_account_id": 0,
		"to_account_id": 3,
		"amount": 100.0
	}`)
	createReq, _ = http.NewRequest("POST", "/transaction/", reader)
	createReq.Header.Set("Content-Type", "application/json")
	createReq.RemoteAddr = "127.0.0.1:8080"
	rr = httptest.NewRecorder()
	createHandler.ServeHTTP(rr, createReq)
	testRequest(t, rr, http.StatusTooManyRequests, `Rate Limit Exceeded`)
}
