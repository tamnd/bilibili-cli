package bili

import (
	"context"
	"fmt"
	"iter"
	"testing"
)

// fakeGraph is an in-memory grapher: the walk's bounds, ordering, dedup, and
// note-on-failure path are exercised over it with no network.
type fakeGraph struct {
	resolved map[string]*Identity // raw seed -> identity
	videos   map[string]Video     // bvid -> full video (seed hydration)
	users    map[string]*User     // mid -> full user (seed hydration)
	related  map[string][]Video   // bvid -> related videos
	uploads  map[string][]Video   // mid -> uploaded videos
	fail     map[string]error     // "method:arg" -> error to return
}

func (f *fakeGraph) Resolve(_ context.Context, s string) (*Identity, error) {
	if err := f.fail["resolve:"+s]; err != nil {
		return nil, err
	}
	if id, ok := f.resolved[s]; ok {
		return id, nil
	}
	return &Identity{Kind: KindUnknown, Input: s}, fmt.Errorf("could not classify %q", s)
}

func (f *fakeGraph) Video(_ context.Context, idOrURL string, _ VideoOptions) (*VideoResult, error) {
	if err := f.fail["video:"+idOrURL]; err != nil {
		return nil, err
	}
	if v, ok := f.videos[idOrURL]; ok {
		return &VideoResult{Video: v}, nil
	}
	return nil, fmt.Errorf("video not found: %s", idOrURL)
}

func (f *fakeGraph) User(_ context.Context, mid string) (*User, error) {
	if err := f.fail["user:"+mid]; err != nil {
		return nil, err
	}
	if u, ok := f.users[mid]; ok {
		return u, nil
	}
	return nil, fmt.Errorf("user not found: %s", mid)
}

func (f *fakeGraph) Related(_ context.Context, idOrURL string) ([]Video, error) {
	if err := f.fail["related:"+idOrURL]; err != nil {
		return nil, err
	}
	return f.related[idOrURL], nil
}

func (f *fakeGraph) UserVideos(_ context.Context, mid string, _ ListOptions) iter.Seq2[Video, error] {
	return func(yield func(Video, error) bool) {
		for _, v := range f.uploads[mid] {
			if !yield(v, nil) {
				return
			}
		}
		// A trailing failure models an endpoint that returns some pages then
		// trips the anti-bot: the videos already read survive, plus a note.
		if err := f.fail["uploads:"+mid]; err != nil {
			yield(Video{}, err)
		}
	}
}

// newFakeGraph builds a tiny corpus:
//
//	BV_TURING (owner 2)  --related-->  BV_REL1 (owner 3), BV_REL2 (owner 2)
//	                     --uploader--> user 2
//	BV_REL1   --related-->  BV_TURING (cycle)
//	user 2    --uploads-->  BV_UP1, BV_UP2
//	user 3    --uploads-->  BV_REL1
func newFakeGraph() *fakeGraph {
	vid := func(bvid string, owner int64, ownerName, title string) Video {
		return Video{
			BVID: bvid, Title: title, OwnerMid: owner, OwnerName: ownerName,
			URL: "https://www.bilibili.com/video/" + bvid,
		}
	}
	return &fakeGraph{
		resolved: map[string]*Identity{
			"BV_TURING":                    {Kind: KindVideo, BVID: "BV_TURING"},
			"BV_MISSING":                   {Kind: KindVideo, BVID: "BV_MISSING"},
			"https://space.bilibili.com/2": {Kind: KindUser, Mid: 2},
			"ss123":                        {Kind: KindBangumi, SeasonID: 123},
		},
		videos: map[string]Video{
			"BV_TURING": vid("BV_TURING", 2, "Creator A", "On Computable Numbers"),
		},
		users: map[string]*User{
			"2": {Mid: 2, Name: "Creator A", FollowerCount: 1000},
		},
		related: map[string][]Video{
			"BV_TURING": {vid("BV_REL1", 3, "Creator B", "Related one"), vid("BV_REL2", 2, "Creator A", "Related two")},
			"BV_REL1":   {vid("BV_TURING", 2, "Creator A", "On Computable Numbers")},
		},
		uploads: map[string][]Video{
			"2": {vid("BV_UP1", 2, "Creator A", "Upload one"), vid("BV_UP2", 2, "Creator A", "Upload two")},
			"3": {vid("BV_REL1", 3, "Creator B", "Related one")},
		},
		fail: map[string]error{},
	}
}

