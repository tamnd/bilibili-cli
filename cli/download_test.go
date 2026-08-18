package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// Nothing in this file runs BBDown or ffmpeg, and nothing in it needs either
// one installed. The command is a wrapper, so what is worth testing is the
// wrapping: the arguments it builds, the identifiers it accepts, the formats it
// admits, and what it says when a dependency is not there.

func TestBuildBBDownArgs(t *testing.T) {
	opt := downloadOptions{
		Parts:            "1-3",
		FilePattern:      "<videoTitle>",
		MultiFilePattern: "<videoTitle>/<pageNumberWithZero> <pageTitle>",
	}
	got := buildBBDownArgs("BV1DG411a7Lt", "/tmp/stage", opt, "SESSDATA=abc", "ua/1.0")
	want := []string{
		"BV1DG411a7Lt",
		"--audio-only",
		"--skip-cover",
		"--skip-subtitle",
		"--work-dir", "/tmp/stage",
		"-F", "<videoTitle>",
		"-M", "<videoTitle>/<pageNumberWithZero> <pageTitle>",
		"-p", "1-3",
		"-c", "SESSDATA=abc",
		"--user-agent", "ua/1.0",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildBBDownArgs() mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

// No cookie and no user agent override means neither flag is passed at all,
// rather than passed empty. BBDown reads an empty -c as a cookie.
func TestBBDownGetsNoFlagsItWasNotGiven(t *testing.T) {
	got := buildBBDownArgs("BV1DG411a7Lt", "/tmp/stage", downloadOptions{}, "", "")
	for _, flag := range []string{"-c", "--user-agent", "-p"} {
		if slices.Contains(got, flag) {
			t.Errorf("%s was passed with nothing to put in it: %v", flag, got)
		}
	}
}

func TestSupportedDownloadFormats(t *testing.T) {
	for _, tc := range []struct {
		format string
		want   bool
	}{
		{"m4a", true},
		{"mp3", true},
		{"flac", true},
		{"wav", true},
		{"video", false},
	} {
		if got := isSupportedDownloadFormat(tc.format); got != tc.want {
			t.Fatalf("isSupportedDownloadFormat(%q) = %v, want %v", tc.format, got, tc.want)
		}
	}
}

func TestReplaceExt(t *testing.T) {
	got := replaceExt("Book 3/01 Owl Post.m4a", ".mp3")
	want := "Book 3/01 Owl Post.mp3"
	if got != want {
		t.Fatalf("replaceExt() = %q, want %q", got, want)
	}
}

// A URL, a bare av number and a bvid are three ways to write down the same
// video, and BBDown is handed the same bvid for all three. This is the reason
// the identifier is resolved here rather than passed through.
func TestEveryWayOfWritingAVideoReachesTheSameBvid(t *testing.T) {
	a := &App{}
	for _, in := range []string{
		"BV1DG411a7Lt",
		"https://www.bilibili.com/video/BV1DG411a7Lt",
		"https://www.bilibili.com/video/BV1DG411a7Lt?p=2&spm_id_from=333.999",
	} {
		got, err := downloadTarget(a, in)
		if err != nil {
			t.Fatalf("downloadTarget(%q): %v", in, err)
		}
		if got != "BV1DG411a7Lt" {
			t.Errorf("downloadTarget(%q) = %q, want BV1DG411a7Lt", in, got)
		}
	}

	// An av number is the same video written the other way, and the conversion
	// is offline arithmetic.
	bv, err := downloadTarget(a, "av2")
	if err != nil {
		t.Fatalf("downloadTarget(av2): %v", err)
	}
	if !strings.HasPrefix(bv, "BV") {
		t.Errorf("downloadTarget(av2) = %q, want a bvid", bv)
	}
}

// An id for something that is not a video is refused here, by name. Handing it
// to BBDown would produce a complaint about a page that does not exist, which
// sends the reader looking at the network for a mistake that is in the argument.
func TestAnIdentifierThatIsNotAVideoIsRefusedByName(t *testing.T) {
	a := &App{}
	for id, kind := range map[string]string{
		"ss12345": "bangumi",
		"au1":     "audio",
		"cv1":     "article",
	} {
		_, err := downloadTarget(a, id)
		if err == nil {
			t.Fatalf("downloadTarget(%q) succeeded, and it is not a video", id)
		}
		if !strings.Contains(err.Error(), kind) {
			t.Errorf("downloadTarget(%q) said %q, which does not say it is a %s", id, err, kind)
		}
	}
}

// A missing dependency has to name the binary, say what it is wanted for, and
// say where to get it. "exec: BBDown: executable file not found in $PATH" is a
// true statement that leaves the reader to work all three out.
func TestAMissingDependencyNamesItselfAndTheWayOut(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	_, err := resolveExecutable(bbdown, "")
	if err == nil {
		t.Fatal("BBDown resolved from an empty PATH")
	}
	for _, want := range []string{"BBDown", "--bbdown-bin", "BILI_BBDOWN_BIN", "github.com/nilaoda/BBDown"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the message does not mention %q: %s", want, err)
		}
	}

	// ffmpeg has a way out that BBDown does not: a format that does not need
	// it. The message says so, because the fastest fix is often not to install
	// anything.
	_, err = resolveExecutable(ffmpeg, "")
	if err == nil {
		t.Fatal("ffmpeg resolved from an empty PATH")
	}
	if !strings.Contains(err.Error(), "--format m4a") {
		t.Errorf("the ffmpeg message does not offer the format that needs no ffmpeg: %s", err)
	}
}

// Pointing --bbdown-bin at nothing and having nothing installed are different
// mistakes with different fixes, so they do not share a message.
func TestAnExplicitPathThatIsEmptySaysSo(t *testing.T) {
	_, err := resolveExecutable(bbdown, filepath.Join(t.TempDir(), "nope"))
	if err == nil {
		t.Fatal("a path that does not exist resolved anyway")
	}
	if !strings.Contains(err.Error(), "nothing runnable there") {
		t.Errorf("an explicit bad path got the not-installed message: %s", err)
	}
}

// A tool the user pointed at is used even when a copy is on PATH, since being
// specific is the only reason to pass the flag.
func TestAnExplicitPathWins(t *testing.T) {
	dir := t.TempDir()
	mine := filepath.Join(dir, "BBDown")
	if err := os.WriteFile(mine, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := resolveExecutable(bbdown, mine)
	if err != nil {
		t.Fatalf("resolveExecutable: %v", err)
	}
	if got != mine {
		t.Errorf("resolveExecutable() = %q, want %q", got, mine)
	}
}

func TestFFmpegArgsPerFormat(t *testing.T) {
	for _, tc := range []struct {
		format  string
		quality string
		want    []string
	}{
		{"mp3", "best", []string{"libmp3lame", "-q:a", "0"}},
		{"mp3", "low", []string{"libmp3lame", "-q:a", "6"}},
		{"mp3", "", []string{"libmp3lame", "-q:a", "2"}},
		{"flac", "best", []string{"flac"}},
		{"wav", "worst", []string{"pcm_s16le"}},
	} {
		got := ffmpegArgs("/stage/in.m4a", "/out/out."+tc.format, tc.format, tc.quality)
		if got[len(got)-1] != "/out/out."+tc.format {
			t.Errorf("%s: the output path is not last: %v", tc.format, got)
		}
		codec := got[slices.Index(got, "-codec:a")+1 : len(got)-1]
		if !reflect.DeepEqual(codec, tc.want) {
			t.Errorf("ffmpegArgs(%s, %q) codec args = %v, want %v", tc.format, tc.quality, codec, tc.want)
		}
	}
}

// --quality is an mp3 setting. flac and wav are lossless, so there is nothing
// for a preset to mean, and asking for a worse one changes nothing.
func TestQualityIsIgnoredForTheLosslessFormats(t *testing.T) {
	for _, format := range []string{"flac", "wav"} {
		best := ffmpegArgs("/in.m4a", "/out."+format, format, "best")
		worst := ffmpegArgs("/in.m4a", "/out."+format, format, "worst")
		if !reflect.DeepEqual(best, worst) {
			t.Errorf("%s: --quality changed the command: %v vs %v", format, best, worst)
		}
	}
}

// A dry run prints the command and runs nothing, which is what the flag
// promises everywhere else in this tool. It also means the whole path up to the
// first exec is exercised offline, including the dependency lookups.
func TestADryRunPrintsTheCommandAndRunsNothing(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "BBDown")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "music")

	a := &App{dryRun: true, quiet: true}
	res, err := runDownload(a, "BV1DG411a7Lt", downloadOptions{
		Format: "m4a", OutputDir: out, BBDownBin: fake, FilePattern: "<videoTitle>",
	})
	if err != nil {
		t.Fatalf("runDownload: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("a dry run produced %d results, and it downloaded nothing", len(res))
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("a dry run created the output directory %s", out)
	}
}

// An unsupported format fails before anything is looked up or created, so a
// typo costs nothing and leaves nothing behind.
func TestAnUnsupportedFormatFailsFirst(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "music")
	a := &App{quiet: true}
	if _, err := runDownload(a, "BV1DG411a7Lt", downloadOptions{Format: "ogg", OutputDir: out}); err == nil {
		t.Fatal("--format ogg was accepted")
	} else if !strings.Contains(err.Error(), "m4a") {
		t.Errorf("the message does not say what is supported: %s", err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("a rejected format created the output directory %s", out)
	}
}

func TestQuoteCommandCanBePastedBack(t *testing.T) {
	got := quoteCommand("/usr/bin/BBDown", []string{"BV1", "-M", "<videoTitle>/<pageTitle>", "-c", ""})
	want := `/usr/bin/BBDown BV1 -M '<videoTitle>/<pageTitle>' -c ''`
	if got != want {
		t.Errorf("quoteCommand() = %s, want %s", got, want)
	}
}

// A title in Chinese is the common case, and quoting every non-ASCII rune
// would make every line of progress output unreadable for no safety gained.
func TestAQuotedCommandDoesNotMangleChinese(t *testing.T) {
	got := shellQuote("影视飓风")
	if got != "'影视飓风'" {
		t.Errorf("shellQuote() = %s", got)
	}
}
