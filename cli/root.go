package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/x/term"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/tamnd/bilibili-cli/bili"
)

// build-time vars (injected via -ldflags)
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// App holds global state shared by every command.
type App struct {
	// global flags
	output     string
	fields     string
	noHeader   bool
	template   string
	limit      int
	page       int
	pageSize   int
	order      string
	cookie     string
	cookieFile string
	rate       time.Duration
	retries    int
	timeout    time.Duration
	workers    int
	cache      bool
	noCache    bool
	cacheTTL   time.Duration
	lang       string
	quiet      bool
	verbose    int
	color      string
	proxy      string
	userAgent  string
	raw        bool
	dryRun     bool
	yes        bool

	client  *bili.Client
	rootCtx context.Context
}

// Root builds the root command tree.
func Root() *cobra.Command {
	app := &App{}
	root := &cobra.Command{
		Use:   "bili",
		Short: "A delightful command line for Bilibili",
		Long: "bili turns bilibili.com into a fast, scriptable command line: resolve any\n" +
			"video, user, comment, danmaku, dynamic, live room, bangumi, audio, article,\n" +
			"or favorite into clean structured records you can pipe anywhere.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       Version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			app.rootCtx = cmd.Context()
		},
	}

	pf := root.PersistentFlags()
	pf.StringVarP(&app.output, "output", "o", "auto", "list|table|markdown|json|jsonl|csv|tsv|url|raw")
	pf.StringVar(&app.fields, "fields", "", "comma-separated columns to keep/order")
	pf.BoolVar(&app.noHeader, "no-header", false, "omit the header row")
	pf.StringVar(&app.template, "template", "", "Go text/template applied per record")
	pf.IntVarP(&app.limit, "limit", "n", 0, "max records emitted (0 = unlimited)")
	pf.IntVar(&app.page, "page", 0, "start page where the endpoint paginates")
	pf.IntVar(&app.pageSize, "page-size", 0, "page size (endpoint-dependent default)")
	pf.StringVar(&app.order, "order", "", "sort order where supported")
	pf.StringVar(&app.cookie, "cookie", "", "cookie header (SESSDATA=...; ...)")
	pf.StringVar(&app.cookieFile, "cookie-file", "", "path to a cookie file")
	pf.DurationVar(&app.rate, "rate", 350*time.Millisecond, "min delay between requests")
	pf.IntVar(&app.retries, "retries", 4, "retry attempts on 429/-412/5xx")
	pf.DurationVar(&app.timeout, "timeout", 30*time.Second, "per-request timeout")
	pf.IntVarP(&app.workers, "workers", "j", 4, "concurrency for fan-out commands")
	pf.BoolVar(&app.cache, "cache", true, "use the on-disk response cache")
	pf.BoolVar(&app.noCache, "no-cache", false, "bypass the on-disk cache")
	pf.DurationVar(&app.cacheTTL, "cache-ttl", time.Hour, "cache freshness window")
	pf.StringVar(&app.lang, "lang", "zh-CN", "locale for localized fields")
	pf.BoolVarP(&app.quiet, "quiet", "q", false, "suppress progress on stderr")
	pf.CountVarP(&app.verbose, "verbose", "v", "increase verbosity (repeatable)")
	pf.StringVar(&app.color, "color", "auto", "auto|always|never")
	pf.StringVar(&app.proxy, "proxy", "", "HTTP/SOCKS proxy URL")
	pf.StringVar(&app.userAgent, "user-agent", "", "override the default desktop UA")
	pf.BoolVar(&app.raw, "raw", false, "print each record as pretty-printed JSON")
	pf.BoolVar(&app.dryRun, "dry-run", false, "print the requests that would be made")
	pf.BoolVarP(&app.yes, "yes", "y", false, "assume yes to prompts")

	root.AddCommand(
		newVideoCmd(app),
		newUserCmd(app),
		newSearchCmd(app),
		newCommentsCmd(app),
		newDanmakuCmd(app),
		newDynamicCmd(app),
		newDynamicsCmd(app),
		newLiveCmd(app),
		newBangumiCmd(app),
		newAudioCmd(app),
		newArticleCmd(app),
		newFavoriteCmd(app),
		newFavoritesCmd(app),
		newPopularCmd(app),
		newRankCmd(app),
		newTrendingCmd(app),
		newRelatedCmd(app),
		newSuggestCmd(app),
		newStreamsCmd(app),
		newIDCmd(app),
		newNavCmd(app),
		newDiscoverCmd(app),
		newCrawlCmd(app),
		newConfigCmd(app),
		newCacheCmd(app),
		newVersionCmd(app),
	)
	return root
}