func TestParseEdges(t *testing.T) {
	t.Run("empty is the content default", func(t *testing.T) {
		got, err := ParseEdges("")
		if err != nil {
			t.Fatal(err)
		}
		if got.String() != DefaultEdges().String() {
			t.Errorf("empty = %q, want default %q", got, DefaultEdges())
		}
		if !got.Has(EdgeRelated) || !got.Has(EdgeUploads) || got.Has(EdgeUploader) {
			t.Errorf("content = %q, want related+uploads", got)
		}
	})
	t.Run("preset expands", func(t *testing.T) {
		got, err := ParseEdges("creators")
		if err != nil {
			t.Fatal(err)
		}
		if !got.Has(EdgeUploader) || !got.Has(EdgeUploads) || got.Has(EdgeRelated) {
			t.Errorf("creators = %q, want uploader+uploads", got)
		}
	})
	t.Run("mixed edge and preset", func(t *testing.T) {
		// "related" is an edge and must not be shadowed by a preset of the same
		// name; the presets are deliberately disjoint from the edge names.
		got, err := ParseEdges("related,creators")
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range allEdges {
			if !got.Has(e) {
				t.Errorf("related,creators missing %s: %q", e, got)
			}
		}
	})
	t.Run("unknown token errors", func(t *testing.T) {
		if _, err := ParseEdges("nope"); err == nil {
			t.Error("want error for unknown token")
		}
	})
}

func TestEdgeSourceTarget(t *testing.T) {
	cases := []struct {
		e        Edge
		src, dst NodeKind
	}{
		{EdgeRelated, NodeVideo, NodeVideo},
		{EdgeUploader, NodeVideo, NodeUser},
		{EdgeUploads, NodeUser, NodeVideo},
	}
	for _, c := range cases {
		if got := c.e.source(); got != c.src {
			t.Errorf("%s.source() = %s, want %s", c.e, got, c.src)
		}
		if got := c.e.Target(); got != c.dst {
			t.Errorf("%s.Target() = %s, want %s", c.e, got, c.dst)
		}
	}
}

