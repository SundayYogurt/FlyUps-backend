package api

import (
	"flyup/config"
	"flyup/internal/api/rest"
	"flyup/internal/api/rest/handler"
	"flyup/internal/domain"
	"flyup/internal/helper"
	"flyup/internal/repository"
	"flyup/internal/service"
	"flyup/pkg/notification"
	"flyup/pkg/redis"
	"log"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func StartServer(cfg config.AppConfig) {
	app := fiber.New(fiber.Config{
		BodyLimit: 50 * 1024 * 1024, // 50 MB
	})

	log.Println("DSN =", cfg.Dsn)
	db, err := gorm.Open(postgres.Open(cfg.Dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("database connection error %v\n", err)
	}
	log.Println("database connected")
	
	// FIX: ป้องกันปัญหา Duplicate Slug ตอนทำ Migration (กรณีมีข้อมูลเดิมอยู่แล้ว)
	// เราจะเพิ่ม column slug แบบธรรมดาก่อน (ถ้ายังไม่มี) แล้วไล่แก้ตัวที่ว่างให้มีค่า
	_ = db.Exec("ALTER TABLE projects ADD COLUMN IF NOT EXISTS slug text DEFAULT ''")
	var legacyProjects []domain.Project
	if err := db.Where("slug = '' OR slug IS NULL").Find(&legacyProjects).Error; err == nil && len(legacyProjects) > 0 {
		log.Printf("found %d projects with empty slug, generating slugs...", len(legacyProjects))
		for _, p := range legacyProjects {
			newSlug := helper.GenerateProjectSlug(p.Title, p.ID)
			db.Model(&domain.Project{}).Where("id = ?", p.ID).Update("slug", newSlug)
		}
	}

	// run migration
	err = db.AutoMigrate(
		&domain.User{},
		&domain.University{},
		&domain.UniversityDomain{},
		&domain.UserConsent{},
		&domain.StudentProfile{},
		&domain.BankAccount{},
		&domain.IdCardVerification{},
		&domain.StudentCardVerification{},

		// project
		&domain.Project{},
		&domain.ProjectMedia{},
		&domain.StorySection{},
		&domain.ProjectFAQ{},
		&domain.Milestone{},
		&domain.MilestoneVote{},
		&domain.ProjectInvestment{},
		&domain.ProjectUpdate{},
		&domain.ProjectThread{},
		&domain.ProjectThreadMessage{},
		&domain.ProjectCategory{},

		// investment & payment
		&domain.Investment{},
		&domain.Transaction{},
		&domain.Disbursement{},

		// notification
		&domain.Notification{},

		// complaints
		&domain.Complaint{},

		//meeting
		&domain.Meeting{},

		// profit pools
		&domain.ProfitPool{},
		&domain.InvestorProfitPayout{},

		// chat
		&domain.ChatSession{},
		&domain.ChatMessage{},
		&domain.ChatAction{},
		&domain.ChatSession{},
		&domain.ChatMessage{},
		&domain.ChatAction{},

		// admin logs
		&domain.AdminLog{},
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

	// Security headers
	app.Use(func(c fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "geolocation=(), camera=(), microphone=()")
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Set("Server", "")
		return c.Next()
	})

	app.Get("/", HealthCheck)
	app.Get("/swagger/doc.json", swaggerJSON)
	app.Get("/swagger*", swaggerUI)

	notificationClient := notification.NewNotificationClient(cfg)
	notifSvc := service.NewNotificationService(repository.NewNotificationRepository(db))
	auth := helper.SetupAuth(cfg.AppSecret)
	userRepo := repository.NewUserRepository(db)
	middleware := rest.SetupMiddleware(auth, userRepo)

	cloudinarySvc, err := helper.NewCloudinary(
		cfg.CloudinaryCloudName,
		cfg.CloudinaryAPIKey,
		cfg.CloudinaryAPISecret,
	)
	if err != nil {
		log.Fatalf("cloudinary init error %v", err)
	}

	rdb := redis.NewRedisClient(
		cfg.RedisAddr,
		cfg.RedisPassword,
		cfg.RedisDB,
	)

	cacheClient := rdb
	if cacheClient == nil {
		log.Fatal("redis client not initialized")
	}

	if err := rdb.Ping(); err != nil {
		log.Fatal("Redis connection failed:", err)
	}

	log.Println("Redis connected!")

	validate := validator.New()

	adminLogSvc := service.NewAdminLogService(repository.NewAdminLogRepository(db))

	rh := &rest.RestHandler{
		App:          app,
		DB:           db,
		Auth:         auth,
		Config:       cfg,
		Notification: notificationClient,
		Middlewares:  middleware,
		Validator:    validate,
		Cloudinary:   cloudinarySvc,
		NotifSvc:     notifSvc,
		Cache:        cacheClient,
		AdminLogSvc:  adminLogSvc,
	}

	// Background lifecycle job:
	// auto mark funding projects as failed when end date passed and funding < softcap.
	investmentRepo := repository.NewInvestmentRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	disbursementRepo := repository.NewDisbursementRepository(db)
	investmentSvc := service.NewInvestmentService(
		repository.NewProjectRepository(db),
		investmentRepo,
		transactionRepo,
		userRepo,
		disbursementRepo,
		cfg.StripeSecretKey,
		cfg.StripeWebhookSecret,
		notifSvc,
		notificationClient,
	)
	rh.InvestmentSvc = investmentSvc
	projectSvc := service.NewProjectService(
		repository.NewProjectRepository(db),
		repository.NewUserRepository(db),
		cloudinarySvc,
		notifSvc,
		notificationClient,
		investmentSvc,
		repository.NewDisbursementRepository(db),
	)
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			if err := projectSvc.AutoProjectLifecycleTick(time.Now().UTC()); err != nil {
				log.Printf("auto-expire funding job error: %v", err)
			}
			<-ticker.C
		}
	}()

	setupRoutes(rh)

	log.Fatal(app.Listen(":" + cfg.ServerPort))

}

