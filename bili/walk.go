package bili

import (
	"context"
	"fmt"
	"iter"
	"strings"
)

// NodeKind is the kind of object a walk node represents. The graph has two
// kinds: videos and the creators who upload them.
type NodeKind string

const (
	NodeVideo NodeKind = "video"
	NodeUser  NodeKind = "user"
)

// Edge is one relationship the walker can follow. Each edge has a fixed source
// and target node kind (see source/Target).
type Edge string

const (
	// EdgeRelated is the recommendation edge: a video to the videos bilibili
	// suggests alongside it. video -> video.
	EdgeRelated Edge = "related"
	// EdgeUploader is the authorship edge: a video to the creator who posted
	// it. video -> user.
	EdgeUploader Edge = "uploader"
	// EdgeUploads is the catalogue edge: a creator to the videos they have
	// uploaded. user -> video.
	EdgeUploads Edge = "uploads"
)

var allEdges = []Edge{EdgeRelated, EdgeUploader, EdgeUploads}

var knownEdges = map[Edge]bool{
	EdgeRelated:  true,
	EdgeUploader: true,
	EdgeUploads:  true,
}

// source reports the node kind an edge departs from.
func (e Edge) source() NodeKind {
	if e == EdgeUploads {
		return NodeUser
	}
	return NodeVideo
}

// Target reports the node kind an edge arrives at. The walker uses it to tag a
// neighbor with the right kind as it is enqueued, so a node's kind is decided by
// the edge that found it.
func (e Edge) Target() NodeKind {
	if e == EdgeUploader {
		return NodeUser
	}
	return NodeVideo
}

// EdgeSet is a set of edges to follow.
type EdgeSet map[Edge]bool

func newEdgeSet(es ...Edge) EdgeSet {
	s := make(EdgeSet, len(es))
	for _, e := range es {
		s[e] = true
	}
	return s
}

// Has reports whether the set contains an edge.
func (s EdgeSet) Has(e Edge) bool { return s[e] }

// List returns the edges in the set in canonical (catalogue) order.
func (s EdgeSet) List() []Edge {
	var out []Edge
	for _, e := range allEdges {
		if s[e] {
			out = append(out, e)
		}
	}
	return out
}

// String renders the set as a comma-separated list in canonical order.
func (s EdgeSet) String() string { return joinEdges(s.List()) }

func (s EdgeSet) clone() EdgeSet {
	out := make(EdgeSet, len(s))
	for e := range s {
		out[e] = true
	}
	return out
}

// edgePresets are named slices of the edge set. Preset names and edge names are
// deliberately disjoint so a preset can never shadow a same-named edge.
var edgePresets = map[string]EdgeSet{
	"content":  newEdgeSet(EdgeRelated, EdgeUploads),
	"creators": newEdgeSet(EdgeUploader, EdgeUploads),
	"all":      newEdgeSet(allEdges...),
}

var presetNames = []string{"content", "creators", "all"}

// DefaultEdges is the edge set used when --follow is empty: the content preset.
func DefaultEdges() EdgeSet { return edgePresets["content"].clone() }

// EdgeHelp is a one-line catalogue of presets and edges for flag help and error
// messages.
func EdgeHelp() string {
	return "presets " + strings.Join(presetNames, "|") + "; edges " + strings.Join(edgeNames(), "|")
}

func edgeNames() []string {
	out := make([]string, len(allEdges))
	for i, e := range allEdges {
		out[i] = string(e)
	}
	return out
}

// ParseEdges turns a --follow spec into an edge set. The spec is empty (the
// default), a preset name, or a comma-separated list mixing presets and edge
// names. Each token resolves as a preset first and an edge name second; an
// unrecognized token is an error that names the catalogue.
func ParseEdges(spec string) (EdgeSet, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return DefaultEdges(), nil
	}
	set := newEdgeSet()
	for _, tok := range strings.Split(spec, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		if p, ok := edgePresets[tok]; ok {
			for e := range p {
				set[e] = true
			}
			continue
		}
		if e := Edge(tok); knownEdges[e] {
			set[e] = true
			continue
		}
		return nil, fmt.Errorf("unknown edge or preset %q; %s", tok, EdgeHelp())
	}
	if len(set) == 0 {
		return DefaultEdges(), nil
	}
	return set, nil
}

