package handlers

import (
	"errors"
	"flyup/internal/api/rest"
	"flyup/internal/dto"
	"flyup/internal/helper"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"flyup/config"
	"flyup/internal/repository"
	"flyup/internal/service"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"golang.org/x/oauth2"
)

type UserHandler struct {
	svc         service.UserService
	validator   *validator.Validate
	auth        helper.Auth
	googleOAuth *oauth2.Config
	config      config.AppConfig
}

func SetupUserRoutes(rh *rest.RestHandler) {

	app := rh.App

	// create an instance of user service & inject to handler
	svc := service.NewUserService(
		repository.NewUserRepository(rh.DB),
		repository.NewUniversityRepository(rh.DB),
		rh.Auth,
		rh.Config,
		rh.NotifSvc,
	)

	// Setup Google OAuth
	googleOAuth := helper.SetupGoogleOAuth(rh.Config)

	handler := UserHandler{
		svc:         svc,
		validator:   rh.Validator,
		auth:        rh.Auth,
		googleOAuth: googleOAuth,
		config:      rh.Config,
	}

	pubRoutes := app.Group("/")
	pubRoutes.Post("/Signup", handler.SignUp)
	pubRoutes.Get("/verify-email", handler.VerifyEmail)
	pubRoutes.Post("/signin", handler.Signing)
	pubRoutes.Post("/forgot-password", handler.ForgotPassword)
	pubRoutes.Post("/reset-password", handler.SetPassword)

	pubRoutes.Get("/auth/google", handler.GoogleLogin)
	pubRoutes.Get("/auth/google/callback", handler.GoogleCallback)

	//private route
	privateRoutes := app.Group("/user", rh.Middlewares.Authorize)
	privateRoutes.Get("/me", handler.Me)
	privateRoutes.Patch("/profile", handler.UpdateProfile)
	privateRoutes.Post("/student-verify", handler.VerifyStudent)
	privateRoutes.Post("/id-verify", handler.VerifyIDCard)
	privateRoutes.Post("/add-bank", handler.AddBankAccount)
	privateRoutes.Patch("/update-bank/:id", handler.UpdateBankAccount)
	privateRoutes.Post("/signout", handler.SignOut)
	privateRoutes.Put("/change-password", handler.ChangePassword)
	privateRoutes.Put("/add-password", handler.AddPassword)

	//admin route
	adminRoutes := app.Group("/admin", rh.Middlewares.AuthorizeAdmin)
	adminRoutes.Get("/user-banks/:id", handler.GetBankUserAccounts)
	adminRoutes.Patch("/approve-student-card/:id", handler.ApproveStudentCard)
	adminRoutes.Patch("/approve-id-card/:id", handler.ApproveCardID)
	adminRoutes.Patch("/reject-student-card/:id", handler.RejectStudentCard)
	adminRoutes.Patch("/reject-id-card/:id", handler.RejectCardID)
	adminRoutes.Patch("/suspend-user/:id", handler.SuspendUser)
	adminRoutes.Patch("/rollback-user/:id", handler.RollbackUser)
	adminRoutes.Post("/create-university", handler.CreateUniversity)
	adminRoutes.Put("update-university/:id", handler.UpdateUniversity)
	adminRoutes.Get("/university/:id", handler.GetUniversity)
	adminRoutes.Get("/universities", handler.GetUniversities)
	adminRoutes.Delete("/delete-university/:id", handler.DeleteUniversity)
	adminRoutes.Post("/create-university-domain/:id", handler.CreateUniversityDomain)
	adminRoutes.Put("/update-university-domain/:id", handler.UpdateUniversityDomain)
	adminRoutes.Delete("/delete-university-domain/:id", handler.DeleteUniversityDomain)
	adminRoutes.Get("/student-verifications", handler.GetStudentsCardRequest)
	adminRoutes.Get("/id-card-verifications", handler.GetCardIDRequests)
	adminRoutes.Get("/list-users", handler.ListUsers)
}

