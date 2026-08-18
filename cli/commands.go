package cli

import (
	"fmt"
	"iter"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tamnd/bilibili-cli/bili"
)

// emit writes records from a slice and closes the output.
func emitAll[T any](a *App, items []T) error {
	out, err := a.newOutput()
	if err != nil {
		return err
	}
	for _, it := range items {
		if err := out.Emit(it); err != nil {
			return err
		}
	}
	if err := out.Close(); err != nil {
		return err
	}
	return noResults(a, out)
}

// emitSeq writes records from an iterator (streaming) and closes the output.
func emitSeq[T any](a *App, seq iter.Seq2[T, error]) error {
	return emitSeqFunc(a, seq, func(v T) any { return v })
}

// emitSeqFunc is emitSeq with a per-record transform, used to unwrap a wrapper
// record (e.g. SearchResult) into its concrete payload before emitting.
func emitSeqFunc[T any](a *App, seq iter.Seq2[T, error], fn func(T) any) error {
	out, err := a.newOutput()
	if err != nil {
		return err
	}
	var seqErr error
	for v, e := range seq {
		if e != nil {
			seqErr = e
			break
		}
		if err := out.Emit(fn(v)); err != nil {
			return err
		}
	}
	if cerr := out.Close(); cerr != nil && seqErr == nil {
		seqErr = cerr
	}
	if seqErr != nil {
		return seqErr
	}
	return noResults(a, out)
}

// noResults reports ErrNoResults when a command finished cleanly and wrote
// nothing. That is a result and it gets its own exit code, so that a caller can
// tell an empty answer from a refusal without reading the records.
//
// A dry run is exempt. It writes nothing by design, and reporting that as an
// empty result would say something about the site when nothing was asked of it.
func noResults(a *App, out *Output) error {
	if a.dryRun || out.Records() > 0 {
		return nil
	}
	return ErrNoResults
}

// runEach applies fn to every target and keeps going after a failure, because
// one refused folder listing in five hundred is not a refused run.
//
// Every failure is named on stderr as it happens, and the counts follow, so a
// run that partly worked says which part did not rather than leaving the caller
// to diff the output against the input.
func runEach[T any](a *App, targets []string, fn func(string) ([]T, error)) ([]T, error) {
	var all []T
	var failed []error
	for _, t := range targets {
		vs, err := fn(t)
		if err != nil {
			failed = append(failed, err)
			a.progress("%s: %v", t, err)
			continue
		}
		all = append(all, vs...)
	}
	return all, runError(a, len(targets), failed)
}

// runError reduces per-target failures into the one error that decides the exit
// code for the run.
//
// A status becomes the run's exit code only when it covers every target. A run
// that partly worked exits 0 with its counts on stderr, and a run where every
// target failed differently exits 1, because no single status describes it and
// picking one of them would be a guess presented as a fact.
func runError(a *App, total int, failed []error) error {
	if len(failed) == 0 {
		return nil
	}
	a.progress("%d of %d failed", len(failed), total)
	if len(failed) < total {
		return nil
	}
	kind := bili.Kind(failed[0])
	for _, e := range failed[1:] {
		if bili.Kind(e) != kind {
			// %v and not %w on purpose: wrapping would let the exit code reach
			// through to a status that covers one target out of many.
			return fmt.Errorf("all %d targets failed, in more than one way, the first being: %v", total, failed[0])
		}
	}
	return failed[0]
}

func emitOne(a *App, v any) error {
	return emitAll(a, []any{v})
}

// ---- video ----