func joinEdges(es []Edge) string {
	parts := make([]string, len(es))
	for i, e := range es {
		parts[i] = string(e)
	}
	return strings.Join(parts, ",")
}

// Node is one object reached by a walk, tagged with how it was reached. Exactly
// one of Video or User is set, per Kind.
type Node struct {
	Kind   NodeKind `json:"kind"`
	Depth  int      `json:"depth"`
	Via    Edge     `json:"via,omitempty"`
	Parent string   `json:"parent,omitempty"`
	Video  *Video   `json:"video,omitempty"`
	User   *User    `json:"user,omitempty"`
}

// Endpoint returns the node's identity: a BVID for a video, a mid for a user.
func (n *Node) Endpoint() string {
	switch n.Kind {
	case NodeUser:
		if n.User != nil {
			return itoa(n.User.Mid)
		}
	default:
		if n.Video != nil {
			return n.Video.BVID
		}
	}
	return ""
}

// nodeKey is the dedup key for a node: kind plus identity.
func nodeKey(kind NodeKind, ref string) string {
	if kind == NodeUser {
		return "u:" + ref
	}
	return "v:" + ref
}

// Seed is a starting point for a walk: any id or URL the client can resolve to
// a video or a user.
type Seed struct {
	Raw string
}

// WalkOptions bounds a walk. Depth is the number of hops from each seed; Max
// caps the total nodes streamed; Fanout caps neighbors followed per edge. Note,
// when set, receives a one-line message for each non-fatal failure deeper in the
// walk.
type WalkOptions struct {
	Depth  int
	Max    int
	Fanout int
	Edges  EdgeSet
	Note   func(string)
}

// grapher is the subset of *Client the walker needs. It exists so the BFS logic
// is tested over an in-memory fake with no network.
type grapher interface {
	Resolve(ctx context.Context, s string) (*Identity, error)
	Video(ctx context.Context, idOrURL string, opt VideoOptions) (*VideoResult, error)
	User(ctx context.Context, mid string) (*User, error)
	Related(ctx context.Context, idOrURL string) ([]Video, error)
	UserVideos(ctx context.Context, mid string, opt ListOptions) iter.Seq2[Video, error]
}

var _ grapher = (*Client)(nil)

// Walker performs a breadth-first walk of the graph over a grapher.
type Walker struct {
	g grapher
}

// NewWalker returns a walker backed by g.
func NewWalker(g grapher) *Walker { return &Walker{g: g} }

// Walk runs a breadth-first walk from a client, streaming each node to emit.
func (c *Client) Walk(ctx context.Context, seeds []Seed, opts WalkOptions, emit func(*Node) error) error {
	return NewWalker(c).Walk(ctx, seeds, opts, emit)
}

// frontier is a queued walk item. A seed carries only seedRaw and is resolved
// when popped; a neighbor arrives pre-hydrated with its kind, ref, and payload.
type frontier struct {
	seedRaw string
	kind    NodeKind
	ref     string
	depth   int
	via     Edge
	parent  string
	video   *Video
	user    *User
}

// Walk runs a breadth-first walk from the seeds, streaming each node to emit as
// it is reached. A seed that cannot be fetched is fatal; a failure deeper in the
// walk is reported through opts.Note and the walk continues.
func (w *Walker) Walk(ctx context.Context, seeds []Seed, opts WalkOptions, emit func(*Node) error) error {
	if opts.Edges == nil {
		opts.Edges = DefaultEdges()
	}
	visited := make(map[string]bool)
	queue := make([]frontier, 0, len(seeds))
	for _, s := range seeds {
		queue = append(queue, frontier{seedRaw: s.Raw, depth: 0})
	}
	emitted := 0
	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		f := queue[0]
		queue = queue[1:]

		node, key, err := w.hydrate(ctx, f)
		if err != nil {
			if f.depth == 0 {
				// A seed that cannot be fetched fails the walk, like a single read.
				return err
			}
			note(opts, err)
			continue
		}
		if visited[key] {
			continue
		}
		visited[key] = true

		if err := emit(node); err != nil {
			return err
		}
		emitted++
		if opts.Max > 0 && emitted >= opts.Max {
			return nil
		}
		if f.depth >= opts.Depth {
			continue
		}
		queue = append(queue, w.neighbors(ctx, node, f.depth, opts)...)
	}
	return nil
}