// SignUp godoc
// @Summary Register a new user
// @Description Create a new user account
// @Tags Users
// @Accept json
// @Produce json
// @Param request body dto.UserSignUp true "SignUp Request body"
// @Success 201 {object} object "Success message"
// @Failure 400 {object} object "Validation failed"
// @Failure 409 {object} object "User already registered"
// @Failure 500 {object} object "Internal Server Error"
// @Router /SignUp [post]
func (h *UserHandler) SignUp(ctx fiber.Ctx) error {
	user := dto.UserSignUp{}

	//Bind JSON Body
	if err := ctx.Bind().Body(&user); err != nil {
		// ใช้ ErrorMessage เพื่อส่ง Error จากการ Bind กลับไปตรงๆ
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}

	//Validate Data
	if err := h.validator.Struct(user); err != nil {
		// ใช้ BadRequestError พร้อมบอกรายละเอียดการ Validate
		return rest.BadRequestError(ctx, "Validation failed: "+err.Error())
	}

	//Call Service Logic
	msg, err := h.svc.SignUp(user)
	if err != nil {
		errStr := err.Error()

		//กรณีข้อมูลซ้ำ (409 Conflict)
		if strings.Contains(errStr, "already registered") {
			return rest.ErrorMessage(ctx, http.StatusConflict, err)
		}

		// กรณี Business Logic ไม่ผ่าน (400 Bad Request)
		// เพิ่มเช็คคำว่า "record not found" หรือ "domain"
		if strings.Contains(errStr, "password") ||
			strings.Contains(errStr, "มหาวิทยาลัย") ||
			strings.Contains(errStr, "domain") ||
			strings.Contains(errStr, "not found") ||
			strings.Contains(errStr, "invalid email") {
			return rest.BadRequestError(ctx, errStr)
		}

		// กรณี Error อื่นๆ (500 Internal Error)
		log.Printf("[SignUp Error]: %v", err)
		return rest.InternalError(ctx, err)
	}

	// Success Response (201 Created)
	return rest.SuccessResponse(ctx, msg, nil)
}

// VerifyEmail godoc
// @Summary Verify user email
// @Description Verify email address using the token
// @Tags Users
// @Accept json
// @Produce json
// @Param token query string true "Verification Token"
// @Success 200 {object} object "Success message"
// @Failure 400 {object} object "Token missing or invalid"
// @Router /verify-email [get]
func (h *UserHandler) VerifyEmail(ctx fiber.Ctx) error {

	token := ctx.Query("token")

	if token == "" {
		return rest.BadRequestError(ctx, "token is required")
	}

	req := dto.VerifyEmailRequest{
		Token: token,
	}

	msg, err := h.svc.VerifyEmail(req)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, msg, nil)
}

// Signing godoc
// @Summary User Signing
// @Description Authenticate a user and return login token
// @Tags Users
// @Accept json
// @Produce json
// @Param request body dto.UserSigning true "Signing Request body"
// @Success 200 {object} object "Token information"
// @Failure 400 {object} object "Invalid input"
// @Failure 401 {object} object "Incorrect credentials"
// @Failure 403 {object} object "Email not verified"
// @Router /signin [post]
func (h *UserHandler) Signing(ctx fiber.Ctx) error {
	signingInput := dto.UserSigning{}
	err := ctx.Bind().Body(&signingInput)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{
			"message": "please provide valid inputs",
		})
	}
	token, err := h.svc.Signing(signingInput.Email, signingInput.Password)

	if err != nil {
		errMsg := err.Error()

		// แยก error แต่ละประเภท
		switch errMsg {
		case "please verify your email first":
			return ctx.Status(http.StatusForbidden).JSON(fiber.Map{
				"message": errMsg,
			})
		case "your account has been suspended":
			return ctx.Status(http.StatusForbidden).JSON(fiber.Map{
				"message": errMsg,
			})
		case "this email is registered with Google. Please use 'Continue with Google' to login":
			return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{
				"message":      errMsg,
				"login_method": "google",
			})
		default:
			// invalid email or password
			log.Println("signin error:", err)
			return ctx.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"message": "invalid email or password",
			})
		}
	}
	// set cookie
	ctx.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    token,
		HTTPOnly: true,
		Secure:   true,   // false ถ้า localhost
		SameSite: "None", // required for cross-site cookie on Safari/iOS
		Path:     "/",
		MaxAge:   60 * 60 * 24, // 1 day
	})

	return ctx.Status(http.StatusOK).JSON(fiber.Map{
		"message": "login",
		"token":   token,
	})
}