func newVideoCmd(a *App) *cobra.Command {
	var pages, related, tags, stat, noDetail bool
	cmd := &cobra.Command{
		Use:     "video <id|url>...",
		Short:   "Resolve one or more videos to full metadata",
		Args:    cobra.MinimumNArgs(1),
		Example: "  bili video BV1GJ411x7h7\n  bili video av170001 -o json\n  bili video BV1GJ411x7h7 --pages\n  bili search lofi -o url | bili video -",
		RunE: func(cmd *cobra.Command, args []string) error {
			ids := readArgsOrStdin(args)
			c := a.Client()
			if pages {
				ps, err := runEach(a, ids, func(id string) ([]bili.Page, error) {
					return c.Pages(a.ctx(), id)
				})
				if err != nil {
					return err
				}
				return emitAll(a, ps)
			}
			if tags {
				ts, err := runEach(a, ids, func(id string) ([]map[string]string, error) {
					res, err := c.Video(a.ctx(), id, bili.VideoOptions{})
					if err != nil {
						return nil, err
					}
					out := make([]map[string]string, 0, len(res.Video.Tags))
					for _, t := range res.Video.Tags {
						out = append(out, map[string]string{"tag": t})
					}
					return out, nil
				})
				if err != nil {
					return err
				}
				return emitAll(a, ts)
			}
			if related {
				rel, err := runEach(a, ids, func(id string) ([]bili.Video, error) {
					return c.Related(a.ctx(), id)
				})
				if err != nil {
					return err
				}
				return emitAll(a, rel)
			}
			vids, err := runEach(a, ids, func(id string) ([]bili.Video, error) {
				res, err := c.Video(a.ctx(), id, bili.VideoOptions{Related: related})
				if err != nil {
					return nil, err
				}
				return []bili.Video{res.Video}, nil
			})
			if err != nil {
				return err
			}
			return emitAll(a, vids)
		},
	}
	f := cmd.Flags()
	f.BoolVar(&pages, "pages", false, "list the parts (P) with cids")
	f.BoolVar(&related, "related", false, "list related videos instead")
	f.BoolVar(&tags, "tags", false, "list the video tags only")
	f.BoolVar(&stat, "stat", false, "live stat counters only")
	f.BoolVar(&noDetail, "no-detail", false, "basic view, faster")
	return cmd
}

// ---- user ----

func newUserCmd(a *App) *cobra.Command {
	var videos, dynamics, stat bool
	var keyword string
	cmd := &cobra.Command{
		Use:     "user <mid|url>",
		Short:   "A creator's profile, catalogue, stat, or dynamics",
		Args:    cobra.ExactArgs(1),
		Example: "  bili user 2\n  bili user 2 --videos --limit 50\n  bili user 2 --dynamics",
		RunE: func(cmd *cobra.Command, args []string) error {
			mid := normalizeMid(a, args[0])
			c := a.Client()
			switch {
			case videos:
				return emitSeq(a, c.UserVideos(a.ctx(), mid, a.listOpts(keyword)))
			case dynamics:
				return emitSeq(a, c.Dynamics(a.ctx(), mid, a.listOpts("")))
			default:
				u, err := c.User(a.ctx(), mid)
				if err != nil {
					return err
				}
				return emitOne(a, u)
			}
		},
	}
	f := cmd.Flags()
	f.BoolVar(&videos, "videos", false, "stream uploaded videos")
	f.BoolVar(&dynamics, "dynamics", false, "stream the dynamics feed")
	f.BoolVar(&stat, "stat", false, "follower/following/views/likes only")
	f.StringVar(&keyword, "keyword", "", "filter the catalogue server-side")
	return cmd
}

func normalizeMid(a *App, s string) string {
	if id, err := a.Client().Resolve(a.ctx(), s); err == nil && id.Mid != 0 {
		return fmt.Sprint(id.Mid)
	}
	// strip space.bilibili.com/<mid>
	if i := strings.LastIndex(s, "/"); i >= 0 && i < len(s)-1 {
		return strings.TrimSpace(s[i+1:])
	}
	return s
}

// ---- search ----

