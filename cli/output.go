package cli

import (
	"io"

	"github.com/tamnd/any-cli/kit/render"
)

// Format is an output encoding. Aliased from the render package so the rest of
// the CLI compiles without changes.
type Format = render.Format

const (
	FormatAuto     Format = render.Auto
	FormatTable    Format = render.Table
	FormatList     Format = render.List
	FormatMarkdown Format = render.Markdown
	FormatJSON     Format = render.JSON
	FormatJSONL    Format = render.JSONL
	FormatCSV      Format = render.CSV
	FormatTSV      Format = render.TSV
	FormatURL      Format = render.URL
	FormatRaw      Format = render.Raw
)

// Output wraps render.Renderer so commands call Emit/Close without changes.
type Output struct {
	r        *render.Renderer
	suppress bool
}

// NewOutput builds an Output. isTTY and color control ANSI emission; width is
// the terminal column count (0 = no limit).
func NewOutput(w io.Writer, format Format, fields []string, noHeader bool, tmpl string, isTTY, color bool, width int) (*Output, error) {
	r, err := render.New(render.Options{
		Format:   format,
		Writer:   w,
		IsTTY:    isTTY,
		Color:    color,
		Fields:   fields,
		NoHeader: noHeader,
		Template: tmpl,
		Width:    width,
	})
	if err != nil {
		return nil, err
	}
	return &Output{r: r}, nil
}

func (o *Output) Emit(v any) error {
	if o.suppress {
		return nil
	}
	return o.r.Emit(v)
}

func (o *Output) Close() error {
	if o.suppress {
		return nil
	}
	return o.r.Flush()
}

// Format returns the resolved format (after Auto is collapsed).
func (o *Output) Format() Format { return o.r.Format() }

// Write passes raw bytes through to the underlying writer, bypassing formatting.
func (o *Output) Write(b []byte) (int, error) {
	if o.suppress {
		return len(b), nil
	}
	return o.r.Write(b)
}
