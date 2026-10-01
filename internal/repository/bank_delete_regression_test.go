package repository

import (
	"flyup/internal/domain"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"testing"
)

func TestDeleteBankAccountOwnershipAndDefault(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:bank-delete?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.BankAccount{}); err != nil {
		t.Fatal(err)
	}
	repo := NewUserRepository(db)
	first := domain.BankAccount{UserID: 7, AccountNumber: "12345678", IsDefault: true}
	second := domain.BankAccount{UserID: 7, AccountNumber: "87654321"}
	other := domain.BankAccount{UserID: 8, AccountNumber: "11111111", IsDefault: true}
	for _, bank := range []*domain.BankAccount{&first, &second, &other} {
		if err := repo.CreateBankAccount(bank); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.DeleteBankAccount(8, first.ID); err == nil {
		t.Fatal("deleted another user's bank")
	}
	if err := repo.DeleteBankAccount(7, first.ID); err == nil {
		t.Fatal("deleted default account")
	}
	if err := repo.DeleteBankAccount(7, second.ID); err != nil {
		t.Fatal(err)
	}
	banks, err := repo.FindBankByUserId(7)
	if err != nil || len(banks) != 1 || banks[0].ID != first.ID || !banks[0].IsDefault {
		t.Fatalf("default account changed: %+v, %v", banks, err)
	}
	if err := repo.DeleteBankAccount(7, first.ID); err == nil {
		t.Fatal("deleted last default account")
	}
	others, err := repo.FindBankByUserId(8)
	if err != nil || len(others) != 1 || !others[0].IsDefault {
		t.Fatalf("other user's account changed: %+v, %v", others, err)
	}
}
