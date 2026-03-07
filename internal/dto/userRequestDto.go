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