// ForgotPassword godoc
// @Summary Forgot Password
// @Description Send password reset link to user email
// @Tags Users
// @Accept json
// @Produce json
// @Param request body dto.ForgotPasswordRequest true "Forgot Password Request body"
// @Success 200 {object} object "Reset link sent message"
// @Failure 400 {object} object "Invalid email or user not found"
// @Router /forgot-password [post]
func (h *UserHandler) ForgotPassword(ctx fiber.Ctx) error {
	var req dto.ForgotPasswordRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid email")
	}

	if err := h.svc.ForgotPassword(req.Email); err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}
	return rest.SuccessResponse(ctx, "reset link sent", nil)
}

// SetPassword godoc
// @Summary Reset Password
// @Description Set a new password using reset token
// @Tags Users
// @Accept json
// @Produce json
// @Param reset_token query string true "Reset Token"
// @Param request body object true "New Password Request, e.g., {\"new_password\": \"password123\"}"
// @Success 200 {object} object "Password updated successfully"
// @Failure 400 {object} object "Invalid input or token"
// @Router /reset-password [post]
func (h *UserHandler) SetPassword(ctx fiber.Ctx) error {

	token := strings.TrimSpace(ctx.Query("reset_token"))

	var body struct {
		NewPassword string `json:"new_password"`
	}

	if err := ctx.Bind().Body(&body); err != nil {
		return rest.BadRequestError(ctx, "invalid json body")
	}

	if token == "" || strings.TrimSpace(body.NewPassword) == "" {
		return rest.BadRequestError(ctx, "token and new_password are required")
	}

	if err := h.svc.SetPassword(token, body.NewPassword); err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "password updated successfully", nil)
}

// Me godoc
// @Summary Get User Profile
// @Description Get current logged-in user profile
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} object "User Profile Data"
// @Failure 401 {object} object "Unauthorized"
// @Failure 500 {object} object "Internal Server Error"
// @Router /user/me [get]
func (h *UserHandler) Me(ctx fiber.Ctx) error {

	user := h.auth.GetCurrentUser(ctx)

	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	profile, err := h.svc.GetProfile(user.ID)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "success", profile)
}

// UpdateProfile godoc
// @Summary Update User Profile
// @Description Update the current user's profile information
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.ProfileInput true "Profile Update Map"
// @Success 200 {object} object "profile updated successfully"
// @Failure 400 {object} object "Validation failed or invalid input"
// @Failure 401 {object} object "Unauthorized"
// @Failure 500 {object} object "Internal Server Error"
// @Router /user/profile [patch]
func (h *UserHandler) UpdateProfile(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}
	log.Printf("[UpdateProfile] current user: id=%d email=%s role=%s", user.ID, user.Email, user.Role)
	// bind JSON body
	var req dto.ProfileInput
	if err := ctx.Bind().Body(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid request body: "+err.Error())
	}

	// validate input
	if err := h.validator.Struct(req); err != nil {
		return rest.BadRequestError(ctx, "validation failed: "+err.Error())
	}

	log.Printf("REQ: %+v", req)

	// call service
	if err := h.svc.UpdateProfile(user.ID, req); err != nil {
		// business/input errors -> 400 เพื่อ debug ง่าย
		errStr := err.Error()
		if strings.Contains(errStr, "invalid") ||
			strings.Contains(errStr, "not found") ||
			strings.Contains(errStr, "cannot") ||
			strings.Contains(errStr, "validation") ||
			strings.Contains(errStr, "domain") ||
			strings.Contains(errStr, "university") {
			return rest.BadRequestError(ctx, errStr)
		}
		return rest.InternalError(ctx, err)
	}

	// response
	return rest.SuccessResponse(ctx, "profile updated successfully", nil)
}

// VerifyStudent godoc
// @Summary Submit student verification
// @Description Submit request to verify user's student status
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.VerifyStudentInput true "Student Verification Input"
// @Success 200 {object} object "successfully submit verify to admin!"
// @Failure 400 {object} object "Invalid input"
// @Failure 401 {object} object "Unauthorized"
// @Router /user/student-verify [post]
func (h *UserHandler) VerifyStudent(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	var req dto.VerifyStudentInput
	if err := ctx.Bind().Body(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid request body: "+err.Error())
	}

	// validate input
	if err := h.validator.Struct(req); err != nil {
		return rest.BadRequestError(ctx, "validation failed: "+err.Error())
	}

	err := h.svc.VerifyStudent(user.ID, req)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "successfully submit verify to admin!", nil)
}