func newSearchCmd(a *App) *cobra.Command {
	var typ, order string
	var duration, tid int
	cmd := &cobra.Command{
		Use:     "search <query>",
		Short:   "Search videos, users, bangumi, live rooms, or articles",
		Args:    cobra.MinimumNArgs(1),
		Example: "  bili search 原神\n  bili search 原神 --type video --limit 100\n  bili search 凡人修仙传 --type bangumi",
		RunE: func(cmd *cobra.Command, args []string) error {
			q := strings.Join(args, " ")
			opt := bili.SearchOptions{
				Type: typ, Order: firstNonEmpty(order, a.order), Duration: duration,
				Tid: tid, Page: a.page, PageSize: a.pageSize, Limit: a.limit,
			}
			return emitSeqFunc(a, a.Client().Search(a.ctx(), q, opt),
				func(r bili.SearchResult) any { return r.Payload() })
		},
	}
	f := cmd.Flags()
	f.StringVar(&typ, "type", "video", "video|user|bangumi|film|live_room|article|all")
	f.StringVar(&order, "order", "", "totalrank|click|pubdate|dm|stow")
	f.IntVar(&duration, "duration", 0, "1:<10m 2:10-30m 3:30-60m 4:>60m")
	f.IntVar(&tid, "tid", 0, "restrict to a partition id")
	return cmd
}

// ---- comments ----

func newCommentsCmd(a *App) *cobra.Command {
	var order string
	var replies bool
	cmd := &cobra.Command{
		Use:     "comments <id|url>",
		Short:   "Every comment (and reply) on a video, article, audio, or dynamic",
		Args:    cobra.ExactArgs(1),
		Example: "  bili comments BV1GJ411x7h7\n  bili comments BV1GJ411x7h7 --order time --replies",
		RunE: func(cmd *cobra.Command, args []string) error {
			opt := bili.CommentOptions{Order: firstNonEmpty(order, a.order), Replies: replies, Limit: a.limit}
			return emitSeq(a, a.Client().Comments(a.ctx(), args[0], opt))
		},
	}
	cmd.Flags().StringVar(&order, "order", "hot", "hot|time")
	cmd.Flags().BoolVar(&replies, "replies", false, "expand nested replies")
	return cmd
}

// ---- danmaku ----

func newDanmakuCmd(a *App) *cobra.Command {
	var part int
	var xmlMode bool
	cmd := &cobra.Command{
		Use:     "danmaku <id|url>",
		Short:   "Bullet-chat (danmaku) for a video part",
		Args:    cobra.ExactArgs(1),
		Example: "  bili danmaku BV1GJ411x7h7\n  bili danmaku BV1GJ411x7h7 -p 2 -o csv",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := a.Client()
			if xmlMode {
				dms, err := c.DanmakuXML(a.ctx(), args[0], part)
				if err != nil {
					return err
				}
				if a.limit > 0 && len(dms) > a.limit {
					dms = dms[:a.limit]
				}
				return emitAll(a, dms)
			}
			seq := c.Danmaku(a.ctx(), args[0], part)
			if a.limit > 0 {
				seq = limitSeq(seq, a.limit)
			}
			return emitSeq(a, seq)
		},
	}
	cmd.Flags().IntVarP(&part, "part", "p", 1, "which part (P)")
	cmd.Flags().BoolVar(&xmlMode, "xml", false, "legacy XML snapshot transport")
	return cmd
}

func limitSeq[T any](seq iter.Seq2[T, error], n int) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		i := 0
		for v, e := range seq {
			if !yield(v, e) {
				return
			}
			if e != nil {
				return
			}
			i++
			if i >= n {
				return
			}
		}
	}
}

// ---- dynamic / dynamics ----

func newDynamicCmd(a *App) *cobra.Command {
	return &cobra.Command{
		Use:   "dynamic <id|url>",
		Short: "One dynamic (feed) post in full",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if rid, err := a.Client().Resolve(a.ctx(), id); err == nil && rid.Dynamic != "" {
				id = rid.Dynamic
			}
			d, err := a.Client().Dynamic(a.ctx(), id)
			if err != nil {
				return err
			}
			return emitOne(a, d)
		},
	}
}

