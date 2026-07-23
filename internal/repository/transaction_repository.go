package repository

import (
	"flyup/internal/domain"

	"gorm.io/gorm"
)

type TransactionRepository interface {
	Create(txn *domain.Transaction) error
	FindByPaymentIntentID(intentID string) (*domain.Transaction, error)
	FindByInvestmentID(investmentID uint) (*domain.Transaction, error)
	UpdateStatus(id uint, status domain.TransactionStatus) error
	UpdateStripeFeesAndNet(id uint, stripeFee float64, stripeFeeVAT float64, netAmount float64) error
}

type transactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) TransactionRepository {
	return &transactionRepository{db}
}

// Create บันทึกรายการธุรกรรม (transaction) ใหม่ลงฐานข้อมูล
func (r *transactionRepository) Create(txn *domain.Transaction) error {
	return r.db.Create(txn).Error
}

// FindByPaymentIntentID ค้นหาธุรกรรมจาก Stripe payment intent ID พร้อมโหลดข้อมูลการลงทุน (Investment) มาด้วย
func (r *transactionRepository) FindByPaymentIntentID(intentID string) (*domain.Transaction, error) {
	txn := domain.Transaction{}

	err := r.db.Preload("Investment").Where("stripe_payment_intent_id = ?", intentID).First(&txn).Error

	if err != nil {
		return nil, err
	}

	return &txn, nil
}

// FindByInvestmentID ค้นหาธุรกรรมจาก investment ID
func (r *transactionRepository) FindByInvestmentID(investmentID uint) (*domain.Transaction, error) {
	txn := domain.Transaction{}

	err := r.db.Where("investment_id = ?", investmentID).First(&txn).Error

	if err != nil {
		return nil, err
	}

	return &txn, nil
}

// UpdateStatus อัปเดตสถานะ (status) ของธุรกรรมตาม ID
func (r *transactionRepository) UpdateStatus(id uint, status domain.TransactionStatus) error {
	return r.db.Model(&domain.Transaction{}).Where("id = ?", id).Update("status", status).Error
}

// UpdateStripeFeesAndNet อัปเดตค่าธรรมเนียม Stripe (fee, VAT ของ fee) และยอดเงินสุทธิ (net amount) ของธุรกรรม
func (r *transactionRepository) UpdateStripeFeesAndNet(id uint, stripeFee float64, stripeFeeVAT float64, netAmount float64) error {
	return r.db.Model(&domain.Transaction{}).Where("id = ?", id).Updates(map[string]interface{}{
		"stripe_fee":     stripeFee,
		"stripe_fee_vat": stripeFeeVAT,
		"net_amount":     netAmount,
	}).Error
}
