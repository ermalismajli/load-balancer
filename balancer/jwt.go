package balancer

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const (
	// For this example, we'll use a simple symmetric key
	// In production, you would use proper key management
	jwtSecretKey = "your-secret-key-replace-in-production"
)

// Claims represents the JWT claims
type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// ValidateJWT validates the JWT token and returns the role
func ValidateJWT(tokenString string) (string, error) {
	if tokenString == "" {
		return "", fmt.Errorf("no token provided")
	}
	
	// Remove 'Bearer ' prefix if present
	if strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	}
	
	// Parse and validate the token
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Make sure token uses the correct signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecretKey), nil
	})
	
	if err != nil {
		return "", err
	}
	
	if !token.Valid {
		return "", fmt.Errorf("invalid token")
	}
	
	// Validate the role claim - it should be one of "User", "Client", or "Admin"
	role := claims.Role
	if role != "User" && role != "Client" && role != "Admin" {
		return "", fmt.Errorf("invalid role claim: %s", role)
	}
	
	return role, nil
}

// GenerateJWT creates a JWT token with the specified role claim
// This is helpful for testing purposes
func GenerateJWT(role string) (string, error) {
	if role != "User" && role != "Client" && role != "Admin" {
		return "", fmt.Errorf("invalid role: %s", role)
	}
	
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecretKey))
	if err != nil {
		return "", err
	}
	
	return tokenString, nil
}