// hydrate turns a frontier item into a node and its dedup key. A pre-hydrated
// neighbor is wrapped as-is; a seed is resolved and fetched by identity.
func (w *Walker) hydrate(ctx context.Context, f frontier) (*Node, string, error) {
	if f.seedRaw == "" {
		switch f.kind {
		case NodeUser:
			return &Node{Kind: NodeUser, Depth: f.depth, Via: f.via, Parent: f.parent, User: f.user}, nodeKey(NodeUser, f.ref), nil
		default:
			return &Node{Kind: NodeVideo, Depth: f.depth, Via: f.via, Parent: f.parent, Video: f.video}, nodeKey(NodeVideo, f.ref), nil
		}
	}
	id, err := w.g.Resolve(ctx, f.seedRaw)
	if err != nil {
		return nil, "", err
	}
	switch id.Kind {
	case KindVideo:
		res, err := w.g.Video(ctx, id.BVID, VideoOptions{})
		if err != nil {
			return nil, "", err
		}
		v := res.Video
		return &Node{Kind: NodeVideo, Depth: 0, Video: &v}, nodeKey(NodeVideo, v.BVID), nil
	case KindUser:
		u, err := w.g.User(ctx, itoa(id.Mid))
		if err != nil {
			return nil, "", err
		}
		return &Node{Kind: NodeUser, Depth: 0, User: u}, nodeKey(NodeUser, itoa(id.Mid)), nil
	default:
		return nil, "", fmt.Errorf("discover walks videos and users; %q resolves to a %s", f.seedRaw, id.Kind)
	}
}

// neighbors expands one node into its frontier items, following only the edges
// whose source matches the node's kind. Each neighbor is pre-built from the list
// it came in on, so it needs no refetch when popped.
func (w *Walker) neighbors(ctx context.Context, n *Node, depth int, opts WalkOptions) []frontier {
	limit := opts.Fanout
	if limit <= 0 {
		limit = opts.Max
	}
	var out []frontier
	switch n.Kind {
	case NodeVideo:
		v := n.Video
		if opts.Edges.Has(EdgeRelated) {
			rel, err := w.g.Related(ctx, v.BVID)
			for _, r := range capVideos(rel, limit) {
				rr := r
				out = append(out, frontier{kind: NodeVideo, ref: rr.BVID, depth: depth + 1, via: EdgeRelated, parent: v.BVID, video: &rr})
			}
			note(opts, err)
		}
		if opts.Edges.Has(EdgeUploader) && v.OwnerMid != 0 {
			u := &User{Mid: v.OwnerMid, Name: v.OwnerName}
			out = append(out, frontier{kind: NodeUser, ref: itoa(v.OwnerMid), depth: depth + 1, via: EdgeUploader, parent: v.BVID, user: u})
		}
	case NodeUser:
		u := n.User
		if opts.Edges.Has(EdgeUploads) {
			vids, err := collectVideos(w.g.UserVideos(ctx, itoa(u.Mid), ListOptions{Limit: limit}), limit)
			for _, vv := range vids {
				v2 := vv
				out = append(out, frontier{kind: NodeVideo, ref: v2.BVID, depth: depth + 1, via: EdgeUploads, parent: itoa(u.Mid), video: &v2})
			}
			note(opts, err)
		}
	}
	return out
}

// collectVideos drains a video iterator up to limit, returning whatever it
// gathered plus the first error. A mid-stream error keeps the videos already
// read so the walk can still follow them.
func collectVideos(seq iter.Seq2[Video, error], limit int) ([]Video, error) {
	var out []Video
	for v, err := range seq {
		if err != nil {
			return out, err
		}
		out = append(out, v)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func capVideos(vs []Video, limit int) []Video {
	if limit > 0 && len(vs) > limit {
		return vs[:limit]
	}
	return vs
}

func note(opts WalkOptions, err error) {
	if err == nil || opts.Note == nil {
		return
	}
	opts.Note(err.Error())
}
