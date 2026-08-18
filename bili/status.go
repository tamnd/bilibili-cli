package bili

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
)

// Response classification.
//
// This API says code 0 when it means no. Two endpoints were measured returning
// the success code with the success message and carrying nothing at all, signed
// or not, and a third refuses with an HTML page and no envelope in sight. A
// client that reads the code and starts decoding cannot tell any of that from a
// user whose favorites folder is genuinely empty.
//
// So nothing is decoded until the response has been sorted into exactly one
// state. The sorting is a pure function of the HTTP status, the content type,
// the envelope code and one static fact per endpoint, which is whether that
// endpoint is known to carry a payload when it answers. That last fact is what
// makes refused_silent recognisable, and it is not in the response: it is in
// the request. An endpoint known to carry a payload, answering with none, has
// refused.

// Status is the state one response was sorted into.
type Status string

const (
	// StatusOK is a payload.
	StatusOK Status = "ok"

	// StatusEmpty is a payload that is legitimately empty, which is a real
	// answer and not a refusal. An empty favorites folder is an empty folder.
	StatusEmpty Status = "empty"

	// StatusRefusedSilent is code 0 carrying nothing, on an endpoint that is
	// known to carry something. This is the state that does not exist in any
	// generic HTTP client, and it is the reason this file exists.
	StatusRefusedSilent Status = "refused_silent"

	// StatusRisk is -352, or an HTTP 412 with an HTML body.
	StatusRisk Status = "risk"

	// StatusForbidden is -403.
	StatusForbidden Status = "forbidden"

	// StatusNotFound is -404, or one of the private not found constants.
	StatusNotFound Status = "not_found"

	// StatusRate is -509, or an HTTP 429.
	StatusRate Status = "rate"

	// StatusError is not one of the seven. It is the absence of a
	// classification: a code this client has no state for, or a body that would
	// not parse. It is kept distinct so that "we do not recognise this" never
	// gets quietly rounded down to one of the states that means something.
	StatusError Status = "error"
)

// Refused reports whether the response denied the request rather than answering
// it. StatusEmpty is deliberately not a refusal: nothing came back because
// there was nothing to send.
func (s Status) Refused() bool {
	switch s {
	case StatusOK, StatusEmpty:
		return false
	}
	return true
}

// label is how a state reads in front of a person. The constants are machine
// names and belong in output that is parsed; an error message is not that.
func (s Status) label() string {
	switch s {
	case StatusRefusedSilent:
		return "refused"
	case StatusRisk:
		return "risk control"
	case StatusRate:
		return "rate limited"
	case StatusForbidden:
		return "forbidden"
	case StatusNotFound:
		return "not found"
	case StatusError:
		return "unrecognised response"
	}
	return string(s)
}

// Cacheable reports whether a response in this state is worth keeping.
//
// A refusal never is. Every refusal in this API is a statement about this
// request at this moment, from this address, and writing one to disk turns a
// transient into a permanent for the length of the TTL. The -352 case is the
// one that bites: it clears on its own in minutes, and a cached copy of it
// makes the tool look broken for an hour.
func (s Status) Cacheable() bool { return !s.Refused() }

// statusForCode maps an envelope code to a state, and reports whether the code
// was one this client recognises.
func statusForCode(code int) (Status, bool) {
	switch code {
	case -352:
		return StatusRisk, true
	case -403:
		return StatusForbidden, true
	case -404, 62002, 62004, 4511001:
		return StatusNotFound, true
	case -509:
		return StatusRate, true
	}
	return StatusError, false
}

// statusForHTTP maps a non 2xx HTTP status to a state.
//
// The 412 is the interesting one. Risk control serves it with an HTML
// interstitial and no envelope, so a JSON decoder meeting it reports a parse
// error, which is a true statement about the bytes and a useless statement
// about what happened.
func statusForHTTP(code int) (Status, bool) {
	switch code {
	case http.StatusPreconditionFailed:
		return StatusRisk, true
	case http.StatusTooManyRequests:
		return StatusRate, true
	case http.StatusForbidden:
		return StatusForbidden, true
	case http.StatusNotFound:
		return StatusNotFound, true
	}
	return StatusError, false
}

// payloadEndpoints is the set of endpoints known to carry a payload when they
// answer, derived from the requirement matrix so that there is one table rather
// than two that can disagree.
//
// An endpoint absent from the matrix has no rule, and an endpoint with no rule
// is never called refused_silent. Being wrong in that direction costs a caller
// an empty result they have to interpret themselves, which is where they
// already were. Being wrong in the other direction would invent a refusal that
// never happened.
var payloadEndpoints = sync.OnceValue(func() map[string]bool {
	m := make(map[string]bool, len(Matrix()))
	for _, r := range Matrix() {
		m[r.Base] = r.Payload
	}
	return m
})

// carriesPayload reports whether base always carries a payload when it answers.
func carriesPayload(base string) bool { return payloadEndpoints()[base] }

