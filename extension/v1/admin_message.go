package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	AdminMessageVersion = 1
	MaxAdminMessageSize = 64 << 10
)

var ErrInvalidAdminMessage = errors.New("invalid admin extension message")

// AdminMessage is the only data envelope exchanged between the host shell and
// an external extension surface. Authentication cookies, bearer tokens and
// arbitrary executable code are not part of this contract.
type AdminMessage struct {
	Version   int             `json:"version"`
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

var allowedAdminMessageTypes = map[string]struct{}{
	"init": {}, "ready": {}, "bound": {}, "revoke": {}, "resize": {}, "navigate": {}, "request": {}, "response": {}, "error": {},
}

func (m AdminMessage) Validate() error {
	if m.Version != AdminMessageVersion {
		return ErrInvalidAdminMessage
	}
	if _, ok := allowedAdminMessageTypes[m.Type]; !ok || len(m.RequestID) > 64 || (m.RequestID != "" && !validIdentifier(m.RequestID)) {
		return ErrInvalidAdminMessage
	}
	if len(m.Payload) > MaxAdminMessageSize || (len(m.Payload) > 0 && !json.Valid(m.Payload)) {
		return fmt.Errorf("%w: payload", ErrInvalidAdminMessage)
	}
	return nil
}

func ParseAdminMessage(data []byte) (AdminMessage, error) {
	if len(data) == 0 || len(data) > MaxAdminMessageSize {
		return AdminMessage{}, ErrInvalidAdminMessage
	}
	var message AdminMessage
	if err := json.Unmarshal(data, &message); err != nil {
		return AdminMessage{}, ErrInvalidAdminMessage
	}
	if err := message.Validate(); err != nil {
		return AdminMessage{}, err
	}
	return message, nil
}

// AdminBootstrapPayload is the non-secret context the host may send after the
// iframe handshake. It contains no session cookie or bearer token.
type AdminBootstrapPayload struct {
	ExtensionID string   `json:"extension_id"`
	SurfaceID   string   `json:"surface_id"`
	Locale      string   `json:"locale"`
	Permissions []string `json:"permissions,omitempty"`
}

// AdminChannelInitPayload is deliberately non-secret bootstrap data sent to
// the exact iframe origin. The challenge only binds that concrete iframe to a
// short-lived host handshake; it is never an authorization credential.
type AdminChannelInitPayload struct {
	ExtensionID string `json:"extension_id"`
	SurfaceID   string `json:"surface_id"`
	Challenge   string `json:"challenge"`
}

func (p AdminChannelInitPayload) Validate() error {
	if !validIdentifier(p.ExtensionID) || !validIdentifier(p.SurfaceID) || !validIdentifier(p.Challenge) || len(p.Challenge) < 16 {
		return ErrInvalidAdminMessage
	}
	return nil
}

func (p AdminBootstrapPayload) Validate() error {
	if !validIdentifier(p.ExtensionID) || !validIdentifier(p.SurfaceID) || len(p.Locale) > 32 || strings.ContainsAny(p.Locale, "\r\n") {
		return ErrInvalidAdminMessage
	}
	for _, permission := range p.Permissions {
		if !validPermission(permission) {
			return ErrInvalidAdminMessage
		}
	}
	return nil
}