// VerifyIDCard godoc
// @Summary Submit ID card verification
// @Description Submit ID card info for verification
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.VerifyIDInput true "ID Card Verification Input"
// @Success 200 {object} object "verification submitted successfully"
// @Failure 400 {object} object "Invalid request"
// @Failure 401 {object} object "Unauthorized"
// @Router /user/id-verify [post]
func (h *UserHandler) VerifyIDCard(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	var req dto.VerifyIDInput
	if err := ctx.Bind().Body(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid request body: "+err.Error())
	}

	// validate input
	if err := h.validator.Struct(req); err != nil {
		return rest.BadRequestError(ctx, "validation failed: "+err.Error())
	}

	err := h.svc.VerifyID(user.ID, req)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "verification submitted successfully", nil)
}

// AddBankAccount godoc
// @Summary Add bank account
// @Description Add a new bank account to user profile
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.BankRequest true "Bank Account Request"
// @Success 200 {object} object "bank account added successfully"
// @Failure 400 {object} object "Invalid request"
// @Failure 401 {object} object "Unauthorized"
// @Failure 500 {object} object "Internal Error"
// @Router /user/add-bank [post]
func (h *UserHandler) AddBankAccount(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	var req dto.BankRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid request body: "+err.Error())
	}

	err := h.svc.AddBankAccount(user.ID, req)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "bank account added successfully", nil)
}

// UpdateBankAccount godoc
// @Summary Update bank account
// @Description Update an existing bank account
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Bank Account ID"
// @Param request body dto.BankRequest true "Bank Account Details"
// @Success 200 {object} object "bank account updated successfully"
// @Failure 400 {object} object "Invalid request"
// @Failure 401 {object} object "Unauthorized"
// @Router /user/update-bank/{id} [patch]
func (h *UserHandler) UpdateBankAccount(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	bankId := ctx.Params("id")
	bankIdParsed, err := strconv.ParseUint(bankId, 10, 64)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid Bank ID: "+err.Error())
	}

	var req dto.BankRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid request body: "+err.Error())
	}

	err = h.svc.UpdateBankAccount(user.ID, uint(bankIdParsed), req)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}
	return rest.SuccessResponse(ctx, "bank account updated successfully", nil)
}

// GetBankUserAccounts godoc
// @Summary Admin get user bank accounts
// @Description Get bank accounts for a user (admin only)
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"f
// @Success 200 {object} object "List of bank accounts"
// @Failure 400 {object} object "Invalid request"
// @Router /admin/user-banks/{id} [get]
func (h *UserHandler) GetBankUserAccounts(ctx fiber.Ctx) error {
	user := ctx.Params("id")
	userParsed, err := strconv.ParseUint(user, 10, 64)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid user ID: "+err.Error())
	}

	users, err := h.svc.FindBankByUserID(uint(userParsed))
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "success", users)
}

// ApproveStudentCard godoc
// @Summary Approve student card
// @Description Admin approves a student card verification
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} object "student card approved"
// @Failure 400 {object} object "Invalid request"
// @Failure 401 {object} object "Unauthorized"
// @Router /admin/approve-student-card/{id} [patch]
func (h *UserHandler) ApproveStudentCard(ctx fiber.Ctx) error {
	admin := h.auth.GetCurrentUser(ctx)
	if admin.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	userIDParam := ctx.Params("id")

	userID, err := strconv.ParseUint(userIDParam, 10, 64)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid user ID: "+err.Error())
	}

	err = h.svc.ApproveStudentCard(uint(userID), admin.ID)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "student card approved", map[string]interface{}{
		"user_id": userID,
	})
}

