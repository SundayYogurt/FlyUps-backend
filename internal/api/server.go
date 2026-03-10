package api

import (
	"flyup/config"
	"flyup/internal/api/rest"
	"flyup/internal/api/rest/handlers"
	"flyup/internal/database/seeder"
	"flyup/internal/domain"
	"flyup/internal/helper"
	"flyup/pkg/notification"
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func StartServer(cfg config.AppConfig) {
	app := fiber.New()

	log.Println("DSN =", cfg.Dsn)
	db, err := gorm.Open(postgres.Open(cfg.Dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("database connection error %v\n", err)
	}
	log.Println("database connected")

	// run migration
	err = db.AutoMigrate(
		&domain.User{},
		&domain.University{},
		&domain.UniversityDomain{},
		&domain.UserConsent{},
		&domain.StudentProfile{},

		// project
		&domain.Project{},
		&domain.ProjectCategory{},
		&domain.ProjectMedia{},
		&domain.ProjectStorySection{},
		&domain.ProjectRisk{},
		&domain.ProjectFAQ{},
		&domain.ProjectFundingPolicy{},
		&domain.ProjectProfitPolicy{},

		// milestone
		&domain.Milestone{},
		&domain.MilestoneSubmission{},
		&domain.MilestoneEvidence{},
	)

	if err != nil {
		log.Fatalf("error on runing migration %v", err.Error())
	}

	log.Println("migration was successful")

	// run seeders
	err = seeder.SeedProjectCategories(db)
	if err != nil {
		log.Fatalf("error running seeders %v", err)
	}

	// cors configuration
	c := cors.New(cors.Config{
		AllowOrigins: []string{cfg.BaseURL},
		AllowHeaders: []string{"Content-Type", "Accept", "Authorization"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
	})

	app.Use(c)

	app.Get("/", func(c fiber.Ctx) error {
		return rest.SuccessResponse(c, "I am Healthy", fiber.Map{
			"status": "ok with 200 status code",
		})
	})
	notificationClient := notification.NewNotificationClient(cfg)
	auth := helper.SetupAuth(cfg.AppSecret)

	middleware := rest.SetupMiddleware(auth)

	rh := &rest.RestHandler{
		App:          app,
		DB:           db,
		Auth:         auth,
		Config:       cfg,
		Notification: notificationClient,
		Middlewares:  middleware,
	}

	setupRoutes(rh)

	log.Fatal(app.Listen(":" + cfg.ServerPort))

}

func setupRoutes(rh *rest.RestHandler) {
	// user handler
	handlers.SetupUserRoutes(rh)

	handlers.SetupProjectRoutes(rh)

}
