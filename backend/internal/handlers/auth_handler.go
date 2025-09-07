package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"strconv"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rudraprasaaad/task-scheduler/internal/auth"
	"github.com/rudraprasaaad/task-scheduler/internal/config"
	"github.com/rudraprasaaad/task-scheduler/internal/models"
	"github.com/rudraprasaaad/task-scheduler/internal/repository"
)

type AuthHandler struct {
	userRepo *repository.UserRepository
	authCfg  config.AuthConfig
}

func NewAuthHandler(userRepo *repository.UserRepository, authCfg config.AuthConfig) *AuthHandler {
	return &AuthHandler{
		userRepo: userRepo,
		authCfg:  authCfg,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user := &models.User{Email: creds.Email}
	if err := user.SetPassword(creds.Password); err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	if err := h.userRepo.Create(r.Context(), user); err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User created successfully"})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid request Body", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByEmail(r.Context(), creds.Email)
	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	if !user.CheckPassword(creds.Password) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	role := "user"

	expirationTime := time.Now().Add(h.authCfg.TokenExpiration)

	claims := &auth.Claims{
		UserID: strconv.FormatInt(user.ID, 10),
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	tokenString, err := auth.GenrerateToken(claims, h.authCfg.JWTSecret)
	if err != nil {
		http.Error(w, "Failed to create token", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    tokenString,
		Expires:  expirationTime,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Login Successful"})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Logout Successful"})
}