func setupRoutes(rh *rest.RestHandler) {
	handler.SetupUserRoutes(rh)
	handler.SetupProjectRoutes(rh)
	handler.SetupInvestmentRoutes(rh)
	handler.SetupDisbursementRoutes(rh)
	handler.SetupUploadRoutes(rh)
	handler.SetupNotificationRoutes(rh)
	handler.SetupComplaintRoutes(rh)
	handler.SetupProfitPoolRoutes(rh)
	handler.SetupChatRoutes(rh)
	handler.SetupAdminBadgeRoutes(rh)
	handler.SetupPioneerBadgeRoutes(rh)
	handler.SetupBoosterBadgeRoutes(rh)
	handler.SetupFinancialRoutes(rh)
	handler.SetupProjectFinancialRoutes(rh)
	handler.SetupAdminLogRoutes(rh)
}

func HealthCheck(ctx fiber.Ctx) error {
	return ctx.Status(200).JSON(fiber.Map{
		"message": "Healthy",
	})
}

func swaggerUI(ctx fiber.Ctx) error {
	html := `<!DOCTYPE html>
<html>
  <head>
    <title>FlyUps API Docs</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
      window.onload = function() {
        window.ui = SwaggerUIBundle({
          url: "/swagger/doc.json",
          dom_id: '#swagger-ui',
          presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
          layout: "BaseLayout",
          responseInterceptor: function(response) {
            if (response.url.includes("/signin") && response.status === 200) {
              try {
                const data = JSON.parse(response.data);
                if (data.token) {
                  window.ui.preauthorizeApiKey("BearerAuth", "Bearer " + data.token);
                }
              } catch (e) {
                console.error("Swagger Auto-Login Error:", e);
              }
            }
            return response;
          }
        })
      }
    </script>
  </body>
</html>`
	ctx.Set("Content-Type", "text/html")
	return ctx.SendString(html)
}

func swaggerJSON(ctx fiber.Ctx) error {
	return ctx.SendFile("./docs/swagger.json")
}