// isEmptyPayload reports whether a payload carries nothing.
//
// An empty array is not in this set on purpose. A server that went to the
// trouble of emitting [] is answering with a list that happens to have no
// entries, which is exactly the legitimate empty result. Every silent refusal
// measured came back as null or as {}.
func isEmptyPayload(p json.RawMessage) bool {
	s := strings.TrimSpace(string(p))
	return s == "" || s == "null" || s == "{}"
}

// isEmptyList reports whether a payload is an answer with nothing in it.
func isEmptyList(p json.RawMessage) bool {
	return strings.TrimSpace(string(p)) == "[]"
}

// isHTML reports whether a response is a web page rather than an API answer.
// The content type is checked rather than sniffed at the body, because risk
// control's interstitial is a complete valid HTML document and there is no
// reason to guess when the server has already said what it sent.
func isHTML(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "text/html")
}

// classify sorts one JSON response into a state and returns the envelope it
// parsed, so that the caller does not parse it a second time.
//
// The order matters and is the order in the specification. The HTTP status is
// read first, because a 412 has no envelope to read. The content type is read
// second, because a JSON surface answering text/html has been intercepted
// whatever the status line said. The envelope is decoded third. The code is
// mapped fourth. The payload rule runs last, and it is the step a client
// written against the documented envelope would never think to include.
func classify(res result) (Status, envelope, *APIError) {
	var env envelope

	if res.status < 200 || res.status >= 300 {
		st, known := statusForHTTP(res.status)
		return st, env, httpError(st, known, res)
	}

	if isHTML(res.contentType) {
		return StatusRisk, env, riskError(res)
	}

	if err := json.Unmarshal(res.body, &env); err != nil {
		return StatusError, env, &APIError{
			Message:  snippet(res.body),
			Hint:     "the response was not JSON and was not an interception either, so this client does not know what it is",
			Kind:     ErrGeneric,
			Status:   StatusError,
			Endpoint: res.base,
		}
	}

	if env.Code != 0 {
		e := apiError(env.Code, env.Message)
		e.Endpoint = res.base
		if st, known := statusForCode(env.Code); known {
			e.Status = st
		} else {
			e.Status = StatusError
		}
		return e.Status, env, e
	}

	payload := env.payload()
	switch {
	case isEmptyList(payload):
		return StatusEmpty, env, nil
	case !isEmptyPayload(payload):
		return StatusOK, env, nil
	case carriesPayload(res.base):
		return StatusRefusedSilent, env, refusedSilentError(res.base)
	default:
		return StatusEmpty, env, nil
	}
}

// classifyBinary sorts a response from a surface that is not JSON, which is the
// protobuf danmaku segment stream. There is no envelope to read, so the only
// two questions available are whether it was intercepted and whether any bytes
// came back.
func classifyBinary(res result) (Status, *APIError) {
	if res.status < 200 || res.status >= 300 {
		st, known := statusForHTTP(res.status)
		return st, httpError(st, known, res)
	}
	if isHTML(res.contentType) {
		return StatusRisk, riskError(res)
	}
	if len(res.body) == 0 {
		return StatusRefusedSilent, refusedSilentError(res.base)
	}
	return StatusOK, nil
}

func httpError(st Status, known bool, res result) *APIError {
	if st == StatusRisk {
		return riskError(res)
	}
	e := &APIError{
		Message:  http.StatusText(res.status),
		Status:   st,
		Endpoint: res.base,
	}
	switch {
	case !known:
		e.Hint, e.Kind = "the server answered with a status this client has no state for", ErrNetwork
	case st == StatusRate:
		e.Hint, e.Kind = "rate limited, slow down with --rate and retry", ErrRate
	case st == StatusForbidden:
		e.Hint, e.Kind = "access denied", ErrAccess
	case st == StatusNotFound:
		e.Hint, e.Kind = "not found or content removed", ErrNotFound
	}
	return e
}

func riskError(res result) *APIError {
	return &APIError{
		Message:  "intercepted by risk control",
		Hint:     riskHint,
		Kind:     ErrAccess,
		Status:   StatusRisk,
		Endpoint: res.base,
	}
}

func refusedSilentError(base string) *APIError {
	return &APIError{
		Message:  "code 0 with no payload",
		Hint:     "this endpoint always carries a payload when it answers, so this is a refusal and not an empty result. It refuses anonymous callers, and only a logged-in cookie via --cookie or BILI_COOKIE changes that",
		Kind:     ErrAccess,
		Status:   StatusRefusedSilent,
		Endpoint: base,
	}
}

// snippet is the first 200 bytes of a body, for the case where classification
// itself failed. Attaching it is the difference between a bug report somebody
// can act on and one that says the response was unparseable.
func snippet(b []byte) string {
	const max = 200
	s := strings.TrimSpace(string(b))
	if len(s) > max {
		return s[:max] + "..."
	}
	if s == "" {
		return "(empty body)"
	}
	return s
}
