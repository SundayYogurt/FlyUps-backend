// Backfill fingerprints for ID card verifications created before card-number OCR was added.
// The default run only reports the number of rows. -apply makes billed iApp OCR calls.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"flyup/internal/domain"
	"flyup/internal/helper"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	apply := flag.Bool("apply", false, "read legacy card images with iApp OCR and persist fingerprints")
	limit := flag.Int("limit", 0, "maximum number of records to process (0 means all)")
	flag.Parse()
	if *limit < 0 {
		log.Fatal("limit must be nonnegative")
	}
	_ = godotenv.Load()
	dsn := os.Getenv("DSN")
	if dsn == "" {
		log.Fatal("DSN is required")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		log.Fatal(err)
	}
	if !*apply {
		var count int64
		query := db.Unscoped().Model(&domain.IdCardVerification{})
		if db.Migrator().HasColumn(&domain.IdCardVerification{}, "card_fingerprint") {
			query = query.Where("card_fingerprint IS NULL")
		}
		if err := query.Count(&count).Error; err != nil {
			log.Fatal(err)
		}
		fmt.Printf("legacy ID card records without a fingerprint: %d\n", count)
		fmt.Println("run with -apply to read the card images and fill these records")
		return
	}
	secret := os.Getenv("APP_SECRET")
	apiKey := os.Getenv("IAPP_API_KEY")
	if secret == "" || apiKey == "" {
		log.Fatal("APP_SECRET and IAPP_API_KEY are required with -apply")
	}
	if err := db.AutoMigrate(&domain.IdCardVerification{}); err != nil {
		log.Fatal(err)
	}
	var records []domain.IdCardVerification
	query := db.Unscoped().Where("card_fingerprint IS NULL").Order("id ASC")
	if *limit > 0 {
		query = query.Limit(*limit)
	}
	if err := query.Find(&records).Error; err != nil {
		log.Fatal(err)
	}
	ocr := helper.NewIAppService(apiKey)
	updated, sameUserDuplicates, unresolved := 0, 0, 0
	for _, record := range records {
		if record.Document == "" {
			log.Printf("record %d (user %d): missing card image", record.ID, record.UserID)
			unresolved++
			continue
		}
		payload, err := ocr.ReadIDCardFront(record.Document)
		if err != nil {
			log.Printf("record %d (user %d): OCR failed: %v", record.ID, record.UserID, err)
			unresolved++
			continue
		}
		var result struct {
			IDNumber string `json:"id_number"`
		}
		if err := json.Unmarshal([]byte(payload), &result); err != nil {
			log.Printf("record %d (user %d): invalid OCR response", record.ID, record.UserID)
			unresolved++
			continue
		}
		number, err := helper.NormalizeThaiIDNumber(result.IDNumber)
		if err != nil {
			log.Printf("record %d (user %d): unreadable card number", record.ID, record.UserID)
			unresolved++
			continue
		}
		fingerprint := helper.IDCardFingerprint(number, secret)
		var owner domain.IdCardVerification
		err = db.Unscoped().Where("card_fingerprint = ?", fingerprint).First(&owner).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("record %d (user %d): lookup failed: %v", record.ID, record.UserID, err)
			unresolved++
			continue
		}
		if err == nil {
			if owner.UserID == record.UserID {
				sameUserDuplicates++
				continue
			}
			log.Printf("record %d (user %d): card also belongs to user %d in record %d", record.ID, record.UserID, owner.UserID, owner.ID)
			unresolved++
			continue
		}
		write := db.Unscoped().Model(&domain.IdCardVerification{}).
			Where("id = ? AND card_fingerprint IS NULL", record.ID).
			Update("card_fingerprint", fingerprint)
		if write.Error != nil {
			log.Printf("record %d (user %d): duplicate card or database error: %v", record.ID, record.UserID, write.Error)
			unresolved++
			continue
		}
		updated += int(write.RowsAffected)
	}
	fmt.Printf("fingerprints added: %d; same-user duplicate records: %d; unresolved records: %d\n", updated, sameUserDuplicates, unresolved)
	if unresolved > 0 {
		os.Exit(1)
	}
}
