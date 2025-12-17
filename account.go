package main

import (
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var bcryptCost = bcrypt.DefaultCost

type LoginResponse struct {
	Number int64  `json:"number"`
	Token  string `json:"token"`
}

type LoginRequest struct {
	Number   int64  `json:"number"`
	Password string `json:"password"`
}

type TransferRequest struct {
	ToAccount int `json:"toAccount"`
	Amount    int `json:"amount"`
}

type CreateAccountRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Password  string `json:"password"`
}

type Account struct {
	ID                int       `json:"id"`
	FirstName         string    `json:"firstName"`
	LastName          string    `json:"lastName"`
	Number            int64     `json:"number"`
	EncryptedPassword string    `json:"-"`
	Balance           int64     `json:"balance"`
	CreatedAt         time.Time `json:"createdAt"`
}

type AccountView struct {
	ID        int       `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Number    int64     `json:"number"`
	Balance   int64     `json:"balance"`
	CreatedAt time.Time `json:"createdAt"`
}

func (a *Account) View() AccountView {
	return AccountView{
		ID:        a.ID,
		FirstName: a.FirstName,
		LastName:  a.LastName,
		Number:    a.Number,
		Balance:   a.Balance,
		CreatedAt: a.CreatedAt,
	}
}

func (a *Account) FullName() string {
	return a.FirstName + " " + a.LastName
}

func (a *Account) ValidPassword(pw string) bool {
	return bcrypt.CompareHashAndPassword(
		[]byte(a.EncryptedPassword),
		[]byte(pw),
	) == nil
}

func generateAccountNumber() int64 {
	return rand.Int63n(1_000_000_000_000)
}

func NewAccount(firstName, lastName, password string) (*Account, error) {
	encpw, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcryptCost,
	)
	if err != nil {
		return nil, err
	}

	return &Account{
		FirstName:         firstName,
		LastName:          lastName,
		EncryptedPassword: string(encpw),
		Number:            generateAccountNumber(),
		CreatedAt:         time.Now().UTC(),
	}, nil
}