// Client lazily builds the bili client from resolved config.
func (a *App) Client() *bili.Client {
	if a.client != nil {
		return a.client
	}
	cfg := bili.DefaultConfig()
	cfg.Rate = a.rate
	cfg.Retries = a.retries
	cfg.Timeout = a.timeout
	cfg.CacheTTL = a.cacheTTL
	cfg.NoCache = a.noCache || !a.cache
	cfg.DryRun = a.dryRun
	cfg.Lang = a.lang
	cfg.Proxy = firstNonEmpty(a.proxy, os.Getenv("BILI_PROXY"))
	cfg.UserAgent = firstNonEmpty(a.userAgent, os.Getenv("BILI_USER_AGENT"))
	if cfg.UserAgent == "" {
		cfg.UserAgent = bili.DefaultUserAgent
	}
	cfg.Cookie = a.resolveCookie()
	a.client = bili.NewClient(cfg)
	return a.client
}

func (a *App) resolveCookie() string {
	if a.cookie != "" {
		return a.cookie
	}
	if c := os.Getenv("BILI_COOKIE"); c != "" {
		return c
	}
	file := firstNonEmpty(a.cookieFile, os.Getenv("BILI_COOKIE_FILE"))
	if file != "" {
		if b, err := os.ReadFile(file); err == nil {
			return parseCookieFile(string(b))
		}
	}
	return ""
}

func parseCookieFile(s string) string {
	var parts []string
	for line := range strings.SplitSeq(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Netscape cookie format has 7 tab-separated fields
		if f := strings.Split(line, "\t"); len(f) == 7 {
			parts = append(parts, f[5]+"="+f[6])
			continue
		}
		parts = append(parts, line)
	}
	return strings.Join(parts, "; ")
}

// resolveFormat resolves "auto" to the right format, respecting BILI_OUTPUT.
func (a *App) resolveFormat() Format {
	f := a.output
	if v := os.Getenv("BILI_OUTPUT"); v != "" && f == "auto" {
		f = v
	}
	if a.raw {
		return FormatRaw
	}
	return Format(f)
}

// resolveColor returns true when ANSI color should be emitted.
func (a *App) resolveColor() bool {
	switch a.color {
	case "always":
		return true
	case "never":
		return false
	default:
		return isatty.IsTerminal(os.Stdout.Fd())
	}
}

// newOutput builds the configured Output writer.
func (a *App) newOutput() (*Output, error) {
	var fields []string
	if a.fields != "" {
		fields = strings.Split(a.fields, ",")
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}
	}
	isTTY := isatty.IsTerminal(os.Stdout.Fd())
	out, err := NewOutput(os.Stdout, a.resolveFormat(), fields, a.noHeader, a.template, isTTY, a.resolveColor(), termWidth())
	if err != nil {
		return nil, err
	}
	out.suppress = a.dryRun
	return out, nil
}

// termWidth reports the terminal column count, or 0 when stdout is not a
// terminal. The renderer uses it to shrink a too-wide table; 0 means no limit.
func termWidth() int {
	if v := os.Getenv("COLUMNS"); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	if w, _, err := term.GetSize(os.Stdout.Fd()); err == nil && w > 0 {
		return w
	}
	return 0
}

func (a *App) progress(format string, args ...any) {
	if a.quiet {
		return
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func (a *App) ctx() context.Context {
	if a.rootCtx != nil {
		return a.rootCtx
	}
	return context.Background()
}

// readArgsOrStdin returns args, or lines from stdin when the single arg is "-".
func readArgsOrStdin(args []string) []string {
	if len(args) == 1 && args[0] == "-" {
		var out []string
		sc := bufio.NewScanner(os.Stdin)
		sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line != "" {
				out = append(out, line)
			}
		}
		return out
	}
	return args
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}
