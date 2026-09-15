package adminstudio

import (
	"errors"
	"net/url"
	"strings"
)

type PublicationState string

const (
	StateDraft     PublicationState = "draft"
	StateScheduled PublicationState = "scheduled"
	StatePublished PublicationState = "published"
	StateRetracted PublicationState = "retracted"
)

var ErrInvalidTransition = errors.New("invalid publication transition")

// CanTransition centralizes editorial state rules without knowing a module's
// persistence or provider. Scheduling itself remains an application concern.
func CanTransition(from, to PublicationState) bool {
	if from == to {
		return true
	}
	switch from {
	case StateDraft, StateRetracted:
		return to == StateScheduled || to == StatePublished
	case StateScheduled:
		return to == StatePublished || to == StateDraft || to == StateRetracted
	case StatePublished:
		return to == StateRetracted || to == StateDraft
	default:
		return false
	}
}

func ValidateTransition(from, to PublicationState) error {
	if !CanTransition(from, to) {
		return ErrInvalidTransition
	}
	return nil
}

// PreviewURL appends a short-lived preview marker to a same-origin path. The
// caller is responsible for signing/expiring the marker at its auth boundary.
func PreviewURL(path, token string) (string, error) {
	if path == "" || !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || token == "" {
		return "", errors.New("invalid preview target")
	}
	u, err := url.Parse(path)
	if err != nil || u.Host != "" {
		return "", errors.New("invalid preview target")
	}
	q := u.Query()
	q.Set("preview", token)
	u.RawQuery = q.Encode()
	return u.String(), nil
}