// ApproveCardID godoc
// @Summary Approve ID card
// @Description Admin approves an ID card verification
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} object "id card approved"
// @Failure 400 {object} object "Invalid request"
// @Failure 401 {object} object "Unauthorized"
// @Router /admin/approve-id-card/{id} [patch]
func (h *UserHandler) ApproveCardID(ctx fiber.Ctx) error {
	admin := h.auth.GetCurrentUser(ctx)
	if admin.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	userIDParam := ctx.Params("id")

	userID, err := strconv.ParseUint(userIDParam, 10, 64)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid user ID: "+err.Error())
	}

	err = h.svc.ApproveIdCard(uint(userID), admin.ID)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "id card approved", map[string]interface{}{
		"user_id": userID,
	})
}

// RejectStudentCard godoc
// @Summary Reject student card
// @Description Admin rejects a student card verification
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} object "student card rejected"
// @Failure 400 {object} object "Invalid request"
// @Failure 401 {object} object "Unauthorized"
// @Router /admin/reject-student-card/{id} [patch]
func (h *UserHandler) RejectStudentCard(ctx fiber.Ctx) error {
	admin := h.auth.GetCurrentUser(ctx)
	if admin.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	userIDParam := ctx.Params("id")

	userID, err := strconv.ParseUint(userIDParam, 10, 64)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid user ID: "+err.Error())
	}

	err = h.svc.RejectStudentCard(uint(userID), admin.ID)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "student card rejected", map[string]interface{}{
		"user_id": userID,
	})
}

// RejectCardID godoc
// @Summary Reject ID card
// @Description Admin rejects an ID card verification
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} object "id card rejected"
// @Failure 400 {object} object "Invalid request"
// @Failure 401 {object} object "Unauthorized"
// @Router /admin/reject-id-card/{id} [patch]
func (h *UserHandler) RejectCardID(ctx fiber.Ctx) error {
	admin := h.auth.GetCurrentUser(ctx)
	if admin.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	userIDParam := ctx.Params("id")

	userID, err := strconv.ParseUint(userIDParam, 10, 64)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid user ID: "+err.Error())
	}

	err = h.svc.RejectIdCard(uint(userID), admin.ID)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "id card rejected", map[string]interface{}{
		"user_id": userID,
	})
}

// SuspendUser godoc
// @Summary Suspend user account
// @Description Admin suspends a user account with a reason
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param request body dto.SuspendUserInput true "Suspend reason"
// @Success 200 {object} object "user suspended"
// @Failure 400 {object} object "Invalid request or reason"
// @Failure 401 {object} object "Unauthorized"
// @Router /admin/suspend-user/{id} [patch]
func (h *UserHandler) SuspendUser(ctx fiber.Ctx) error {
	admin := h.auth.GetCurrentUser(ctx)
	if admin.ID == 0 {
		return rest.ErrorMessage(ctx, 401, errors.New("unauthorized"))
	}

	userIDParam := ctx.Params("id")

	userID, err := strconv.ParseUint(userIDParam, 10, 64)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid user id")
	}

	var req dto.SuspendUserInput
	if err := ctx.Bind().Body(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid body")
	}

	if err := h.validator.Struct(req); err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	if err := h.svc.SuspendUser(admin.ID, uint(userID), req.Reason); err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "user suspended", nil)
}

// RollbackUser godoc
// @Summary Rollback suspended user
// @Description Admin restores a suspended user back to active status
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} object "user come back to active!"
// @Failure 400 {object} object "Invalid user ID or rollback error"
// @Failure 401 {object} object "Unauthorized"
// @Router /admin/rollback-user/{id} [patch]
func (h *UserHandler) RollbackUser(ctx fiber.Ctx) error {
	admin := h.auth.GetCurrentUser(ctx)
	if admin.ID == 0 {
		return rest.ErrorMessage(ctx, 401, errors.New("unauthorized"))
	}

	userIDParam := ctx.Params("id")

	userID, err := strconv.ParseUint(userIDParam, 10, 64)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid user id")
	}

	if err := h.svc.RollbackActiveUser(uint(userID)); err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "user come back to active!", nil)
}

