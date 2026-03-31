package api

import (
	"flyup/config"
	"flyup/internal/api/rest"
	"flyup/internal/api/rest/handlers"
	"flyup/internal/domain"
	"flyup/internal/helper"
	"flyup/pkg/notification"
	"log"

	"github.com/go-playground/validator/v10"
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
		&domain.ProjectMedia{},
		&domain.StorySection{},
		&domain.ProjectFAQ{},
		&domain.Milestone{},
		&domain.ProjectInvestment{},
		&domain.ProjectUpdate{},
		&domain.ProjectThread{},
		&domain.ProjectThreadMessage{},
		&domain.ProjectCategory{},

		// investment & payment
		&domain.Investment{},
		&domain.Transaction{},
	)

	if err != nil {
		log.Fatalf("error on running migration %v", err.Error())
	}

	log.Println("migration was successful")

	// cors configuration
	c := cors.New(cors.Config{
		AllowOrigins: []string{
			cfg.BaseURL,
			"https://www.fly-up.app",
			"https://fly-up.app",
			"http://localhost:3000", // สำหรับทดสอบ Local
			"http://localhost:5173", // สำหรับเปิดทดสอบด้วย Vite
		},
		AllowCredentials: true,
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
		},
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
			"OPTIONS",
		},
		ExposeHeaders: []string{
			"Content-Length",
		},
		MaxAge: 86400,
	})

	app.Use(c)

	app.Get("/", HealthCheck)

	notificationClient := notification.NewNotificationClient(cfg)
	auth := helper.SetupAuth(cfg.AppSecret)

	middleware := rest.SetupMiddleware(auth)

	cloudinarySvc, err := helper.NewCloudinary(
		cfg.CloudinaryCloudName,
		cfg.CloudinaryAPIKey,
		cfg.CloudinaryAPISecret,
	)
	if err != nil {
		log.Fatalf("cloudinary init error %v", err)
	}

	validate := validator.New()

	rh := &rest.RestHandler{
		App:          app,
		DB:           db,
		Auth:         auth,
		Config:       cfg,
		Notification: notificationClient,
		Middlewares:  middleware,
		Validator:    validate,
		Cloudinary:   cloudinarySvc,
	}
	setupRoutes(rh)

	log.Fatal(app.Listen(":" + cfg.ServerPort))

}

func setupRoutes(rh *rest.RestHandler) {
	// user handler
	handlers.SetupUserRoutes(rh)

	handlers.SetupProjectRoutes(rh)
	handlers.SetupInvestmentRoutes(rh)
}

func HealthCheck(ctx fiber.Ctx) error {
	return ctx.Status(200).JSON(fiber.Map{
		"message": "Healthy",
	})
}
