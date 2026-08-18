package bili

import (
	"errors"
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

	// Status is the state the response was classified into. It is the field to
	// branch on, and the field the exit code is derived from. Code is upstream's
	// vocabulary and has gaps; this one does not.
	Status Status

	// Endpoint is the base URL that refused, without query parameters. A user
	// told which endpoint said no can go and look at it.
	Endpoint string
}

// riskHint is the advice attached to every risk refusal, whether it arrived as
// a -352 or as an HTML page behind a 412.
const riskHint = "risk control: this endpoint is gated by bilibili's anti-bot for anonymous access, supply a logged-in cookie via --cookie or BILI_COOKIE and retry"

// ErrKind is the kind of a refusal, which is the state its response was sorted
// into. The two used to be separate enums that had to be kept in agreement by
// hand, and they drifted: an error could be ErrAccess while its response was
// classified not_found. The kind of an error and the state of the response are
// one fact, so they are one type, and the names below are the vocabulary for
// callers who think of these as error kinds rather than as response states.
type ErrKind = Status

const (
	ErrRisk          = StatusRisk
	ErrForbidden     = StatusForbidden
	ErrEmpty         = StatusEmpty
	ErrRefusedSilent = StatusRefusedSilent
	ErrNetwork       = StatusNetwork
	ErrRate          = StatusRate
	ErrNotFound      = StatusNotFound

	// ErrGeneric is a refusal this client has no state for. It is not a kind so
	// much as the absence of one, kept distinct so that an unrecognised response
	// is never quietly rounded down to a kind that means something.
	ErrGeneric = StatusError
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

// apiError maps a code and message into a typed error.
//
// The state comes from statusForCode, which is the one table, and this function
// only adds the English hint. A code that statusForCode does not know is an
// unrecognised response and says so, rather than being given a plausible state.
func apiError(code int, message string) *APIError {
	e := &APIError{Code: code, Message: message, Status: StatusError}
	if st, known := statusForCode(code); known {
		e.Status = st
	}
	switch code {
	case -101:
		e.Hint = "not logged in: this endpoint needs cookies, pass --cookie or BILI_COOKIE"
	case -400:
		e.Hint = "bad request: an argument was not one this endpoint accepts"
	case -403:
		e.Hint = "access denied"
	case -404, 62002, 62004:
		e.Hint = "not found or content removed/invisible"
	case 4511001:
		e.Hint = "audio not found or removed: this track does not exist or is no longer public"
	case -352:
		e.Hint = riskHint
	case -412:
		e.Hint = "request intercepted by risk control: retrying will not clear it, supply a logged-in cookie via --cookie or BILI_COOKIE"
	case -509:
		e.Hint = "rate limit exceeded, slow down with --rate and retry"
	case 22001, 22002, 22003, 22004, 22005, 22006, 22007:
		e.Hint = "comment area unavailable"
	}
	return e
}

// Kind reports the kind of an error, which is StatusError for anything that did
// not come from this package. It unwraps, so a caller that wrapped the error
// with context still gets the answer.
func Kind(err error) ErrKind {
	var ae *APIError
	if errors.As(err, &ae) {
		return ae.Status
	}
	return ErrGeneric
}
