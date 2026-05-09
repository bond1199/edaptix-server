package response

type LoginResponse struct {
	UserID       int64  `json:"user_id"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Initialized  bool   `json:"initialized"`
}

type RegisterResponse struct {
	UserID       int64  `json:"user_id"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Initialized  bool   `json:"initialized"`
}