func newDynamicsCmd(a *App) *cobra.Command {
	return &cobra.Command{
		Use:   "dynamics <mid|url>",
		Short: "A user's whole dynamics feed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return emitSeq(a, a.Client().Dynamics(a.ctx(), normalizeMid(a, args[0]), a.listOpts("")))
		},
	}
}

// ---- live ----

func newLiveCmd(a *App) *cobra.Command {
	var byUID, browse bool
	var area int
	cmd := &cobra.Command{
		Use:     "live <room|url>",
		Short:   "Live room info, or browse rooms by area",
		Args:    cobra.MaximumNArgs(1),
		Example: "  bili live 21452505\n  bili live --uid 2\n  bili live --browse --area 1",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := a.Client()
			if browse {
				return emitSeq(a, c.BrowseLive(a.ctx(), area, a.listOpts("")))
			}
			if len(args) == 0 {
				return fmt.Errorf("provide a room id or --uid/--browse")
			}
			room, err := c.Live(a.ctx(), normalizeMid(a, args[0]), byUID)
			if err != nil {
				return err
			}
			return emitOne(a, room)
		},
	}
	cmd.Flags().BoolVar(&byUID, "uid", false, "treat the argument as a streamer uid")
	cmd.Flags().BoolVar(&browse, "browse", false, "browse live rooms in an area")
	cmd.Flags().IntVar(&area, "area", 0, "area id for --browse")
	return cmd
}

// ---- bangumi ----

func newBangumiCmd(a *App) *cobra.Command {
	var episodes bool
	cmd := &cobra.Command{
		Use:     "bangumi <ss|ep|md|url>",
		Short:   "An anime/film season with every episode",
		Args:    cobra.ExactArgs(1),
		Example: "  bili bangumi ss12345\n  bili bangumi ep234567 --episodes",
		RunE: func(cmd *cobra.Command, args []string) error {
			b, err := a.Client().Bangumi(a.ctx(), args[0])
			if err != nil {
				return err
			}
			if episodes {
				return emitAll(a, b.Episodes)
			}
			return emitOne(a, b)
		},
	}
	cmd.Flags().BoolVar(&episodes, "episodes", false, "emit the episode list")
	return cmd
}

// ---- audio ----

func newAudioCmd(a *App) *cobra.Command {
	return &cobra.Command{
		Use:   "audio <au|url>",
		Short: "An audio track's metadata and stat",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			au, err := a.Client().Audio(a.ctx(), args[0])
			if err != nil {
				return err
			}
			return emitOne(a, au)
		},
	}
}

// ---- article ----

func newArticleCmd(a *App) *cobra.Command {
	var text bool
	cmd := &cobra.Command{
		Use:   "article <cv|url>",
		Short: "A column article's metadata (and text)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			art, err := a.Client().Article(a.ctx(), args[0], text)
			if err != nil {
				return err
			}
			if text {
				fmt.Println(art.ContentText)
				return nil
			}
			return emitOne(a, art)
		},
	}
	cmd.Flags().BoolVar(&text, "text", false, "print the plain-text body")
	return cmd
}

// ---- favorites / favorite ----

func newFavoritesCmd(a *App) *cobra.Command {
	return &cobra.Command{
		Use:   "favorites <mid|url>",
		Short: "A user's favorite folders",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			favs, err := a.Client().Favorites(a.ctx(), normalizeMid(a, args[0]))
			if err != nil {
				return err
			}
			return emitAll(a, favs)
		},
	}
}

func newFavoriteCmd(a *App) *cobra.Command {
	return &cobra.Command{
		Use:   "favorite <ml|url>",
		Short: "The videos inside a favorite folder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return emitSeq(a, a.Client().FavoriteItems(a.ctx(), args[0], a.listOpts("")))
		},
	}
}

// ---- discovery ----

