package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tamnd/bilibili-cli/bili"
)

// bili download is a wrapper and says so.
//
// BBDown already solves the part of this that is hard: the DASH manifest, the
// stream selection, the multi part handling and the merge. A second Go
// implementation of that would be a second thing to maintain that does the same
// job worse, and it would rot on the same schedule as the first one. So this
// command does the parts BBDown does not: it resolves whatever the user pasted
// through the resolver every other command uses, hands BBDown a bvid, and
// converts the result when the format asked for is not the one BBDown emits.
//
// It downloads audio. It is not a general media downloader and the flag surface
// says so, so there is nothing here that decrypts anything.

// downloadResult is one file that ended up on disk.
type downloadResult struct {
	Path   string `json:"path"`
	Format string `json:"format"`
	Bytes  int64  `json:"bytes"`
}

type downloadOptions struct {
	Format           string
	OutputDir        string
	Parts            string
	Quality          string
	BBDownBin        string
	FFmpegBin        string
	FilePattern      string
	MultiFilePattern string
}

// bbdown and ffmpeg are the two executables this command needs and does not
// ship. Everything needed to report a missing one usefully is here rather than
// spread across the call sites, because "exec: BBDown: executable file not
// found in $PATH" is a true statement that helps nobody.
var (
	bbdown = externalTool{
		name:    "BBDown",
		flag:    "--bbdown-bin",
		env:     "BILI_BBDOWN_BIN",
		why:     "bili download delegates the transfer to BBDown",
		install: "https://github.com/nilaoda/BBDown/releases",
	}
	ffmpeg = externalTool{
		name:    "ffmpeg",
		flag:    "--ffmpeg-bin",
		env:     "BILI_FFMPEG_BIN",
		why:     "mp3, flac and wav are transcoded after the download",
		install: "https://ffmpeg.org/download.html",
		escape:  "--format m4a is what BBDown emits natively and needs no ffmpeg at all",
	}
)

// externalTool is a program bili runs and does not bundle.
type externalTool struct {
	name    string // the executable, as it is called on PATH
	flag    string // the flag that points at a copy somewhere else
	env     string // the environment variable that does the same
	why     string // what this command wants it for
	install string // where to get it
	escape  string // how to not need it, when there is a way
}

func newDownloadCmd(a *App) *cobra.Command {
	opt := downloadOptions{
		Format:           "m4a",
		Quality:          "best",
		OutputDir:        ".",
		FilePattern:      "<videoTitle>",
		MultiFilePattern: "<videoTitle>/<pageNumberWithZero> <pageTitle>",
	}
	cmd := &cobra.Command{
		Use:   "download <bvid|url>",
		Short: "Download audio with BBDown",
		Long: "Download audio from a bilibili video by wrapping BBDown.\n" +
			"m4a is what BBDown emits and involves no second process. mp3, flac and\n" +
			"wav are transcoded with ffmpeg once the download finishes.\n" +
			"Both executables are found on PATH and neither is bundled.",
		Args: cobra.ExactArgs(1),
		Example: "  bili download BV1DG411a7Lt\n" +
			"  bili download https://www.bilibili.com/video/BV1DG411a7Lt --parts 1-3 --format mp3\n" +
			"  bili download BV1DG411a7Lt --output-dir ~/Music/audiobooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := runDownload(a, args[0], opt)
			if err != nil {
				return err
			}
			return emitAll(a, res)
		},
	}
	f := cmd.Flags()
	f.StringVar(&opt.Format, "format", opt.Format, "audio format: m4a|mp3|flac|wav")
	f.StringVar(&opt.OutputDir, "output-dir", opt.OutputDir, "directory to save downloaded files")
	f.StringVar(&opt.Parts, "parts", "", "part selection, e.g. 1,3-5,LAST")
	f.StringVar(&opt.Quality, "quality", opt.Quality, "mp3 bitrate preset: best|high|medium|low|worst (mp3 only)")
	f.StringVar(&opt.BBDownBin, "bbdown-bin", "", "BBDown executable path")
	f.StringVar(&opt.FFmpegBin, "ffmpeg-bin", "", "ffmpeg executable path")
	f.StringVar(&opt.FilePattern, "file-pattern", opt.FilePattern, "BBDown single-part file pattern")
	f.StringVar(&opt.MultiFilePattern, "multi-file-pattern", opt.MultiFilePattern, "BBDown multi-part file pattern")
	return cmd
}

