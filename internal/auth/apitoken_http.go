package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

const maxTokenExpiryDays = 3650

// adminMe describes the signed-in staff user for the admin SPA.
func (h HTTPHandler) adminMe(w http.ResponseWriter, r *http.Request) {
	noStoreAuthResponse(w)
	user := UserFromContext(r.Context())
	if user == nil || !user.Active {
		api.WriteError(w, r, http.StatusUnauthorized, "authentication required", nil)
		return
	}
	catalog := h.Service.AvailablePermissions()
	granted := make([]string, 0, len(catalog))
	for _, permission := range catalog {
		if h.Service.Can(user, permission) {
			granted = append(granted, permission)
		}
	}
	mfaEnabled := false
	if h.MFA != nil && h.MFADB != nil {
		if enabled, err := h.MFA.Enabled(r.Context(), h.MFADB, user.ID); err == nil {
			mfaEnabled = enabled
		}
	}
	principal := PrincipalFromContext(r.Context())
	api.WriteData(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id": user.ID, "email": user.Email, "role": user.Role, "active": user.Active,
			"email_verified_at": user.EmailVerifiedAt, "last_login_at": user.LastLoginAt, "mfa_enabled": mfaEnabled,
		},
		"permissions":        granted,
		"permission_catalog": catalog,
		"principal":          principal,
		"csrf_token":         security.CSRFTokenFromContext(r.Context()),
	}, nil)
}

type issueTokenRequest struct {
	Name          string `json:"name"`
	Scope         string `json:"scope"`
	ExpiresInDays int    `json:"expires_in_days"`
}

func (req issueTokenRequest) expiresAt() (*time.Time, string) {
	if req.ExpiresInDays == 0 {
		return nil, ""
	}
	if req.ExpiresInDays < 1 || req.ExpiresInDays > maxTokenExpiryDays {
		return nil, "invalid"
	}
	at := time.Now().UTC().Add(time.Duration(req.ExpiresInDays) * 24 * time.Hour)
	return &at, ""
}

func (h HTTPHandler) listPersonalTokens(w http.ResponseWriter, r *http.Request) {
	noStoreAuthResponse(w)
	user := UserFromContext(r.Context())
	tokens, err := h.Service.ListAPITokens(r.Context(), user.ID)
	if err != nil {
		writeTokenError(w, r, err)
		return
	}
	api.WriteData(w, http.StatusOK, tokens, nil)
}

func (h HTTPHandler) issuePersonalToken(w http.ResponseWriter, r *http.Request) {
	noStoreAuthResponse(w)
	user := UserFromContext(r.Context())
	if PrincipalFromContext(r.Context()).Kind != PrincipalSession {
		api.WriteError(w, r, http.StatusForbidden, "tokens can only be issued from a browser session", nil)
		return
	}
	var req issueTokenRequest
	if !api.DecodeJSON(w, r, &req, maxAuthFormBytes) {
		return
	}
	fields := map[string]string{}
	if strings.TrimSpace(req.Name) == "" {
		fields["name"] = "required"
	}
	expiresAt, expiryError := req.expiresAt()
	if expiryError != "" {
		fields["expires_in_days"] = expiryError
	}
	if len(fields) > 0 {
		api.WriteValidation(w, r, fields)
		return
	}
	plaintext, token, err := h.Service.IssueAPIToken(r.Context(), user.ID, req.Name, APITokenPersonal, nil, expiresAt)
	if err != nil {
		writeTokenError(w, r, err)
		return
	}
	h.recordAuthEvent(r, "auth.token.issued", token.ID)
	api.WriteData(w, http.StatusCreated, map[string]any{"token": plaintext, "item": token}, nil)
}

func (h HTTPHandler) revokePersonalToken(w http.ResponseWriter, r *http.Request) {
	noStoreAuthResponse(w)
	user := UserFromContext(r.Context())
	if err := h.Service.RevokeAPIToken(r.Context(), r.PathValue("id"), user.ID); err != nil {
		writeTokenError(w, r, err)
		return
	}
	h.recordAuthEvent(r, "auth.token.revoked", r.PathValue("id"))
	w.WriteHeader(http.StatusNoContent)
}

func (h HTTPHandler) listAPIKeys(w http.ResponseWriter, r *http.Request) {
	noStoreAuthResponse(w)
	if !h.adminUserManage(r) {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	keys, err := h.Service.ListAPIKeys(r.Context())
	if err != nil {
		writeTokenError(w, r, err)
		return
	}
	api.WriteData(w, http.StatusOK, keys, nil)
}

func (h HTTPHandler) issueAPIKey(w http.ResponseWriter, r *http.Request) {
	noStoreAuthResponse(w)
	if !h.adminUserManage(r) {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	user := UserFromContext(r.Context())
	var req issueTokenRequest
	if !api.DecodeJSON(w, r, &req, maxAuthFormBytes) {
		return
	}
	fields := map[string]string{}
	if strings.TrimSpace(req.Name) == "" {
		fields["name"] = "required"
	}
	if req.Scope != ScopeRead && req.Scope != ScopePreview {
		fields["scope"] = "invalid"
	}
	expiresAt, expiryError := req.expiresAt()
	if expiryError != "" {
		fields["expires_in_days"] = expiryError
	}
	if len(fields) > 0 {
		api.WriteValidation(w, r, fields)
		return
	}
	plaintext, token, err := h.Service.IssueAPIToken(r.Context(), user.ID, req.Name, APITokenKey, []string{req.Scope}, expiresAt)
	if err != nil {
		writeTokenError(w, r, err)
		return
	}
	h.recordAuthEvent(r, "auth.api_key.issued", token.ID)
	api.WriteData(w, http.StatusCreated, map[string]any{"token": plaintext, "item": token}, nil)
}

func (h HTTPHandler) revokeAPIKey(w http.ResponseWriter, r *http.Request) {
	noStoreAuthResponse(w)
	if !h.adminUserManage(r) {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	if err := h.Service.RevokeAPIToken(r.Context(), r.PathValue("id"), ""); err != nil {
		writeTokenError(w, r, err)
		return
	}
	h.recordAuthEvent(r, "auth.api_key.revoked", r.PathValue("id"))
	w.WriteHeader(http.StatusNoContent)
}

func writeTokenError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrAPITokenNotFound):
		api.WriteError(w, r, http.StatusNotFound, "token not found", nil)
	case errors.Is(err, ErrAPITokenInvalid), errors.Is(err, ErrAPITokenScope):
		api.WriteError(w, r, http.StatusUnprocessableEntity, err.Error(), nil)
	case errors.Is(err, ErrAPITokenStoreRequired):
		api.WriteError(w, r, http.StatusServiceUnavailable, "api tokens are unavailable", nil)
	default:
		api.WriteError(w, r, http.StatusInternalServerError, "request could not be completed", nil)
	}
}
