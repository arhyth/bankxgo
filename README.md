# Bankxgo

A simple banking API code exercise in Go.

## REST API
The API supports the following endpoints for handling account operations:

### Create Account
Endpoint: `POST /accounts`  
Description: Creates a new account with a specified currency and email address.  
Request Body:  
```json
{
    "email": "arhyth@gmail.com",
    "currency": "USD"
}
```
Response:
`201` Created with the details of the newly created account.  
```json
{
    "acctId": "1833751339268609975"
}
```
`400` Bad Request if the currency is unsupported, the email is invalid, or an account with the same email already exists.  
`409` Conflict if an account with the same email already exists.  

### Withdraw Funds
Endpoint: `POST /accounts/{acctId}/withdraw`  
Description: Withdraws a specified amount from the user's account.  
Request Body:  
```json
{
    "amount": 100.0
}
```
Response:  
`200` OK on success.  
`400` Bad Request if the amount exceeds the available balance or other validation errors.  
`404` Not Found if the account is not found.

### Deposit Funds
Endpoint: `POST /accounts/{acctId}/deposit`  
Description: Deposits a specified amount into the user's account.  
Request Body:  
```json
{
    "amount": 200.0
}
```
Response:  
`200` OK on success.  
`400` Bad Request if the amount is invalid.  
`404` Not Found if the account is not found.  

### Generate Statement of Account (SOA)
Endpoint: `GET /accounts/{acctId}/statement`  
Description: Generates and returns a Statement of Account (SOA) for the specified account.  
Response:  
`200` OK with a PDF containing the transaction history.  
`404` Not Found if the account is not found.  

### View Balance
Endpoint: `GET /accounts/{acctId}/balance`  
Description: Retrieves the current balance of the user's account.  
Response:  
`200` OK with a JSON object containing the account balance.  
`404` Not Found if the account is not found.
