package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
)

type APIServer struct {
	listenAddr string
	store      Storage
}

func NewAPIServer(listenAddr string, store Storage) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		store:      store,
	}
}

func (s *APIServer) Run() {
	router := mux.NewRouter()

	router.HandleFunc("/login", makeHTTPHandleFunc(s.handleLogin))
	router.HandleFunc("/account", makeHTTPHandleFunc(s.handleAccount))
	router.HandleFunc("/account/{id}", withJWTAuth(makeHTTPHandleFunc(s.handleGetAccountByID), s.store))
	router.HandleFunc("/transfer", withJWTAuth(makeHTTPHandleFunc(s.handleTransfer), s.store))

	log.Println("JSON API server running on port:", s.listenAddr)
	log.Fatal(http.ListenAndServe(s.listenAddr, router))
}

func (s *APIServer) handleLogin(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return newHTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
	defer r.Body.Close()

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}

	acc, err := s.store.GetAccountByNumber(int(req.Number))
	if err != nil {
		return err
	}

	if !acc.ValidPassword(req.Password) {
		return newHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	token, err := createJWT(acc)
	if err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, LoginResponse{
		Token:  token,
		Number: acc.Number,
	})
}

func (s *APIServer) handleAccount(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case http.MethodGet:
		return s.handleGetAccount(w, r)
	case http.MethodPost:
		return s.handleCreateAccount(w, r)
	default:
		return newHTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *APIServer) handleGetAccount(w http.ResponseWriter, _ *http.Request) error {
	accounts, err := s.store.GetAccounts()
	if err != nil {
		return err
	}

	views := make([]AccountView, 0, len(accounts))
	for _, acc := range accounts {
		views = append(views, acc.View())
	}

	return WriteJSON(w, http.StatusOK, views)
}

func (s *APIServer) handleGetAccountByID(w http.ResponseWriter, r *http.Request) error {
	id, err := getID(r)
	if err != nil {
		return err
	}

	switch r.Method {
	case http.MethodGet:
		account, err := s.store.GetAccountByID(id)
		if err != nil {
			return err
		}
		return WriteJSON(w, http.StatusOK, account.View())

	case http.MethodDelete:
		if err := s.store.DeleteAccount(id); err != nil {
			return err
		}
		return WriteJSON(w, http.StatusOK, map[string]int{"deleted": id})

	default:
		return newHTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *APIServer) handleCreateAccount(w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	var req CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}

	account, err := NewAccount(req.FirstName, req.LastName, req.Password)
	if err != nil {
		return err
	}

	if err := s.store.CreateAccount(account); err != nil {
		return err
	}

	return WriteJSON(w, http.StatusCreated, account.View())
}

func (s *APIServer) handleTransfer(w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}

	if req.Amount <= 0 {
		return newHTTPError(http.StatusBadRequest, "invalid transfer amount")
	}

	tokenString := extractToken(r)
	token, err := validateJWT(tokenString)
	if err != nil || !token.Valid {
		return newHTTPError(http.StatusUnauthorized, "invalid token")
	}

	claims := token.Claims.(jwt.MapClaims)
	fromNumber := int64(claims["accountNumber"].(float64))

	err = s.store.Transfer(fromNumber, int64(req.ToAccount), int64(req.Amount))
	if err != nil {
		if errors.Is(err, ErrInsufficientFunds) {
			return newHTTPError(http.StatusConflict, "insufficient funds")
		}
		return err
	}

	return WriteJSON(w, http.StatusOK, map[string]string{
		"status": "transfer successful",
	})
}

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func createJWT(account *Account) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", fmt.Errorf("JWT_SECRET not set")
	}

	claims := jwt.MapClaims{
		"accountNumber": account.Number,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func withJWTAuth(handlerFunc http.HandlerFunc, s Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := extractToken(r)
		if tokenString == "" {
			WriteJSON(w, http.StatusUnauthorized, ApiError{Error: "missing token"})
			return
		}

		token, err := validateJWT(tokenString)
		if err != nil || !token.Valid {
			WriteJSON(w, http.StatusUnauthorized, ApiError{Error: "invalid token"})
			return
		}

		handlerFunc(w, r)
	}
}

func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return r.Header.Get("x-jwt-token")
}

func validateJWT(tokenString string) (*jwt.Token, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET not set")
	}

	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
}

type apiFunc func(http.ResponseWriter, *http.Request) error

type ApiError struct {
	Error string `json:"error"`
}

func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			if httpErr, ok := err.(httpError); ok {
				WriteJSON(w, httpErr.code, ApiError{Error: httpErr.msg})
				return
			}
			WriteJSON(w, http.StatusBadRequest, ApiError{Error: err.Error()})
		}
	}
}

type httpError struct {
	code int
	msg  string
}

func (e httpError) Error() string {
	return e.msg
}

func newHTTPError(code int, msg string) error {
	return httpError{code: code, msg: msg}
}

func getID(r *http.Request) (int, error) {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, fmt.Errorf("invalid id given %s", idStr)
	}
	return id, nil
}
