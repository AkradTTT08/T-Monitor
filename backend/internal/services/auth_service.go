package services

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthService provides methods for authentication logic
type AuthService interface {
	HashPassword(password string) (string, error)
	CheckPasswordHash(password, hash string) bool
	GenerateJWT(userID string, role string) (string, error)
}

type authService struct{}

// NewAuthService creates a new instance of AuthService
func NewAuthService() AuthService {
	return &authService{}
}

// HashPassword hashes a plain text password using bcrypt
func (s *authService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash compares a plain text password with a bcrypt hash
func (s *authService) CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateJWT creates a new JWT token for a user
func (s *authService) GenerateJWT(userID string, role string) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["user_id"] = userID
	claims["role"] = role
	claims["exp"] = time.Now().Add(time.Hour * 72).Unix() // Token expires in 72 hours

	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		panic("JWT_SECRET_KEY environment variable is not set")
	}

	t, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return t, nil
}
