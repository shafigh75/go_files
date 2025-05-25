package main

import (
	"fmt"
	"errors"
)

// BankAccount represents a bank account with a private balance field.
type BankAccount struct {
	balance float64 // unexported field, encapsulated within the package
}

// NewBankAccount is a constructor function that creates a new BankAccount.
func NewBankAccount(initialBalance float64) *BankAccount {
	return &BankAccount{balance: initialBalance}
}

// Deposit adds amount to the bank account and is a public method.
func (b *BankAccount) Deposit(amount float64) {
	if amount > 0 {
		b.balance += amount
	} else {
		fmt.Println("Deposit amount must be positive")
	}
}

// Withdraw subtracts amount from the bank account if funds are sufficient.
func (b *BankAccount) Withdraw(amount float64) error {
	if amount <= 0 {
		return errors.New("withdraw amount must be positive")
	}
	if amount > b.balance {
		return errors.New("insufficient funds")
	}
	b.balance -= amount
	return nil
}

// Balance returns the current balance using a public getter method.
func (b *BankAccount) Balance() float64 {
	return b.balance
}

func main() {
	account := NewBankAccount(1000)
	fmt.Println("Initial balance:", account.Balance())

	account.Deposit(500)
	fmt.Println("Balance after deposit:", account.Balance())

	if err := account.Withdraw(300); err != nil {
		fmt.Println("Withdraw error:", err)
	} else {
		fmt.Println("Balance after withdrawal:", account.Balance())
	}

	// Attempt to withdraw more than the balance
	if err := account.Withdraw(1500); err != nil {
		fmt.Println("Withdraw error:", err)
	}
}
