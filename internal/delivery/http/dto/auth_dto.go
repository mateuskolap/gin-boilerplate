package dto

type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=100" example:"John Doe"`
	Email    string `json:"email" binding:"required,email" example:"john.doe@example.com"`
	Password string `json:"password" binding:"required,min=8" example:"secret12345"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"john.doe@example.com"`
	Password string `json:"password" binding:"required" example:"secret12345"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"a1b2c3d4e5f6..."`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"a1b2c3d4e5f6..."`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" example:"a1b2c3d4e5f6..."`
}
