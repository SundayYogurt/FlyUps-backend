package main

import (
	"errors"
	"flyup/internal/domain"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type domainSeed struct {
	Domain   string
	NameTH   string
	NameEN   string
	Province string
	IsActive bool
}

func main() {
	// best-effort load .env for local dev
	_ = godotenv.Load()

	dsn := strings.TrimSpace(os.Getenv("DSN"))
	if dsn == "" {
		log.Fatal("DSN env var is required")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("database connection error: %v", err)
	}

	// ensure tables exist (safe even if server already migrated)
	if err := db.AutoMigrate(&domain.University{}, &domain.UniversityDomain{}); err != nil {
		log.Fatalf("migration error: %v", err)
	}

	// if DB was restored/imported, sequences may be behind max(id)
	if err := syncPostgresSequences(db); err != nil {
		log.Fatalf("sequence sync error: %v", err)
	}

	seeds := []domainSeed{
		{
			Domain:   "webmail.npru.ac.th",
			NameTH:   "มหาวิทยาลัยราชภัฏนครปฐม",
			NameEN:   "Nakhon Pathom Rajabhat University",
			Province: "Nakhon Pathom",
			IsActive: true,
		},
		// เพิ่มโดเมนอื่นๆ ได้ที่นี่
	}

	start := time.Now()
	for _, s := range seeds {
		if err := upsertUniversityDomain(db, s); err != nil {
			log.Fatalf("seed failed for domain %q: %v", s.Domain, err)
		}
	}

	log.Printf("seed ok (%d domains) in %s", len(seeds), time.Since(start).String())
}

func syncPostgresSequences(db *gorm.DB) error {
	// fix only tables we seed here
	queries := []string{
		`SELECT setval(pg_get_serial_sequence('universities','id'), COALESCE((SELECT MAX(id) FROM universities), 1), true);`,
		`SELECT setval(pg_get_serial_sequence('university_domains','id'), COALESCE((SELECT MAX(id) FROM university_domains), 1), true);`,
	}
	for _, q := range queries {
		if err := db.Exec(q).Error; err != nil {
			return err
		}
	}
	return nil
}

func upsertUniversityDomain(db *gorm.DB, s domainSeed) error {
	d := strings.ToLower(strings.TrimSpace(s.Domain))
	if d == "" {
		return errors.New("domain is empty")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		uni, err := findOrCreateUniversity(tx, s)
		if err != nil {
			return err
		}

		var existing domain.UniversityDomain
		err = tx.Where("domain = ?", d).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			newRow := domain.UniversityDomain{
				UniversityID: uni.ID,
				Domain:       d,
				IsActive:     s.IsActive,
			}
			return tx.Create(&newRow).Error
		}

		existing.UniversityID = uni.ID
		existing.IsActive = s.IsActive
		return tx.Save(&existing).Error
	})
}

func findOrCreateUniversity(tx *gorm.DB, s domainSeed) (*domain.University, error) {
	nameTH := strings.TrimSpace(s.NameTH)
	nameEN := strings.TrimSpace(s.NameEN)
	province := strings.TrimSpace(s.Province)
	domainStr := strings.ToLower(strings.TrimSpace(s.Domain))

	var uni domain.University

	// try find by name_en first (more stable), fallback to name_th
	if nameEN != "" {
		if err := tx.Where("name_en = ?", nameEN).First(&uni).Error; err == nil {
			return &uni, nil
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	if nameTH != "" {
		if err := tx.Where("name_th = ?", nameTH).First(&uni).Error; err == nil {
			return &uni, nil
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	// create new university
	var nameTHPtr *string
	var nameENPtr *string
	var provincePtr *string
	var domainPtr *string

	if nameTH != "" {
		nameTHPtr = &nameTH
	}
	if nameEN != "" {
		nameENPtr = &nameEN
	}
	if province != "" {
		provincePtr = &province
	}
	if domainStr != "" {
		domainPtr = &domainStr
	}

	uni = domain.University{
		NameTH:   nameTHPtr,
		NameEN:   nameENPtr,
		Province: provincePtr,
		Domain:   domainPtr,
	}

	if err := tx.Create(&uni).Error; err != nil {
		return nil, err
	}
	return &uni, nil
}

