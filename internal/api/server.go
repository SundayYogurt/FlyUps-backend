package api

import (
	"flyup/config"
	"flyup/internal/api/rest"
	"flyup/internal/api/rest/handlers"
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
	)

	if err != nil {
		log.Fatalf("error on runing migration %v", err.Error())
	}

	log.Println("migration was successful")

	// cors configuration
	c := cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:3000"},
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

	rh := &rest.RestHandler{
		App:          app,
		DB:           db,
		Auth:         auth,
		Config:       cfg,
		Notification: notificationClient,
	}

	setupRoutes(rh)

	log.Fatal(app.Listen(":" + cfg.ServerPort))

}

func setupRoutes(rh *rest.RestHandler) {
	// user handler
	handlers.SetupUserRoutes(rh)

}
