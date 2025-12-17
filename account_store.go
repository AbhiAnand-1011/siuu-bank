package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

var ErrInsufficientFunds = errors.New("insufficient funds")

type Storage interface {
	CreateAccount(*Account) error
	DeleteAccount(int) error
	UpdateAccount(*Account) error
	GetAccounts() ([]*Account, error)
	GetAccountByID(int) (*Account, error)
	GetAccountByNumber(int) (*Account, error)

	Transfer(fromNumber int64, toNumber int64, amount int64) error
}

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore() (*PostgresStore, error) {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = os.Getenv("PG_CONN")
	}
	if connStr == "" {
		connStr = "user=postgres dbname=postgres password=gobank sslmode=disable"
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresStore{
		db: db,
	}, nil
}

func (s *PostgresStore) Init() error {
	return s.createAccountTable()
}

func (s *PostgresStore) createAccountTable() error {
	query := `CREATE TABLE IF NOT EXISTS account (
		id SERIAL PRIMARY KEY,
		first_name VARCHAR(100),
		last_name VARCHAR(100),
		number BIGINT UNIQUE NOT NULL,
		encrypted_password TEXT,
		balance BIGINT DEFAULT 0,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`

	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStore) CreateAccount(acc *Account) error {
	query := `INSERT INTO account 
	(first_name, last_name, number, encrypted_password, balance, created_at)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING id`

	return s.db.QueryRow(
		query,
		acc.FirstName,
		acc.LastName,
		acc.Number,
		acc.EncryptedPassword,
		acc.Balance,
		acc.CreatedAt,
	).Scan(&acc.ID)
}

func (s *PostgresStore) UpdateAccount(acc *Account) error {
	query := `UPDATE account SET
		first_name = $1,
		last_name = $2,
		encrypted_password = $3,
		balance = $4
		WHERE id = $5`

	_, err := s.db.Exec(
		query,
		acc.FirstName,
		acc.LastName,
		acc.EncryptedPassword,
		acc.Balance,
		acc.ID,
	)

	return err
}

func (s *PostgresStore) DeleteAccount(id int) error {
	_, err := s.db.Exec("DELETE FROM account WHERE id = $1", id)
	return err
}

func (s *PostgresStore) GetAccountByNumber(number int) (*Account, error) {
	account := new(Account)
	query := `SELECT id, first_name, last_name, number, encrypted_password, balance, created_at
		FROM account
		WHERE number = $1`

	err := s.db.QueryRow(query, number).Scan(
		&account.ID,
		&account.FirstName,
		&account.LastName,
		&account.Number,
		&account.EncryptedPassword,
		&account.Balance,
		&account.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("account with number [%d] not found", number)
	}
	if err != nil {
		return nil, err
	}

	return account, nil
}

func (s *PostgresStore) GetAccountByID(id int) (*Account, error) {
	account := new(Account)
	query := `SELECT id, first_name, last_name, number, encrypted_password, balance, created_at
		FROM account
		WHERE id = $1`

	err := s.db.QueryRow(query, id).Scan(
		&account.ID,
		&account.FirstName,
		&account.LastName,
		&account.Number,
		&account.EncryptedPassword,
		&account.Balance,
		&account.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("account %d not found", id)
	}
	if err != nil {
		return nil, err
	}

	return account, nil
}

func (s *PostgresStore) GetAccounts() ([]*Account, error) {
	rows, err := s.db.Query(
		"SELECT id, first_name, last_name, number, encrypted_password, balance, created_at FROM account",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := []*Account{}
	for rows.Next() {
		account, err := scanIntoAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return accounts, nil
}

func (s *PostgresStore) Transfer(fromNumber int64, toNumber int64, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("invalid transfer amount")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	type accRow struct {
		id      int
		balance int64
	}

	getLocked := func(number int64) (*accRow, error) {
		row := tx.QueryRow(
			"SELECT id, balance FROM account WHERE number = $1 FOR UPDATE",
			number,
		)
		var a accRow
		if err := row.Scan(&a.id, &a.balance); err != nil {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("account not found")
			}
			return nil, err
		}
		return &a, nil
	}

	first := fromNumber
	second := toNumber
	if fromNumber > toNumber {
		first, second = toNumber, fromNumber
	}

	firstAcc, err := getLocked(first)
	if err != nil {
		return err
	}
	secondAcc, err := getLocked(second)
	if err != nil {
		return err
	}

	var fromAcc, toAcc *accRow
	if first == fromNumber {
		fromAcc = firstAcc
		toAcc = secondAcc
	} else {
		fromAcc = secondAcc
		toAcc = firstAcc
	}

	if fromAcc.balance < amount {
		return ErrInsufficientFunds
	}

	if _, err := tx.Exec(
		"UPDATE account SET balance = balance - $1 WHERE id = $2",
		amount,
		fromAcc.id,
	); err != nil {
		return err
	}

	if _, err := tx.Exec(
		"UPDATE account SET balance = balance + $1 WHERE id = $2",
		amount,
		toAcc.id,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func scanIntoAccount(rows *sql.Rows) (*Account, error) {
	account := new(Account)
	err := rows.Scan(
		&account.ID,
		&account.FirstName,
		&account.LastName,
		&account.Number,
		&account.EncryptedPassword,
		&account.Balance,
		&account.CreatedAt,
	)

	return account, err
}