// walkAll runs a walk and collects the emitted nodes.
func walkAll(t *testing.T, g grapher, seeds []Seed, opts WalkOptions) []*Node {
	t.Helper()
	var got []*Node
	err := NewWalker(g).Walk(context.Background(), seeds, opts, func(n *Node) error {
		got = append(got, n)
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	return got
}

func endpointsByKind(nodes []*Node, kind NodeKind) []string {
	var out []string
	for _, n := range nodes {
		if n.Kind == kind {
			out = append(out, n.Endpoint())
		}
	}
	return out
}

func TestWalkContentFromVideo(t *testing.T) {
	g := newFakeGraph()
	nodes := walkAll(t, g, []Seed{{Raw: "BV_TURING"}}, WalkOptions{Depth: 1, Edges: DefaultEdges()})
	if len(nodes) == 0 || nodes[0].Endpoint() != "BV_TURING" || nodes[0].Depth != 0 {
		t.Fatalf("first node = %+v, want BV_TURING at depth 0", nodes[0])
	}
	vids := endpointsByKind(nodes, NodeVideo)
	if !contains(vids, "BV_REL1") || !contains(vids, "BV_REL2") {
		t.Errorf("videos = %v, want the related videos", vids)
	}
	// content does not include the uploader edge.
	if len(endpointsByKind(nodes, NodeUser)) != 0 {
		t.Errorf("content should not surface creators, got %v", endpointsByKind(nodes, NodeUser))
	}
	for _, n := range nodes {
		if n.Depth == 1 && n.Via != EdgeRelated {
			t.Errorf("depth-1 node via %q, want related: %+v", n.Via, n)
		}
	}
}

func TestWalkCreatorsFromVideo(t *testing.T) {
	g := newFakeGraph()
	nodes := walkAll(t, g, []Seed{{Raw: "BV_TURING"}}, WalkOptions{Depth: 1, Edges: edgePresets["creators"]})
	users := endpointsByKind(nodes, NodeUser)
	if !contains(users, "2") {
		t.Errorf("creators = %v, want the uploader mid 2", users)
	}
	// related is not in the creators preset.
	if contains(endpointsByKind(nodes, NodeVideo), "BV_REL1") {
		t.Errorf("creators should not follow related videos")
	}
	for _, n := range nodes {
		if n.Kind == NodeUser && n.Via != EdgeUploader {
			t.Errorf("user reached via %q, want uploader", n.Via)
		}
	}
}

func TestWalkFromUser(t *testing.T) {
	g := newFakeGraph()
	nodes := walkAll(t, g, []Seed{{Raw: "https://space.bilibili.com/2"}}, WalkOptions{Depth: 1, Edges: DefaultEdges()})
	if nodes[0].Kind != NodeUser || nodes[0].Endpoint() != "2" {
		t.Fatalf("seed = %+v, want user 2", nodes[0])
	}
	vids := endpointsByKind(nodes, NodeVideo)
	if !contains(vids, "BV_UP1") || !contains(vids, "BV_UP2") {
		t.Errorf("uploads = %v, want the creator's videos", vids)
	}
	for _, n := range nodes {
		if n.Kind == NodeVideo && n.Via != EdgeUploads {
			t.Errorf("video reached via %q, want uploads", n.Via)
		}
	}
}

func TestWalkDepth2(t *testing.T) {
	g := newFakeGraph()
	nodes := walkAll(t, g, []Seed{{Raw: "BV_TURING"}}, WalkOptions{Depth: 2, Edges: edgePresets["creators"]})
	// seed video -> uploader user 2 (d1) -> user 2's uploads (d2)
	vids := endpointsByKind(nodes, NodeVideo)
	if !contains(vids, "BV_UP1") || !contains(vids, "BV_UP2") {
		t.Errorf("depth-2 uploads = %v, want BV_UP1/BV_UP2", vids)
	}
}

func TestWalkDedup(t *testing.T) {
	g := newFakeGraph()
	nodes := walkAll(t, g, []Seed{{Raw: "BV_TURING"}}, WalkOptions{Depth: 3, Edges: edgePresets["all"]})
	seen := map[string]int{}
	for _, n := range nodes {
		seen[string(n.Kind)+":"+n.Endpoint()]++
	}
	for k, c := range seen {
		if c != 1 {
			t.Errorf("node %s emitted %d times, want 1", k, c)
		}
	}
	// The BV_REL1 -> BV_TURING cycle must not re-emit the seed.
	if seen["video:BV_TURING"] != 1 {
		t.Errorf("seed emitted %d times despite the cycle", seen["video:BV_TURING"])
	}
}

func TestWalkBudgetStops(t *testing.T) {
	g := newFakeGraph()
	nodes := walkAll(t, g, []Seed{{Raw: "BV_TURING"}}, WalkOptions{Depth: 3, Max: 2, Edges: edgePresets["all"]})
	if len(nodes) != 2 {
		t.Errorf("emitted %d nodes, want exactly the budget of 2", len(nodes))
	}
}

func TestWalkFanoutCaps(t *testing.T) {
	g := newFakeGraph()
	nodes := walkAll(t, g, []Seed{{Raw: "BV_TURING"}}, WalkOptions{Depth: 1, Fanout: 1, Edges: newEdgeSet(EdgeRelated)})
	if len(nodes) != 2 {
		t.Errorf("emitted %d nodes, want 2 (seed + 1 related)", len(nodes))
	}
}

func TestWalkDepthZeroSeedsOnly(t *testing.T) {
	g := newFakeGraph()
	nodes := walkAll(t, g, []Seed{{Raw: "BV_TURING"}}, WalkOptions{Depth: 0, Edges: edgePresets["all"]})
	if len(nodes) != 1 || nodes[0].Endpoint() != "BV_TURING" {
		t.Errorf("depth 0 = %v, want only the seed", endpointsByKind(nodes, NodeVideo))
	}
}

func TestWalkSeedNotFoundFatal(t *testing.T) {
	g := newFakeGraph()
	g.fail["video:BV_MISSING"] = fmt.Errorf("not found")
	err := NewWalker(g).Walk(context.Background(), []Seed{{Raw: "BV_MISSING"}}, WalkOptions{Depth: 1, Edges: DefaultEdges()}, func(*Node) error { return nil })
	if err == nil {
		t.Error("a seed that cannot be fetched should fail the walk")
	}
}

func TestWalkUnsupportedSeedFatal(t *testing.T) {
	g := newFakeGraph()
	err := NewWalker(g).Walk(context.Background(), []Seed{{Raw: "ss123"}}, WalkOptions{Depth: 1, Edges: DefaultEdges()}, func(*Node) error { return nil })
	if err == nil {
		t.Error("a bangumi seed is unsupported and should fail the walk")
	}
}

func TestWalkDeeperErrorDegrades(t *testing.T) {
	g := newFakeGraph()
	// The related edge of the seed fails; the uploader edge must still be
	// followed, and the failure should be a note, not a fatal error.
	g.fail["related:BV_TURING"] = fmt.Errorf("risk control")
	var notes int
	nodes := walkAll(t, g, []Seed{{Raw: "BV_TURING"}}, WalkOptions{
		Depth: 1, Edges: edgePresets["all"],
		Note: func(string) { notes++ },
	})
	if !contains(endpointsByKind(nodes, NodeUser), "2") {
		t.Errorf("the uploader should survive a failed related edge: %v", nodes)
	}
	if contains(endpointsByKind(nodes, NodeVideo), "BV_REL1") {
		t.Errorf("the failed related edge should yield no related videos")
	}
	if notes == 0 {
		t.Error("a failed edge should produce a note")
	}
}

func TestWalkUploadsPartialError(t *testing.T) {
	g := newFakeGraph()
	// The uploads stream returns its videos and then trips: the videos survive
	// and a note is produced.
	g.fail["uploads:2"] = fmt.Errorf("risk control")
	var notes int
	nodes := walkAll(t, g, []Seed{{Raw: "https://space.bilibili.com/2"}}, WalkOptions{
		Depth: 1, Edges: newEdgeSet(EdgeUploads),
		Note: func(string) { notes++ },
	})
	vids := endpointsByKind(nodes, NodeVideo)
	if !contains(vids, "BV_UP1") || !contains(vids, "BV_UP2") {
		t.Errorf("uploads read before the failure should survive: %v", vids)
	}
	if notes == 0 {
		t.Error("the trailing failure should produce a note")
	}
}

func TestWalkMultipleSeeds(t *testing.T) {
	g := newFakeGraph()
	g.resolved["BV_REL2"] = &Identity{Kind: KindVideo, BVID: "BV_REL2"}
	g.videos["BV_REL2"] = Video{BVID: "BV_REL2", Title: "Related two", OwnerMid: 2, OwnerName: "Creator A"}
	nodes := walkAll(t, g, []Seed{{Raw: "BV_TURING"}, {Raw: "BV_REL2"}}, WalkOptions{Depth: 0, Edges: DefaultEdges()})
	vids := endpointsByKind(nodes, NodeVideo)
	if !contains(vids, "BV_TURING") || !contains(vids, "BV_REL2") {
		t.Errorf("both seeds should be emitted: %v", vids)
	}
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