// GoogleLogin godoc
// @Summary Google Login
// @Description Redirect to Google Login
// @Tags Auth
// @Router /auth/google [get]
func (h *UserHandler) GoogleLogin(ctx fiber.Ctx) error {
	role := ctx.Query("role", "booster") // default booster
	if role != "pioneer" && role != "booster" {
		role = "booster"
	}

	nonce, err := helper.GenerateRandomToken(16)
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	statePayload := role + ":" + nonce
	sig := helper.Sha256HmacHex(statePayload, h.config.AppSecret)
	state := statePayload + ":" + sig

	url := h.googleOAuth.AuthCodeURL(state)
	return ctx.Redirect().To(url)
}

// GoogleCallback godoc
// @Summary Google Callback
// @Description Callback from Google Login
// @Tags Auth
// @Router /auth/google/callback [get]
func (h *UserHandler) GoogleCallback(ctx fiber.Ctx) error {
	baseURL := strings.TrimRight(h.config.BaseURL, "/")
	oauthFailedRedirect := baseURL + "/login?error=oauth_failed"

	state := ctx.Query("state")
	parts := strings.SplitN(state, ":", 3)
	if len(parts) != 3 {
		return ctx.Redirect().To(oauthFailedRedirect)
	}
	reqRole := parts[0]
	nonce := parts[1]
	sig := parts[2]
	if reqRole != "pioneer" && reqRole != "booster" {
		return ctx.Redirect().To(oauthFailedRedirect)
	}
	if nonce == "" || sig == "" {
		return ctx.Redirect().To(oauthFailedRedirect)
	}

	expectedSig := helper.Sha256HmacHex(reqRole+":"+nonce, h.config.AppSecret)
	if sig != expectedSig {
		return ctx.Redirect().To(oauthFailedRedirect)
	}

	code := ctx.Query("code")
	if code == "" {
		return ctx.Redirect().To(oauthFailedRedirect)
	}

	token, err := h.svc.GoogleSigning(code, reqRole, h.googleOAuth)
	if err != nil {
		return ctx.Redirect().To(oauthFailedRedirect)
	}

	ctx.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    token,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "None", // required for cross-site cookie on Safari/iOS
		Path:     "/",
		MaxAge:   3600,
	})

	// Send token to frontend
	redirectUrl := baseURL + "/?token=" + token
	return ctx.Redirect().To(redirectUrl)
}

// SignOut godoc
// @Summary Sign out
// @Description Sign out and clear the auth cookie
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} object "logout success"
// @Router /user/signout [post]
func (h *UserHandler) SignOut(ctx fiber.Ctx) error {
	ctx.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour), // ทำให้หมดอายุทันที
		MaxAge:   -1,                         // เพิ่ม MaxAge เพื่อให้แน่ใจว่าลบได้
		SameSite: "None",                     // keep same attributes when deleting
		Path:     "/",
		HTTPOnly: true,
		Secure:   true,
	})

	return ctx.JSON(fiber.Map{
		"message": "logout success",
	})
}

// UpdateUniversity godoc
// @Summary Update university
// @Description Admin updates university information
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "University ID"
// @Param request body dto.CreateUniversityRequest true "University data"
// @Success 200 {object} object "update university success"
// @Failure 400 {object} object "Invalid request"
// @Failure 401 {object} object "Unauthorized"
// @Router /admin/update-university/{id} [put]
func (h *UserHandler) UpdateUniversity(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.UnauthorizedError(ctx, "unauthorized")
	}

	uid := ctx.Params("id")
	uidParsed, err := strconv.ParseUint(uid, 10, 64)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid university id")
	}
	var req dto.CreateUniversityRequest

	if err := ctx.Bind().Body(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid body")
	}

	uni, err := h.svc.UpdateUniversity(uint(uidParsed), req)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return ctx.JSON(fiber.Map{
		"message": "update university success",
		"data":    uni,
	})
}

// CreateUniversity godoc
// @Summary Create university
// @Description Admin creates a new university
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.CreateUniversityRequest true "University data"
// @Success 200 {object} object "create university success"
// @Failure 400 {object} object "Invalid request"
// @Failure 401 {object} object "Unauthorized"
// @Router /admin/create-university [post]
func (h *UserHandler) CreateUniversity(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.UnauthorizedError(ctx, "unauthorized")
	}

	var req dto.CreateUniversityRequest

	if err := ctx.Bind().Body(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid body")
	}

	uni, err := h.svc.CreateUniversity(req)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return ctx.JSON(fiber.Map{
		"message": "create university success",
		"data":    uni,
	})
}

