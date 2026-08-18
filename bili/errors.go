package bili

import (
	"fmt"
	"strings"
)

// APIError is a refusal, mapped to a clean message.
//
// It is no longer only a non-zero envelope code. A refusal on this API can also
// arrive as an HTML page with an HTTP 412 and no envelope at all, or as the
// success code carrying nothing, so Code is zero for a good number of these and
// Status is the field that says what happened.
type APIError struct {
	Code    int
	Message string // upstream message (often Chinese)
	Hint    string // English hint
	Kind    ErrKind

	// Status is the state the response was classified into. It is the field to
	// branch on. Code is upstream's vocabulary and has gaps; this one does not.
	Status Status

	// Endpoint is the base URL that refused, without query parameters. A user
	// told which endpoint said no can go and look at it.
	Endpoint string
}

// riskHint is the advice attached to every risk refusal, whether it arrived as
// a -352 or as an HTML page behind a 412.
const riskHint = "risk control: this endpoint is gated by bilibili's anti-bot for anonymous access, supply a logged-in cookie via --cookie or BILI_COOKIE and retry"

// ErrKind groups API errors so the CLI can map them to exit codes.
type ErrKind int

const (
	ErrGeneric ErrKind = iota
	ErrNotFound
	ErrAccess
	ErrRate
	ErrNetwork
)

// Error renders the refusal.
//
// A zero code is not printed, because a code of 0 in front of a user reads as
// success and these are all refusals. The endpoint is printed when it is known,
// since the first question anybody asks is which request this was.
func (e *APIError) Error() string {
	var b strings.Builder
	if e.Code != 0 {
		fmt.Fprintf(&b, "bilibili %d: ", e.Code)
	} else if e.Status != "" {
		fmt.Fprintf(&b, "%s: ", e.Status.label())
	}
	if e.Hint != "" {
		fmt.Fprintf(&b, "%s (%s)", e.Hint, e.Message)
	} else {
		b.WriteString(e.Message)
	}
	if e.Endpoint != "" {
		fmt.Fprintf(&b, " [%s]", strings.TrimPrefix(e.Endpoint, "https://"))
	}
	return b.String()
}

// apiError maps a code/message into a typed error.
func apiError(code int, message string) *APIError {
	e := &APIError{Code: code, Message: message, Kind: ErrGeneric}
	if st, known := statusForCode(code); known {
		e.Status = st
	} else {
		e.Status = StatusError
	}
	switch code {
	case -101:
		e.Hint, e.Kind = "not logged in: this endpoint needs cookies, pass --cookie or BILI_COOKIE", ErrAccess
	case -400:
		e.Hint = "bad request"
	case -403:
		e.Hint, e.Kind = "access denied", ErrAccess
	case -404, 62002, 62004:
		e.Hint, e.Kind = "not found or content removed/invisible", ErrNotFound
	case 4511001:
		e.Hint, e.Kind = "not found: this audio track does not exist or is no longer public", ErrNotFound
	case -352:
		e.Hint, e.Kind = riskHint, ErrAccess
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
