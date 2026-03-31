package handler

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/thriftllm/backend/internal/model"
	"github.com/thriftllm/backend/internal/store"
	"golang.org/x/crypto/bcrypt"
)

type SetupHandler struct {
	DB            *store.Postgres
	JWTSecret     string
	SecureCookies bool
}

func (h *SetupHandler) Status(w http.ResponseWriter, r *http.Request) {
	complete, err := h.DB.IsSetupComplete(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check setup status")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"is_complete": complete})
}

func (h *SetupHandler) Setup(w http.ResponseWriter, r *http.Request) {
	complete, err := h.DB.IsSetupComplete(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check setup status")
		return
	}
	if complete {
		writeError(w, http.StatusConflict, "setup already completed")
		return
	}

	var req model.SetupRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "name, email and password are required")
		return
	}

	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user, err := h.DB.CreateAdminUser(r.Context(), req.Name, req.Email, string(hash))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create admin user")
		return
	}

	if err := h.DB.CompleteSetup(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to complete setup")
		return
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID.String(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})
	tokenStr, err := token.SignedString([]byte(h.JWTSecret))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "thrift_token",
		Value:    tokenStr,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.SecureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})

	writeJSON(w, http.StatusCreated, model.LoginResponse{
		Token: tokenStr,
		User:  *user,
	})
}