// GetUniversity godoc
// @Summary Get university by ID
// @Description Admin gets university details by ID
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param id path int true "University ID"
// @Success 200 {object} object "get university success"
// @Failure 400 {object} object "Invalid request"
// @Failure 401 {object} object "Unauthorized"
// @Router /admin/university/{id} [get]
func (h *UserHandler) GetUniversity(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.UnauthorizedError(ctx, "unauthorized")
	}

	uid := ctx.Params("id")
	uidParsed, err := strconv.ParseUint(uid, 10, 64)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid university id")
	}

	uni, err := h.svc.GetUniversityByID(uint(uidParsed))
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return ctx.JSON(fiber.Map{
		"message": "get university success",
		"data":    uni,
	})
}

// GetUniversities godoc
// @Summary Get all universities
// @Description Admin gets list of all universities
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} object "get universities success"
// @Failure 401 {object} object "Unauthorized"
// @Router /admin/universities [get]
func (h *UserHandler) GetUniversities(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.UnauthorizedError(ctx, "unauthorized")
	}

	uni, err := h.svc.GetAllUniversities()
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return ctx.JSON(fiber.Map{
		"message": "get universities success",
		"data":    uni,
	})
}

// DeleteUniversity godoc
// @Summary Delete university
// @Description Admin deletes a university by ID
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param id path int true "University ID"
// @Success 200 {object} object "delete university success"
// @Failure 400 {object} object "Invalid university ID"
// @Failure 401 {object} object "Unauthorized"
// @Router /admin/delete-university/{id} [delete]
func (h *UserHandler) DeleteUniversity(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.UnauthorizedError(ctx, "unauthorized")
	}

	uid := ctx.Params("id")
	uidParsed, err := strconv.ParseUint(uid, 10, 64)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid university id")
	}

	err = h.svc.DeleteUniversity(uint(uidParsed))
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return ctx.JSON(fiber.Map{
		"message": "delete university success",
	})
}

// CreateUniversityDomain godoc
// @Summary Create university domain
// @Description Admin adds a new email domain for a university
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "University ID"
// @Param request body dto.CreateDomainRequest true "Domain data"
// @Success 200 {object} object "create domain of university success"
// @Failure 400 {object} object "Invalid request"
// @Failure 401 {object} object "Unauthorized"
// @Router /admin/create-university-domain/{id} [post]
func (h *UserHandler) CreateUniversityDomain(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.UnauthorizedError(ctx, "unauthorized")
	}

	uid := ctx.Params("id")
	uidParsed, err := strconv.ParseUint(uid, 10, 64)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid university id")
	}

	var req dto.CreateDomainRequest

	if err := ctx.Bind().Body(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid body")
	}

	uni, err := h.svc.CreateDomain(uint(uidParsed), req)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return ctx.JSON(fiber.Map{
		"message": "create domain of university success",
		"data":    uni,
	})
}

// UpdateUniversityDomain godoc
// @Summary Update university domain
// @Description Admin updates an email domain for a university
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Domain ID"
// @Param request body dto.UpdateDomainRequest true "Domain data"
// @Success 200 {object} object "update domain of university success"
// @Failure 400 {object} object "Invalid request"
// @Failure 401 {object} object "Unauthorized"
// @Router /admin/update-university-domain/{id} [put]
func (h *UserHandler) UpdateUniversityDomain(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.UnauthorizedError(ctx, "unauthorized")
	}

	uid := ctx.Params("id")
	uidParsed, err := strconv.ParseUint(uid, 10, 64)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid university id")
	}

	var req dto.UpdateDomainRequest

	if err := ctx.Bind().Body(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid body")
	}

	uni, err := h.svc.UpdateDomain(uint(uidParsed), req)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return ctx.JSON(fiber.Map{
		"message": "update domain of university success",
		"data":    uni,
	})
}

