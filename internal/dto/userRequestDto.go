package dto

type UserSignup struct {
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

type UserSignin struct {
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
	FirstName string  `json:"first_name" validate:"required"`
	LastName  string  `json:"last_name" validate:"required"`
	Phone     string  `json:"phone" validate:"required"`
	Address   *string `json:"address,omitempty"` // optional

	// --- Pioneer only ---
	// UniversityID: จะถูก derive จาก domain ของ email ที่สมัคร (pioneer)
	// ยังคงไว้เพื่อ backward compatibility แต่ server จะ ignore ค่า input นี้
	UniversityID *uint   `json:"university_id,omitempty"`
	Bio          *string `json:"bio,omitempty"`
	Portfolio    *string `json:"portfolio,omitempty"`
	Skills       *string `json:"skills,omitempty"`
	Faculty      *string `json:"faculty,omitempty"`
	Major        *string `json:"major,omitempty"`

	// --- เอกสารยืนยันตัวตน ---
	IDCardFile      *string `json:"id_card_file,omitempty"`      // booster & pioneer
	StudentCardFile *string `json:"student_card_file,omitempty"` // pioneer only

	// --- ข้อมูลบัญชีธนาคาร ---
	BankName        *string `json:"bank_name,omitempty"`
	BankAccountName *string `json:"bank_account_name,omitempty"`
	BankAccountNo   *string `json:"bank_account_no,omitempty"`
	BankProofFile   *string `json:"bank_proof_file,omitempty"`
}
