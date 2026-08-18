package cli

import (
	"context"
	"errors"
	"net"

	"github.com/tamnd/bilibili-cli/bili"
)

// Exit codes.
//
// The point of having more than two is that a script can act on the difference
// without parsing stderr. Risk is separate from network because a risk refusal
// does not clear by retrying and a network failure does. Rate is separate from
// both because it clears by waiting, and it is the only one that does. And a
// silent refusal is separate from an empty result because "the creator has no
// public folders" and "the API refused to tell us whether the creator has
// public folders" are different answers.
const (
	// ExitOK means it did what it was asked.
	ExitOK = 0

	// ExitUsage means this tool could not do what was asked and cannot say more
	// than that: a flag or argument it does not understand, a response it could
	// not classify, or a run that was interrupted.
	ExitUsage = 1

	// ExitRisk means risk control refused the request, as a -352 or an HTTP 412.
	// Retrying will not clear it. A logged-in cookie usually will.
	ExitRisk = 2

	// ExitEmpty means the request succeeded and there was genuinely nothing to
	// return.
	ExitEmpty = 3

	// ExitRefused means the API returned a success code carrying no payload, on
	// an endpoint that always carries one.
	ExitRefused = 4

	// ExitNetwork means a transport failure, a timeout, or a 5xx that outlived
	// the retries.
	ExitNetwork = 5

	// ExitRate means rate limited: a -509 or a 429 that outlived the retries.
	ExitRate = 6

	// ExitNotFound means not found, either as a -404 or as one of the private
	// not found constants an endpoint uses instead.
	ExitNotFound = 7
)

// ErrNoResults is the run that worked and produced no records. It is an error
// only in the sense that it deserves its own exit code; nothing went wrong.
var ErrNoResults = errors.New("nothing to return")

// ExitCode maps an error to the process exit code.
//
// The mapping goes through bili.Kind, so it reads the state the response was
// classified into rather than matching on message text. An error this tool did
// not produce, or one it could not name, is ExitUsage, which is the one code
// that promises nothing about what happened.
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	if errors.Is(err, ErrNoResults) {
		return ExitEmpty
	}
	switch bili.Kind(err) {
	case bili.ErrRisk, bili.ErrForbidden:
		return ExitRisk
	case bili.ErrEmpty:
		return ExitEmpty
	case bili.ErrRefusedSilent:
		return ExitRefused
	case bili.ErrNetwork:
		return ExitNetwork
	case bili.ErrRate:
		return ExitRate
	case bili.ErrNotFound:
		return ExitNotFound
	}
	// Not an APIError. A timeout or a dead connection can still reach here from
	// the paths that talk to the network without going through the classifier,
	// and calling those a usage error would send a caller looking at their
	// flags for a problem that is in the wire.
	var ne net.Error
	if errors.Is(err, context.DeadlineExceeded) || errors.As(err, &ne) {
		return ExitNetwork
	}
	return ExitUsage
}
