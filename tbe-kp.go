package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
)

/*
Дробная часть в данной реализации не предусмотрена.
Нужно исползовать decimal (не понравились реализации которые нашел быстро)
Можно умножать на 100 при вводе, делить на 100 при выводе.
Математика float для денег не годится.
*/
type Account struct {
	ID      int `json:"id"`
	Limit   int `json:"limit"`
	Balance int `json:"balance"`
}
type TransferRequest struct {
	Sender    int `json:"sender"`
	Recipient int `json:"recipient"`
	Amount    int `json:"amount"`
}

func (transfer *TransferRequest) validate() url.Values {
	violations := url.Values{}

	if transfer.Amount <= 0 {
		violations.Add("amount", "Amount field is required and should be positive number")
	}
	if transfer.Sender <= 0 {
		violations.Add("sender", "Sender field is required and should be positive number")
	}
	if transfer.Recipient <= 0 {
		violations.Add("recipient", "Recipient field is required and should be positive number")
	}

	return violations
}

func getAccountById(id int, accounts []*Account) (error, *Account) {
	for _, account := range accounts {
		if account.ID == id {
			return nil, account
		}
	}

	return errors.New(fmt.Sprintf("Account with id \"%d\" not found", id)), &Account{}
}

// 0. Список аккаунтов
func ListAccounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(accounts); err != nil {
		log.Println(err)
	}
}

// 1. Создать аккаунт с определенным балансом (значение баланса должно приходить в запросе).
func CreateAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var account Account
	if err := json.NewDecoder(r.Body).Decode(&account); err != nil {
		SendBadRequest(w, r, []interface{}{err.Error()})
		return
	}

	account.ID = len(accounts) + 1
	mutex.Lock()
	accounts = append(accounts, &account)
	mutex.Unlock()
	if err := json.NewEncoder(w).Encode(account); err != nil {
		log.Println(err)
	}
}

// 2. Получить баланс аккаунта по его id.
func GetAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	params := mux.Vars(r)
	id, _ := strconv.Atoi(params["id"])

	errorNotFound, account := getAccountById(id, accounts)
	if errorNotFound == nil {
		if err := json.NewEncoder(w).Encode(account); err != nil {
			log.Println(err)
		}
		return
	}
	SendNotFound(w, r, errorNotFound.Error())
}

// 3. Перевести средства с одного аккаунта на другой.
func TransferAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var transferRequest TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&transferRequest); err != nil {
		SendBadRequest(w, r, []interface{}{err.Error()})
		return
	}

	if violations := transferRequest.validate(); len(violations) > 0 {
		SendBadRequest(w, r, []interface{}{violations})
		return
	}

	err, recipient := getAccountById(transferRequest.Recipient, accounts)
	if err != nil {
		SendBadRequest(w, r, []interface{}{err.Error()})
		return
	}
	err, sender := getAccountById(transferRequest.Sender, accounts)
	if err != nil {
		SendBadRequest(w, r, []interface{}{err.Error()})
		return
	}

	amount := transferRequest.Amount

	if sender.Limit > sender.Balance-amount {
		message := fmt.Sprintf("Not enough credits on account \"%d\"", sender.ID)
		SendBadRequest(w, r, []interface{}{message})
		return
	}

	mutex.Lock()
	sender.Balance -= amount
	recipient.Balance += amount
	mutex.Unlock()

	if err := json.NewEncoder(w).Encode(sender); err != nil {
		log.Println(err)
	}
}

func SendBadRequest(w http.ResponseWriter, r *http.Request, violations []interface{}) {
	w.WriteHeader(http.StatusBadRequest)
	response := map[string][]interface{}{"errors": violations}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Println(err)
	}
}

func SendNotFound(w http.ResponseWriter, r *http.Request, message string) {
	w.WriteHeader(http.StatusNotFound)
	errorResponse := map[string][]string{"errors": {0: message}}
	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		log.Println(err)
	}
}

var accounts []*Account
var mutex = &sync.Mutex{}

func main() {
	accounts = append(accounts, &Account{ID: 1, Limit: 0, Balance: 1000})
	accounts = append(accounts, &Account{ID: 2, Limit: 0, Balance: 3000})
	accounts = append(accounts, &Account{ID: 3, Limit: 0, Balance: 0})

	router := mux.NewRouter()
	router.HandleFunc("/accounts", ListAccounts).Methods("GET")

	/*
		{
			"balance": 1000, #опционально
			"limit": -1000 	 #опционально
		}
	*/
	router.HandleFunc("/accounts", CreateAccount).Methods("POST")
	router.HandleFunc("/accounts/{id:[0-9]+}", GetAccount).Methods("GET")

	/*
		{
			"sender": 1,    #обязательно
			"recipient": 2, #обязательно
			"amount": 100   #обязательно
		}
	*/
	router.HandleFunc("/accounts/transfer", TransferAccount).Methods("POST")
	fmt.Println("Listen on 127.0.0.1:3000")
	log.Fatal(http.ListenAndServe(":3000", router))
}
