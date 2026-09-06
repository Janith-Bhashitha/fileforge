package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/audit"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/auth"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/mailer"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/storage"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/users"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/validate"
)

const minPasswordLength = 8

type AuthHandler struct {
	users       *users.Service
	jwtSecret   string
	audit       *audit.Recorder
	mailer      *mailer.Mailer
	store       storage.Store
	frontendURL string
	logger      *slog.Logger
}

func NewAuthHandler(userService *users.Service, jwtSecret string, recorder *audit.Recorder, m *mailer.Mailer, store storage.Store, frontendURL string, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{users: userService, jwtSecret: jwtSecret, audit: recorder, mailer: m, store: store, frontendURL: frontendURL, logger: logger}
}

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	if len(req.Password) < minPasswordLength {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("password must be at least %d characters", minPasswordLength))
		return
	}

	u, err := h.users.Register(r.Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		if errors.Is(err, users.ErrEmailTaken) {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to register user")
		return
	}

	token, err := auth.IssueToken(h.jwtSecret, u.ID, u.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	h.audit.Record(r.Context(), audit.Event{
		UserID: &u.ID, Action: audit.ActionUserRegistered,
		ResourceType: "user", ResourceID: &u.ID,
		Metadata: map[string]any{"email": u.Email},
	})

	writeJSON(w, http.StatusCreated, authResponse{Token: token})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	u, err := h.users.Authenticate(r.Context(), req.Email, req.Password)
	if err != nil {
		// Failed logins are recorded without a user ID (the account may not
		// exist at all) but with the attempted address, which is what makes
		// a credential-stuffing pattern visible later.
		h.audit.Record(r.Context(), audit.Event{
			Action:   audit.ActionLoginFailed,
			Metadata: map[string]any{"email": req.Email},
		})
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := auth.IssueToken(h.jwtSecret, u.ID, u.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	h.audit.Record(r.Context(), audit.Event{
		UserID: &u.ID, Action: audit.ActionUserLoggedIn,
		ResourceType: "user", ResourceID: &u.ID,
	})

	writeJSON(w, http.StatusOK, authResponse{Token: token})
}

type meResponse struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
}

func (h *AuthHandler) meResponseFor(u *users.User) meResponse {
	resp := meResponse{ID: u.ID, Email: u.Email, DisplayName: u.DisplayName}
	if u.AvatarKey != nil {
		url := "/avatars/" + *u.AvatarKey
		resp.AvatarURL = &url
	}
	return resp
}

// Me previously read only the JWT claims (id, email) - whatever was
// embedded when the token was issued, which is why a display name set
// after login never showed up. It looks the user up fresh instead.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	u, err := h.users.GetByID(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch user")
		return
	}

	writeJSON(w, http.StatusOK, h.meResponseFor(u))
}

type updateProfileRequest struct {
	DisplayName string `json:"display_name"`
}

// UpdateProfile is the endpoint that never existed - display_name was
// write-once at registration. Avatar is updated separately (UploadAvatar,
// a multipart route) since it's a file, not a JSON field.
func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "display_name is required")
		return
	}

	u, err := h.users.GetByID(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch user")
		return
	}

	if err := h.users.UpdateProfile(r.Context(), claims.UserID, req.DisplayName, u.AvatarKey); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update profile")
		return
	}

	u.DisplayName = req.DisplayName
	writeJSON(w, http.StatusOK, h.meResponseFor(u))
}

