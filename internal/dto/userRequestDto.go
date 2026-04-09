package dto

type UserSignUp struct {
	Role string `json:"role" validate:"required,oneof=pioneer booster"`

	// ข้อมูลส่วนตัว
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
	Email     string `json:"email" validate:"required,email"`

	// เบอร์โทรศัพท์ (ใน UI ไม่มีดอกจัน * แต่ใน DBML ระบุว่า not null ดังนั้นควรบังคับกรอก)
	Phone string `json:"phone" validate:"required"`

	// รหัสผ่าน
	Password string `json:"password" validate:"required,min=8"`

	// ยอมรับข้อตกลง
	AcceptTerms bool `json:"accept_terms" validate:"required,eq=true"`
}

type UserSigning struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type AuthResponse struct {
	UserID int     `json:"user_id"`
	Email  string  `json:"email"`
	Iat    float64 `json:"iat"`
	Expiry float64 `json:"expiry"`
}

type ProfileInput struct {
	// --- ข้อมูลส่วนตัว ---
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	Address   *string `json:"address,omitempty"`
	Picture   *string `json:"picture,omitempty"`

	// --- Pioneer only ---
	// UniversityID: จะถูก derive จาก domain ของ email ที่สมัคร (pioneer)
	// ยังคงไว้เพื่อ backward compatibility แต่ server จะ ignore ค่า input นี้
	UniversityID *uint   `json:"university_id,omitempty"`
	Bio          *string `json:"bio,omitempty"`
	Portfolio    *string `json:"portfolio,omitempty"`
	Skills       *string `json:"skills,omitempty"`
	Faculty      *string `json:"faculty,omitempty"`
	Major        *string `json:"major,omitempty"`
}

type VerifyStudentInput struct {
	StudentCardURL     *string `json:"student_card_url,omitempty" validate:"required"`
	DeclareTruth       *bool   `json:"declare_truth"`
	AcceptPioneerTerms *bool   `json:"accept_pioneer_terms"`
}

type VerifyIDInput struct {
	IDCardURL    *string `json:"id_card_url,omitempty" validate:"required"`
	SelfieURL    *string `json:"selfie_url,omitempty" validate:"required"`
	DeclareTruth *bool   `json:"declare_truth"`
}

type BankRequest struct {
	BankName      *string `json:"bank_name,omitempty"`
	AccountName   *string `json:"account_name,omitempty"`
	AccountNumber *string `json:"account_number"`
}

type SuspendUserInput struct {
	Reason string `json:"reason" validate:"required"`
}

type CreateUniversityRequest struct {
	NameTH   *string `json:"name_th"`
	NameEN   *string `json:"name_en"`
	Province *string `json:"province"`
}

type CreateDomainRequest struct {
	Domain string `json:"domain"`
}

type UpdateDomainRequest struct {
	Domain   *string `json:"domain"`
	IsActive *bool   `json:"is_active"`
}