// runDownload does everything that can fail cheaply before it does anything
// that takes time.
//
// The order matters. Resolving ffmpeg after the download would mean a missing
// ffmpeg is discovered at the end of a forty minute audiobook, and the fix for
// it is a one line install that was available before the first byte moved.
func runDownload(a *App, source string, opt downloadOptions) ([]downloadResult, error) {
	format := strings.ToLower(strings.TrimSpace(opt.Format))
	if !isSupportedDownloadFormat(format) {
		return nil, fmt.Errorf("unsupported format %q (want m4a, mp3, flac or wav)", opt.Format)
	}

	target, err := downloadTarget(a, source)
	if err != nil {
		return nil, err
	}

	bbdownBin, err := resolveExecutable(bbdown, opt.BBDownBin)
	if err != nil {
		return nil, err
	}
	ffmpegBin := ""
	if format != "m4a" {
		if ffmpegBin, err = resolveExecutable(ffmpeg, opt.FFmpegBin); err != nil {
			return nil, err
		}
	}

	outputDir, err := expandDir(opt.OutputDir)
	if err != nil {
		return nil, err
	}

	// BBDown writes into a staging directory and the finished files are moved
	// into place from there, so a run that fails halfway leaves nothing behind
	// in the directory the user pointed at.
	stageDir, err := os.MkdirTemp("", "bili-download-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(stageDir) }()

	args := buildBBDownArgs(target, stageDir, opt, a.resolveCookie(), a.userAgentOverride())
	if a.dryRun {
		a.progress("would run %s", quoteCommand(bbdownBin, args))
		if ffmpegBin != "" {
			a.progress("would then transcode each file to %s with %s", format, ffmpegBin)
		}
		return nil, nil
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, err
	}
	a.progress("running %s", quoteCommand(bbdownBin, args))
	if err := runExternal(a, bbdownBin, args...); err != nil {
		return nil, fmt.Errorf("%s: %w", bbdown.name, err)
	}

	media, err := collectMediaFiles(stageDir)
	if err != nil {
		return nil, err
	}
	if len(media) == 0 {
		// BBDown exited 0 and produced nothing, which is its way of reporting
		// a part selection that matched no parts, among other things.
		return nil, errors.New("BBDown finished without producing an audio file, so there was nothing to save")
	}

	var out []downloadResult
	for _, src := range media {
		rel, err := filepath.Rel(stageDir, src)
		if err != nil {
			return nil, err
		}
		dst := filepath.Join(outputDir, replaceExt(rel, "."+format))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return nil, err
		}
		if format == "m4a" {
			if err := moveFile(src, dst); err != nil {
				return nil, err
			}
		} else {
			ffArgs := ffmpegArgs(src, dst, format, opt.Quality)
			a.progress("running %s", quoteCommand(ffmpegBin, ffArgs))
			if err := runExternal(a, ffmpegBin, ffArgs...); err != nil {
				return nil, fmt.Errorf("%s: %w", ffmpeg.name, err)
			}
		}
		info, err := os.Stat(dst)
		if err != nil {
			return nil, err
		}
		out = append(out, downloadResult{Path: dst, Format: format, Bytes: info.Size()})
	}
	return out, nil
}

// downloadTarget turns whatever was pasted into the bvid BBDown wants.
//
// Doing it here rather than handing the raw string over means a bare av number
// and a b23.tv short link work exactly as they do everywhere else in this tool,
// and it means an id for something that is not a video is refused by name
// instead of becoming a BBDown error about a page that does not exist.
func downloadTarget(a *App, source string) (string, error) {
	id, err := a.Client().Resolve(a.ctx(), source)
	if err != nil {
		return "", err
	}
	if id.Kind != bili.KindVideo || id.BVID == "" {
		return "", fmt.Errorf("%s is a %s and bili download reads videos", source, id.Kind)
	}
	return id.BVID, nil
}

func buildBBDownArgs(bvid, stageDir string, opt downloadOptions, cookie, userAgent string) []string {
	args := []string{
		bvid,
		"--audio-only",
		"--skip-cover",
		"--skip-subtitle",
		"--work-dir", stageDir,
		"-F", opt.FilePattern,
		"-M", opt.MultiFilePattern,
	}
	if opt.Parts != "" {
		args = append(args, "-p", opt.Parts)
	}
	if cookie != "" {
		args = append(args, "-c", cookie)
	}
	// Only an explicit override is passed on. With no override BBDown uses its
	// own user agent, which is the one its endpoints were tested against, and
	// substituting bili's default here would be this tool guessing on behalf of
	// a program that already knows the answer.
	if userAgent != "" {
		args = append(args, "--user-agent", userAgent)
	}
	return args
}

