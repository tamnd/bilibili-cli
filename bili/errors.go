package bili

import "fmt"

// APIError is a non-zero bilibili envelope code mapped to a clean message.
type APIError struct {
	Code    int
	Message string // upstream message (often Chinese)
	Hint    string // English hint
	Kind    ErrKind
}

// ErrKind groups API errors so the CLI can map them to exit codes.
type ErrKind int

const (
	ErrGeneric ErrKind = iota
	ErrNotFound
	ErrAccess
	ErrRate
	ErrNetwork
)

func (e *APIError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("bilibili %d: %s (%s)", e.Code, e.Hint, e.Message)
	}
	return fmt.Sprintf("bilibili %d: %s", e.Code, e.Message)
}

// apiError maps a code/message into a typed error.
func apiError(code int, message string) *APIError {
	e := &APIError{Code: code, Message: message, Kind: ErrGeneric}
	switch code {
	case -101:
		e.Hint, e.Kind = "not logged in: this endpoint needs cookies, pass --cookie or BILI_COOKIE", ErrAccess
	case -400:
		e.Hint = "bad request"
	case -403:
		e.Hint, e.Kind = "access denied", ErrAccess
	case -404, 62002, 62004:
		e.Hint, e.Kind = "not found or content removed/invisible", ErrNotFound
	case -352:
		e.Hint, e.Kind = "risk control: this endpoint is gated by bilibili's anti-bot for anonymous access, supply a logged-in cookie via --cookie or BILI_COOKIE and retry", ErrAccess
	case -412:
		e.Hint, e.Kind = "request intercepted: rate-limited or missing WBI/UA", ErrRate
	case -509:
		e.Hint, e.Kind = "rate limit exceeded", ErrRate
	case 22001, 22002, 22003, 22004, 22005, 22006, 22007:
		e.Hint, e.Kind = "comment area unavailable", ErrAccess
	}
	return e
}

// Kind reports the ErrKind of an error if it is an APIError.
func Kind(err error) ErrKind {
	if ae, ok := err.(*APIError); ok {
		return ae.Kind
	}
	return ErrGeneric
}
