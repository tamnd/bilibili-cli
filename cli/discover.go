package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tamnd/any-cli/kit/render"
	"github.com/tamnd/bilibili-cli/bili"
)

// defaultDiscoverBudget caps a streaming walk when -n is not given, so
// `bili discover <id>` always terminates instead of spidering bilibili forever.
const defaultDiscoverBudget = 500

// newDiscoverCmd is the breadth-first graph walk. Where each single command
// answers one question about one object (a video's related list, a creator's
// uploads), discover chains them: from a seed video or user it follows the
// object's edges and from each neighbor it follows theirs, hop by hop, streaming
// one record per node as it is reached.
func newDiscoverCmd(a *App) *cobra.Command {
	var (
		depth  int
		fanout int
		follow string
	)
	cmd := &cobra.Command{
		Use:     "discover <id|url>...",
		Aliases: []string{"walk", "graph"},
		Short:   "Breadth-first walk of the graph linked from a video or creator",
		Long: `Walk the graph of connected bilibili objects, breadth first, starting from one
or more seeds. A seed is anything bili can resolve to a video or a creator: a
BV/av id, a video URL, or a space.bilibili.com creator URL.

--follow chooses which edges to traverse. It takes a preset or a comma-separated
edge list:

  content   a video's related videos, and a creator's uploads (the default;
            stays in video space)
  creators  a video's uploader, and a creator's uploads (bounces between videos
            and the people who make them)
  all       every edge

Edges: related, uploader, uploads. Name an edge directly to follow just that one,
e.g. --follow uploader to hop from videos to their creators.

--depth is how many hops to follow (default 1; 0 emits only the seeds). --fanout
caps neighbors per edge (default 25). The walk streams nodes and stops after -n
nodes (default 500).

Most read endpoints are open to anonymous callers; a few (creator catalogues in
particular) are gated by bilibili's anti-bot for some IPs. A gated edge deeper in
the walk becomes a one-line note and the walk carries on; supplying a logged-in
--cookie or BILI_COOKIE widens what a walk can reach.

bili discover streams to stdout. To keep a walk, pipe it:
  bili discover BV17x411w7KC --depth 2 -o jsonl > graph.jsonl
For split per-type files instead, see bili crawl.`,
		Args:    cobra.MinimumNArgs(1),
		Example: "  bili discover BV17x411w7KC\n  bili discover BV17x411w7KC --depth 2 -o jsonl > graph.jsonl\n  bili discover https://space.bilibili.com/2 --follow uploads\n  bili search lofi -o url | bili discover - --depth 1",
		RunE: func(cmd *cobra.Command, args []string) error {
			edges, err := bili.ParseEdges(follow)
			if err != nil {
				return err
			}
			seeds := toSeeds(readArgsOrStdin(args))
			if len(seeds) == 0 {
				return fmt.Errorf("no seeds given")
			}

			budget := a.limit
			if budget <= 0 {
				budget = defaultDiscoverBudget
			}

			opts := bili.WalkOptions{
				Depth:  depth,
				Max:    budget,
				Fanout: fanout,
				Edges:  edges,
				Note: func(s string) {
					if !a.quiet {
						fmt.Fprintf(os.Stderr, "bili: note: %s\n", s)
					}
				},
			}

			out, err := a.newOutput()
			if err != nil {
				return err
			}
			walkErr := a.Client().Walk(a.ctx(), seeds, opts, func(n *bili.Node) error {
				return out.Emit(nodeRow(n))
			})
			closeErr := out.Close()
			if walkErr != nil {
				return walkErr
			}
			return closeErr
		},
	}
	f := cmd.Flags()
	f.IntVar(&depth, "depth", 1, "hops to follow from each seed (0 = seeds only)")
	f.IntVar(&fanout, "fanout", 25, "max neighbors to follow per edge (0 = unlimited)")
	f.StringVar(&follow, "follow", "content", "edges to follow ("+bili.EdgeHelp()+")")
	return cmd
}

// toSeeds wraps raw arguments as walk seeds. Classification happens at walk time
// via the client's resolver, so a seed that resolves to neither a video nor a
// creator fails the walk like any bad id.
func toSeeds(args []string) []bili.Seed {
	seeds := make([]bili.Seed, 0, len(args))
	for _, s := range args {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		seeds = append(seeds, bili.Seed{Raw: s})
	}
	return seeds
}

// nodeRow renders a graph node discovered by `bili discover`. The curated
// columns read the walk at a glance (how deep, by which edge, the object and its
// owner) while the full typed node rides in Value for json, jsonl, and templates.
func nodeRow(n *bili.Node) render.Record {
	var id, title, owner, url string
	switch n.Kind {
	case bili.NodeVideo:
		v := n.Video
		id, title, owner, url = v.BVID, oneline(v.Title), v.OwnerName, v.URL
	case bili.NodeUser:
		u := n.User
		id, title = fmt.Sprint(u.Mid), u.Name
		url = "https://space.bilibili.com/" + fmt.Sprint(u.Mid)
	}
	return render.Record{
		Cols:  []string{"depth", "via", "kind", "id", "title", "owner", "url"},
		Vals:  []string{fmt.Sprint(n.Depth), string(n.Via), string(n.Kind), id, title, owner, url},
		Value: n,
	}
}

// oneline flattens a value to a single short line for a curated column, so a
// long title never breaks the list and table layouts.
func oneline(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 80 {
		return s[:79] + "..."
	}
	return s
}
