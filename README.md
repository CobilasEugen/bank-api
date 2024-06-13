# Bank API
Basic API modeling a transaction system, with rate limiting.
The backend is based on three tables: users, accounts (one user may have many accounts) and transactions (between accounts).

The API has the following endpoints:
 - `GET /user/{userId}` - returns user information 
 - `GET /account/{userId}` - returns accounts associated with `userId`
 - `GET /transaction/in/{userId}` - returns transactions into accounts associated with `userId`
 - `GET /transaction/out/{userId}` - returns transactions out of accounts associated with `userId`
 - `POST /user/` - create a user
 - `POST /account/` - create an account
 - `POST /transaction/` - create a transaction

A transaction will fail when the balance of the outgoing account is smaller than the transaction amount.

Rate limiting is implemented using a token bucket. The first time a user/IP address makes a request, a bucket with tokens is associated with it. When making another request, a token is removed from the bucket, and if the bucket is empty, the request is denied with a status code of 429 Too Many Requests. The tokens are replanished at a constant rate, based on the desired max requests per second value, until the bucket if filled.

Users are limited to 5 requests per second. IP addresses are limited to 166 requests per second (10.000 requests per minute). The backend also checks the transactions table when initiating a new transaction. If more than 3 transactions failed in the past day, the transaction is denied.

# Instructions
Run `go run .` to start the server on port 8080. Then make request to the previously mentioned endpoints.
See rate limiting in action by running `ab -n 20 "http://localhost:8080/user/1"`. 15 out of the 20 requests should fail (`ab` makes 20 requests very quickly, and it consumes the 5 tokens in under a second). To test all types of rate limting, run `go test ./tests/`.


# Examples
Run `go run .` to start the server.

 - create user
 ```bash
 curl -X POST http://localhost:8080/user \
            -H "Content-Type: application/json" \
            -d '{
              "name": "Alice"
            }'

 ```

 - create account
 ```bash
 curl -X POST http://localhost:8080/account \
            -H "Content-Type: application/json" \
            -d '{
              "user_id": 1,
              "balance": 1000.0
            }'
 ```

 - do a transaction
 ```bash
 curl -X POST http://localhost:8080/transaction \
            -H "Content-Type: application/json" \
            -d '{
              "from_account_id": 1,
              "to_account_id": 2,
              "amount": 200.0
            }'
 ```

 - check user
 ``` bash
 curl -X GET http://localhost:8080/user/1

 - check accounts
 ``` bash
 curl -X GET http://localhost:8080/account/1
 ```

 - check transactions
 ``` bash
 curl -X GET http://localhost:8080/transaction/in/1
 ```

 ``` bash
 curl -X GET http://localhost:8080/transaction/out/2
 ```