// userAgentOverride is what the user asked for, or empty when they asked for
// nothing. It is deliberately not effectiveUserAgent, which fills in a default.
func (a *App) userAgentOverride() string {
	return firstNonEmpty(a.userAgent, os.Getenv("BILI_USER_AGENT"))
}

func runExternal(a *App, name string, args ...string) error {
	cmd := exec.CommandContext(a.ctx(), name, args...)
	cmd.Stdin = os.Stdin
	// BBDown draws a progress bar and ffmpeg reports on its own, and both go to
	// the terminal as they were written. Parsing either of them into a progress
	// display of bili's own would be a second thing to keep in step with two
	// upstreams, for a worse version of what they already print.
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func collectMediaFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".m4a", ".mp3", ".flac", ".wav":
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// ffmpegArgs is the whole transcode decision in one pure function, which is
// what makes it testable without an ffmpeg on the machine running the tests.
func ffmpegArgs(src, dst, format, quality string) []string {
	args := []string{"-hide_banner", "-loglevel", "warning", "-y", "-i", src, "-vn", "-codec:a"}
	switch format {
	case "mp3":
		// -q:a is a variable bitrate scale, which is where --quality lands.
		args = append(args, "libmp3lame", "-q:a", mp3Quality(quality))
	case "flac":
		args = append(args, "flac")
	case "wav":
		args = append(args, "pcm_s16le")
	}
	return append(args, dst)
}

// mp3Quality maps the preset onto libmp3lame's VBR scale, where 0 is the best
// and 9 is the worst. It is mp3 only: flac and wav are lossless, so there is
// nothing for a quality preset to mean, and --quality is ignored for both.
func mp3Quality(q string) string {
	switch strings.ToLower(strings.TrimSpace(q)) {
	case "best":
		return "0"
	case "high":
		return "2"
	case "medium":
		return "4"
	case "low":
		return "6"
	case "worst":
		return "9"
	default:
		return "2"
	}
}

func isSupportedDownloadFormat(s string) bool {
	switch s {
	case "m4a", "mp3", "flac", "wav":
		return true
	default:
		return false
	}
}

// resolveExecutable finds the tool, or explains what is missing.
//
// An explicit path that is not there and nothing installed at all are different
// mistakes with different fixes, so they get different messages. Neither of
// them is "executable file not found in $PATH".
func resolveExecutable(tool externalTool, flagValue string) (string, error) {
	if chosen := firstNonEmpty(flagValue, os.Getenv(tool.env)); chosen != "" {
		path, err := exec.LookPath(chosen)
		if err != nil {
			return "", fmt.Errorf("%s was pointed at %q and there is nothing runnable there", tool.name, chosen)
		}
		return path, nil
	}
	path, err := exec.LookPath(tool.name)
	if err == nil {
		return path, nil
	}
	msg := fmt.Sprintf("%s is not on PATH. %s, and it is not bundled: install it from %s, or point %s at a copy (%s does the same)",
		tool.name, tool.why, tool.install, tool.flag, tool.env)
	if tool.escape != "" {
		msg += ". " + tool.escape
	}
	return "", errors.New(msg)
}

func expandDir(dir string) (string, error) {
	if dir == "" {
		dir = "."
	}
	if strings.HasPrefix(dir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, dir[2:])
	}
	return filepath.Abs(dir)
}

// moveFile renames, and copies when the rename crosses a filesystem, which it
// does whenever the temporary directory and the output directory are on
// different mounts.
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := out.ReadFrom(in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Remove(src)
}

func replaceExt(path, ext string) string {
	return strings.TrimSuffix(path, filepath.Ext(path)) + ext
}

// quoteCommand renders a command line that can be pasted back into a shell,
// which is the only reason to print one at all.
func quoteCommand(name string, args []string) string {
	quoted := make([]string, 0, len(args)+1)
	quoted = append(quoted, shellQuote(name))
	for _, arg := range args {
		quoted = append(quoted, shellQuote(arg))
	}
	return strings.Join(quoted, " ")
}

// shellQuote quotes anything that is not obviously safe, rather than looking
// for the characters that are dangerous. BBDown's file patterns are the reason:
// <videoTitle>/<pageTitle> contains no quote and no space and is still two
// redirections and a filename to a shell.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case strings.ContainsRune("@%+=:,./-_", r):
		default:
			return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
		}
	}
	return s
}
