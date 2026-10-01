package domain

import "time"

// Shared by incoming profit pools and outgoing investor payouts. No soft delete:
// a transfer remains consumed even if its business record is later archived.
type VerifiedSlip struct {
	ID               uint   `gorm:"primaryKey"`
	SenderBank       string `gorm:"size:10;uniqueIndex:idx_verified_bank_transfer"`
	TransRef         string `gorm:"size:255;uniqueIndex:idx_verified_bank_transfer"`
	ImageURL         string
	RecipientBank    string
	RecipientAccount string
	Amount           float64
	TransferredAt    time.Time
	VerifiedAt       time.Time
}