func newPopularCmd(a *App) *cobra.Command {
	var weekly int
	cmd := &cobra.Command{
		Use:   "popular",
		Short: "The popular feed, or a weekly selection issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			if weekly > 0 {
				vs, err := a.Client().Weekly(a.ctx(), weekly)
				if err != nil {
					return err
				}
				return emitAll(a, vs)
			}
			return emitSeq(a, a.Client().Popular(a.ctx(), a.listOpts("")))
		},
	}
	cmd.Flags().IntVar(&weekly, "weekly", 0, "weekly selection issue number")
	return cmd
}

func newRankCmd(a *App) *cobra.Command {
	var tid int
	cmd := &cobra.Command{
		Use:   "rank",
		Short: "The leaderboard, optionally for one partition",
		RunE: func(cmd *cobra.Command, args []string) error {
			vs, err := a.Client().Ranking(a.ctx(), tid)
			if err != nil {
				return err
			}
			if a.limit > 0 && len(vs) > a.limit {
				vs = vs[:a.limit]
			}
			return emitAll(a, vs)
		},
	}
	cmd.Flags().IntVar(&tid, "tid", 0, "partition id (rid)")
	return cmd
}

func newTrendingCmd(a *App) *cobra.Command {
	return &cobra.Command{
		Use:   "trending",
		Short: "Current hot search terms",
		RunE: func(cmd *cobra.Command, args []string) error {
			limit := a.limit
			if limit == 0 {
				limit = 10
			}
			terms, err := a.Client().Trending(a.ctx(), limit)
			if err != nil {
				return err
			}
			return emitTerms(a, terms)
		},
	}
}

func newRelatedCmd(a *App) *cobra.Command {
	return &cobra.Command{
		Use:   "related <id|url>",
		Short: "Related videos for a video",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vs, err := a.Client().Related(a.ctx(), args[0])
			if err != nil {
				return err
			}
			if a.limit > 0 && len(vs) > a.limit {
				vs = vs[:a.limit]
			}
			return emitAll(a, vs)
		},
	}
}

func newSuggestCmd(a *App) *cobra.Command {
	return &cobra.Command{
		Use:   "suggest <term>",
		Short: "Search autosuggest terms",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			terms, err := a.Client().Suggest(a.ctx(), args[0])
			if err != nil {
				return err
			}
			return emitTerms(a, terms)
		},
	}
}

func emitTerms(a *App, terms []bili.Suggestion) error {
	return emitAll(a, terms)
}

// ---- streams ----

func newStreamsCmd(a *App) *cobra.Command {
	var part, quality int
	cmd := &cobra.Command{
		Use:     "streams <id|url>",
		Short:   "Playable stream URLs for a video part",
		Args:    cobra.ExactArgs(1),
		Example: "  bili streams BV1GJ411x7h7\n  bili streams BV1GJ411x7h7 --quality 80 -o json",
		RunE: func(cmd *cobra.Command, args []string) error {
			ss, err := a.Client().Streams(a.ctx(), args[0], part, quality)
			if err != nil {
				return err
			}
			return emitAll(a, ss)
		},
	}
	cmd.Flags().IntVarP(&part, "part", "p", 1, "which part (P)")
	cmd.Flags().IntVar(&quality, "quality", 0, "quality code (qn)")
	return cmd
}

// ---- id / nav ----

func newIDCmd(a *App) *cobra.Command {
	return &cobra.Command{
		Use:   "id <thing>",
		Short: "Classify and normalize any id or URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := a.Client().Resolve(a.ctx(), args[0])
			if err != nil && id == nil {
				return err
			}
			return emitOne(a, id)
		},
	}
}

func newNavCmd(a *App) *cobra.Command {
	return &cobra.Command{
		Use:   "nav",
		Short: "Login state and current WBI keys (debug)",
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := a.Client().Nav(a.ctx())
			if err != nil {
				return err
			}
			return emitOne(a, n)
		},
	}
}

func (a *App) listOpts(keyword string) bili.ListOptions {
	return bili.ListOptions{
		Page: a.page, PageSize: a.pageSize, Order: a.order, Keyword: keyword, Limit: a.limit,
	}
}
