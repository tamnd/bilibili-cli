package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tamnd/bilibili-cli/bili"
)

// ---- version ----

func newVersionCmd(a *App) *cobra.Command {
	var short bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version, commit, and build date",
		RunE: func(cmd *cobra.Command, args []string) error {
			if short {
				fmt.Println(Version)
				return nil
			}
			info := map[string]string{"version": Version, "commit": Commit, "date": Date}
			f := a.resolveFormat()
			if f == FormatTable || f == FormatAuto || f == FormatList {
				fmt.Printf("bili %s (%s) built %s\n", Version, Commit, Date)
				return nil
			}
			b, _ := json.MarshalIndent(info, "", "  ")
			fmt.Println(string(b))
			return nil
		},
	}
	cmd.Flags().BoolVar(&short, "short", false, "print just the version string")
	return cmd
}

// ---- cache ----

func newCacheCmd(a *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Inspect or clear the on-disk response cache",
	}
	dir := bili.DefaultConfig().CacheDir

	stat := &cobra.Command{
		Use:   "stat",
		Short: "Show cache location, file count, and size",
		RunE: func(cmd *cobra.Command, args []string) error {
			files, bytes := bili.CacheStats(dir)
			return emitOne(a, map[string]any{
				"dir": dir, "files": files, "bytes": bytes,
				"size": humanBytes(bytes),
			})
		},
	}
	clear := &cobra.Command{
		Use:   "clear",
		Short: "Delete every cached response",
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := bili.ClearCache(dir)
			if err != nil {
				return err
			}
			a.progress("removed %d cached files from %s", n, dir)
			return nil
		},
	}
	path := &cobra.Command{
		Use:   "path",
		Short: "Print the cache directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(dir)
			return nil
		},
	}
	cmd.AddCommand(stat, clear, path)
	return cmd
}

func humanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// ---- config ----

func newConfigCmd(a *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show resolved configuration and important paths",
	}
	show := &cobra.Command{
		Use:   "show",
		Short: "Print effective settings (secrets redacted)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cookie := a.resolveCookie()
			return emitOne(a, map[string]any{
				"cache_dir":  bili.DefaultConfig().CacheDir,
				"config_dir": bili.ConfigDir(),
				"data_dir":   bili.DataDir(),
				"user_agent": firstNonEmpty(a.userAgent, os.Getenv("BILI_USER_AGENT"), bili.DefaultUserAgent),
				"cookie_set": cookie != "",
				"cookie":     redactCookie(cookie),
				"rate":       a.rate.String(),
				"retries":    a.retries,
				"timeout":    a.timeout.String(),
				"cache_ttl":  a.cacheTTL.String(),
				"proxy":      firstNonEmpty(a.proxy, os.Getenv("BILI_PROXY")),
			})
		},
	}
	paths := &cobra.Command{
		Use:   "path",
		Short: "Print config, cache, and data directories",
		RunE: func(cmd *cobra.Command, args []string) error {
			return emitOne(a, map[string]any{
				"config_dir": bili.ConfigDir(),
				"cache_dir":  bili.DefaultConfig().CacheDir,
				"data_dir":   bili.DataDir(),
			})
		},
	}
	cmd.AddCommand(show, paths)
	return cmd
}

func redactCookie(c string) string {
	if c == "" {
		return ""
	}
	parts := strings.Split(c, ";")
	for i, p := range parts {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) == 2 {
			v := kv[1]
			if len(v) > 6 {
				v = v[:3] + "…" + v[len(v)-3:]
			} else {
				v = "…"
			}
			parts[i] = kv[0] + "=" + v
		}
	}
	return strings.Join(parts, "; ")
}

// ---- crawl ----

// newCrawlCmd walks outward from seed ids, fetching connected records and writing
// them as JSONL (one file per record kind) into an output directory. It is the
// "crawl everything" entry point: video -> pages, tags, related, owner, comments,
// danmaku.
func newCrawlCmd(a *App) *cobra.Command {
	var outDir string
	var withComments, withDanmaku, withRelated, withUser bool
	cmd := &cobra.Command{
		Use:     "crawl <id|url>...",
		Short:   "Crawl connected records from seed ids into JSONL files",
		Args:    cobra.MinimumNArgs(1),
		Example: "  bili crawl BV17x411w7KC --out ./data\n  bili search lofi -o url | bili crawl - --out ./data --comments",
		RunE: func(cmd *cobra.Command, args []string) error {
			if outDir == "" {
				outDir = "."
			}
			if err := os.MkdirAll(outDir, 0o755); err != nil {
				return err
			}
			c := a.Client()
			seeds := readArgsOrStdin(args)

			vw := newJSONLWriter(filepath.Join(outDir, "videos.jsonl"))
			defer func() { _ = vw.Close() }()
			uw := newJSONLWriter(filepath.Join(outDir, "users.jsonl"))
			defer func() { _ = uw.Close() }()
			cw := newJSONLWriter(filepath.Join(outDir, "comments.jsonl"))
			defer func() { _ = cw.Close() }()
			dw := newJSONLWriter(filepath.Join(outDir, "danmaku.jsonl"))
			defer func() { _ = dw.Close() }()

			seenUser := map[int64]bool{}
			for _, seed := range seeds {
				res, err := c.Video(a.ctx(), seed, bili.VideoOptions{})
				if err != nil {
					a.progress("skip %s: %v", seed, err)
					continue
				}
				v := res.Video
				vw.Write(v)
				a.progress("video %s %q", v.BVID, v.Title)

				if withUser && v.OwnerMid != 0 && !seenUser[v.OwnerMid] {
					seenUser[v.OwnerMid] = true
					if u, err := c.User(a.ctx(), fmt.Sprint(v.OwnerMid)); err == nil {
						uw.Write(u)
					}
				}
				if withRelated {
					if rel, err := c.Related(a.ctx(), v.BVID); err == nil {
						for _, r := range rel {
							vw.Write(r)
						}
					}
				}
				if withComments {
					n := 0
					for cm, err := range c.Comments(a.ctx(), v.BVID, bili.CommentOptions{Order: "hot", Replies: true, Limit: a.limit}) {
						if err != nil {
							break
						}
						cw.Write(cm)
						n++
					}
					a.progress("  %d comments", n)
				}
				if withDanmaku {
					n := 0
					for dm, err := range c.Danmaku(a.ctx(), v.BVID, 1) {
						if err != nil {
							break
						}
						dw.Write(dm)
						n++
						if a.limit > 0 && n >= a.limit {
							break
						}
					}
					a.progress("  %d danmaku", n)
				}
			}
			a.progress("wrote records to %s", outDir)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&outDir, "out", ".", "output directory for JSONL files")
	f.BoolVar(&withComments, "comments", false, "also crawl comments")
	f.BoolVar(&withDanmaku, "danmaku", false, "also crawl danmaku")
	f.BoolVar(&withRelated, "related", true, "also crawl related videos")
	f.BoolVar(&withUser, "user", true, "also crawl the uploader profile")
	return cmd
}

type jsonlWriter struct {
	f   *os.File
	enc *json.Encoder
	err error
}

func newJSONLWriter(path string) *jsonlWriter {
	f, err := os.Create(path)
	if err != nil {
		return &jsonlWriter{err: err}
	}
	return &jsonlWriter{f: f, enc: json.NewEncoder(f)}
}

func (w *jsonlWriter) Write(v any) {
	if w.err != nil || w.enc == nil {
		return
	}
	w.err = w.enc.Encode(v)
}

func (w *jsonlWriter) Close() error {
	if w.f != nil {
		return w.f.Close()
	}
	return w.err
}