// DeleteUniversityDomain godoc
// @Summary Delete university domain
// @Description Admin removes an email domain from a university
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param id path int true "Domain ID"
// @Success 200 {object} object "delete university domain success"
// @Failure 400 {object} object "Invalid domain ID"
// @Failure 401 {object} object "Unauthorized"
// @Router /admin/delete-university-domain/{id} [delete]
func (h *UserHandler) DeleteUniversityDomain(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.UnauthorizedError(ctx, "unauthorized")
	}

	uid := ctx.Params("id")
	uidParsed, err := strconv.ParseUint(uid, 10, 64)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid university id")
	}
	err = h.svc.DeleteDomain(uint(uidParsed))
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return ctx.JSON(fiber.Map{
		"message": "delete university domain success",
	})

}

// ChangePassword godoc
// @Summary Change password
// @Description Authenticated user changes their current password
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.ChangePasswordRequest true "Change password data"
// @Success 200 {object} object "change password success"
// @Failure 400 {object} object "Invalid request body"
// @Failure 401 {object} object "Unauthorized"
// @Failure 500 {object} object "Internal Server Error"
// @Router /user/change-password [put]
func (h *UserHandler) ChangePassword(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.UnauthorizedError(ctx, "unauthorized")
	}

	var req dto.ChangePasswordRequest

	if err := ctx.Bind().Body(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid body")
	}

	err := h.svc.ChangePassword(user.ID, req.OldPassword, req.NewPassword)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return ctx.JSON(fiber.Map{
		"message": "change password success",
	})

}

// GetStudentsCardRequest godoc
// @Summary Get student card verification requests
// @Description Admin gets all pending student card verification requests
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} object "get students card verify request success"
// @Failure 401 {object} object "Unauthorized"
// @Failure 500 {object} object "Internal Server Error"
// @Router /admin/student-verifications [get]
func (h *UserHandler) GetStudentsCardRequest(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.UnauthorizedError(ctx, "unauthorized")
	}

	reqs, err := h.svc.GetAllStudentVerifyRequest()
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return ctx.JSON(fiber.Map{
		"message": "get students card verify request success",
		"data":    reqs,
	})
}

// GetCardIDRequests godoc
// @Summary Get ID card verification requests
// @Description Admin gets all pending national ID card verification requests
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} object "get users cardID verify request success"
// @Failure 401 {object} object "Unauthorized"
// @Failure 500 {object} object "Internal Server Error"
// @Router /admin/id-card-verifications [get]
func (h *UserHandler) GetCardIDRequests(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.UnauthorizedError(ctx, "unauthorized")
	}

	reqs, err := h.svc.GetAllCardIDVerifyRequest()
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return ctx.JSON(fiber.Map{
		"message": "get users cardID verify request success",
		"data":    reqs,
	})
}

// AddPassword godoc
// @Summary Add password for Google user
// @Description Allows a user who signed up via Google to set a local password
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.AddPasswordRequest true "New password"
// @Success 200 {object} object "add password success"
// @Failure 400 {object} object "Invalid request"
// @Failure 401 {object} object "Unauthorized"
// @Failure 500 {object} object "Internal Server Error"
// @Router /user/add-password [put]
func (h *UserHandler) AddPassword(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.UnauthorizedError(ctx, "unauthorized")
	}
	var req dto.AddPasswordRequest

	if err := ctx.Bind().Body(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid body")
	}

	err := h.svc.AddPasswordForGoogle(user.ID, req.NewPassword)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return ctx.JSON(fiber.Map{
		"message": "add password success",
	})

}

// ListUsers godoc
// @Summary List all users
// @Description Admin gets a paginated list of users with optional filters
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Param role query string false "User role (booster/pioneer)"
// @Param status query string false "User status (active/suspended)"
// @Param search query string false "Search by email, first name, or last name"
// @Success 200 {object} object "success"
// @Failure 401 {object} object "Unauthorized"
// @Failure 500 {object} object "Internal Server Error"
// @Router /admin/list-users [get]
func (h *UserHandler) ListUsers(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.UnauthorizedError(ctx, "unauthorized")
	}

	page, err := strconv.Atoi(ctx.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	
	pageSize, err := strconv.Atoi(ctx.Query("page_size", "10"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	role := ctx.Query("role", "")
	status := ctx.Query("status", "")
	search := ctx.Query("search", "")

	users, total, err := h.svc.ListUser(page, pageSize, role, status, search)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return ctx.JSON(fiber.Map{
		"message": "success",
		"data":    users,
		"meta": fiber.Map{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}
