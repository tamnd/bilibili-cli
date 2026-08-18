package bili

import (
	"errors"
	"strings"
)

// Provenance.
//
// A record from this tool is not a document that was sitting somewhere waiting
// to be fetched. It is assembled, sometimes from three endpoints with different
// gates, and the assembly is invisible in the result: a user record with a
// follower count and no view count looks exactly like a user record from an
// endpoint that carries neither. The envelope is what makes the assembly
// legible after the fact.
//
// It answers four questions about the reading rather than about the thing
// read: which endpoint answered, whether the request was signed, what state the
// response was sorted into, and when. The fifth field is the one that pays for
// the other four, and it is Missed: the fields that are not in this record and
// the reason each of them is not.

// Envelope is the provenance of one record.
type Envelope struct {
	// Endpoint is the path that answered, without the host. It is the short
	// form the requirement matrix uses, so a reader can look the row up.
	Endpoint string `json:"endpoint"`

	// Signed records whether the request carried a WBI signature. Whether an
	// endpoint needs one is measured rather than derivable, and it moves, so
	// the answer for this request is worth keeping with this record.
	Signed bool `json:"signed"`

	// Status is the state the response was sorted into. On an emitted record it
	// is almost always ok or empty, since a refusal produces an error instead;
	// it is here so that a record and a refusal can be read the same way.
	Status Status `json:"status"`

	// Fetched is when the reading happened. It duplicates FetchedAt on the
	// records that have one, deliberately: FetchedAt is a field of the thing and
	// this is a field of the reading, and the two come apart the moment a record
	// is assembled from more than one request.
	Fetched string `json:"fetched"`

	// Bytes is the size of the response body this record came out of.
	Bytes int `json:"bytes,omitempty"`

	// Missed names the fields that are absent and says why for each. An absent
	// field with no entry here was absent because there was nothing to put in
	// it. An absent field with an entry was stopped by something, and the entry
	// is what stopped it.
	Missed map[string]string `json:"missed,omitempty"`
}

// miss records that a field is absent and why. It is safe on a nil envelope,
// because a record built without one is still a record and a caller should not
// have to check.
func (e *Envelope) miss(field, why string) {
	if e == nil {
		return
	}
	if e.Missed == nil {
		e.Missed = make(map[string]string, 1)
	}
	e.Missed[field] = why
}

// clone returns a copy, for the case where one request produces many records
// and one of them has something to say that the others do not.
func (e *Envelope) clone() *Envelope {
	if e == nil {
		return nil
	}
	c := *e
	c.Missed = nil
	for k, v := range e.Missed {
		c.miss(k, v)
	}
	return &c
}

// endpointName is the matrix's short form of a URL: the path with the scheme,
// the host and the leading slash removed.
func endpointName(base string) string {
	s := strings.TrimPrefix(strings.TrimPrefix(base, "https://"), "http://")
	if _, path, ok := strings.Cut(s, "/"); ok {
		return path
	}
	return s
}

// refusalNote is how a refusal reads inside Missed. The message a person gets
// on stderr is a paragraph with advice in it; this is one clause, because it
// appears once per field and is read alongside the value that is not there.
func refusalNote(err error) string {
	var ae *APIError
	if !errors.As(err, &ae) {
		return err.Error()
	}
	name := endpointName(ae.Endpoint)
	if name == "" {
		return string(ae.Status) + ": " + ae.Message
	}
	return name + " " + string(ae.Status) + ": " + ae.Message
}
