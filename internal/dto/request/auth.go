package request

type RegisterRequest struct {
	Phone    string `json:"phone" binding:"required,len=11"`
	Password string `json:"password" binding:"required,min=6,max=20"`
	SMSCode  string `json:"sms_code" binding:"required,len=6"`
	RealName string `json:"real_name" binding:"required,min=2,max=50"`
	Grade    int    `json:"grade" binding:"required,min=1,max=12"`
}

type LoginRequest struct {
	Phone    string `json:"phone" binding:"required,len=11"`
	Password string `json:"password" binding:"required"`
	DeviceID string `json:"device_id"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type SendSMSRequest struct {
	Phone string `json:"phone" binding:"required,len=11"`
}

type VerifySMSRequest struct {
	Phone   string `json:"phone" binding:"required,len=11"`
	SMSCode string `json:"sms_code" binding:"required,len=6"`
}