// UploadAvatar is deliberately a separate endpoint from UpdateProfile:
// mixing a file upload into a JSON profile update would mean either two
// content types on one route or shoehorning image bytes into base64 JSON.
func (h *AuthHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 5<<20) // 5MB - a profile picture, not a document
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart upload")
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		writeError(w, http.StatusBadRequest, "avatar file is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read upload")
		return
	}

	mimeType, err := validate.DetectAndValidate(data)
	if err != nil || (mimeType != "image/jpeg" && mimeType != "image/png") {
		writeError(w, http.StatusUnsupportedMediaType, "avatar must be a JPEG or PNG image")
		return
	}

	key, err := h.store.Save(r.Context(), claims.UserID, data, filepath.Ext(header.Filename))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store avatar")
		return
	}

	u, err := h.users.GetByID(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch user")
		return
	}
	oldAvatarKey := u.AvatarKey

	if err := h.users.UpdateProfile(r.Context(), claims.UserID, u.DisplayName, &key); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save avatar")
		return
	}

	// Best-effort: an old avatar left behind on a failed delete is unused
	// storage, not a correctness problem, so it isn't worth failing the
	// request over.
	if oldAvatarKey != nil {
		_ = h.store.Delete(r.Context(), *oldAvatarKey)
	}

	u.AvatarKey = &key
	writeJSON(w, http.StatusOK, h.meResponseFor(u))
}

// ServeAvatar is intentionally unauthenticated: avatars are rendered as
// plain <img> tags, which can't attach an Authorization header, and the
// storage key is an opaque UUID rather than anything guessable or
// sequential - the same exposure trade-off already accepted for every
// other storage key in this app.
func (h *AuthHandler) ServeAvatar(w http.ResponseWriter, r *http.Request) {
	// "*" not a named parameter: the key is namespaced by owner and so spans
	// several path segments. The store rejects unsafe keys itself.
	key := chi.URLParam(r, "*")
	if key == "" {
		writeError(w, http.StatusBadRequest, "missing avatar key")
		return
	}

	localPath, release, err := h.store.Fetch(r.Context(), key)
	if err != nil {
		writeError(w, http.StatusNotFound, "avatar not found")
		return
	}
	defer release()

	http.ServeFile(w, r, localPath)
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.NewPassword) < minPasswordLength {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("password must be at least %d characters", minPasswordLength))
		return
	}

	if err := h.users.ChangePassword(r.Context(), claims.UserID, req.CurrentPassword, req.NewPassword); err != nil {
		if errors.Is(err, users.ErrWrongCurrentPassword) {
			writeError(w, http.StatusUnauthorized, "current password is incorrect")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to change password")
		return
	}

	h.audit.Record(r.Context(), audit.Event{
		UserID: &claims.UserID, Action: "user.password_changed",
		ResourceType: "user", ResourceID: &claims.UserID,
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

// ForgotPassword returns the same response whether or not the email has an
// account - anything else turns "forgot password" into a way to check
// which addresses are registered.
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	u, token, err := h.users.RequestPasswordReset(r.Context(), req.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to process request")
		return
	}

	if u != nil {
		resetURL := h.frontendURL + "/reset-password?token=" + token
		if h.mailer.Configured() {
			// Off the request path: mail is only sent when the address has
			// an account, so waiting here would make a registered email
			// measurably slower to answer - a timing oracle giving away
			// exactly what the identical response body withholds.
			email, url := u.Email, resetURL
			go func() {
				if err := h.mailer.SendPasswordReset(email, url); err != nil {
					h.logger.Error("failed to send password reset email", "error", err)
				}
			}()
		} else {
			// No SMTP configured - this is the dev/demo fallback, not a
			// silent failure: the link is right here in the server log.
			h.logger.Info("password reset requested (SMTP not configured, logging link instead)", "email", u.Email, "reset_url", resetURL)
		}
		h.audit.Record(r.Context(), audit.Event{
			UserID: &u.ID, Action: "user.password_reset_requested",
			ResourceType: "user", ResourceID: &u.ID,
		})
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "if that email has an account, a reset link has been sent"})
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.NewPassword) < minPasswordLength {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("password must be at least %d characters", minPasswordLength))
		return
	}

	if err := h.users.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		if errors.Is(err, users.ErrInvalidResetToken) {
			writeError(w, http.StatusBadRequest, "this reset link is invalid or has expired")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to reset password")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "password updated"})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
