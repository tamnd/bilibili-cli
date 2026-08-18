package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tamnd/bilibili-cli/bili"
)

// newVerifyCmd builds `bili verify`, which re-measures the requirement matrix
// against the live API.
//
// This is not a test of this tool. It is a test of the assumptions this tool
// was built on, which live on somebody else's servers and change without
// notice. The offline suite can prove that the client signs what the matrix
// says it signs; only a live run can tell you the matrix is still true.
//
// It is deliberately not part of `go test`. A test suite that reaches the
// network fails on a train, and a contributor should never have to be online to
// run the gate.
func newVerifyCmd(a *App) *cobra.Command {
	var live, strict bool

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Re-measure the endpoint requirement matrix against the live API",
		Long: "Requests every endpoint in the requirement matrix and reports the rows " +
			"that no longer behave the way they were recorded. Requires --live, because " +
			"it is the only command in this tool that exists to make requests rather " +
			"than to answer a question.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !live {
				return fmt.Errorf("verify makes live requests and needs --live to say so")
			}

			rows := bili.Matrix()

			// Progress goes to stderr so that -o json on stdout stays machine
			// readable while a run that can take twenty minutes is happening.
			progress := func(o bili.Observation) {
				if !a.quiet {
					fmt.Fprintf(os.Stderr, "%-44s %s\n", o.Name, describe(o))
				}
			}
			if !a.quiet {
				fmt.Fprintf(os.Stderr, "%d rows. Any that refuse are asked again after %s.\n\n",
					len(rows), bili.ProbeBackoff)
			}

			out, err := a.Client().VerifyMatrix(a.ctx(), rows, progress)
			if err != nil {
				return err
			}

			moved, unmeasured := 0, 0
			for _, o := range out {
				switch {
				case o.Moved:
					moved++
				case o.Unmeasured:
					unmeasured++
				}
			}

			if err := emitAll(a, out); err != nil {
				return err
			}

			// The two counts are reported separately and only the first one is
			// an error. A row the site rate limited twice was not measured, and
			// saying that it moved would be a claim this run cannot support.
			if unmeasured > 0 {
				fmt.Fprintf(os.Stderr, "\n%d of %d rows were rate limited on both passes and not measured\n",
					unmeasured, len(rows))
			}
			if moved > 0 {
				fmt.Fprintf(os.Stderr, "\n%d of %d rows moved\n", moved, len(rows))
				if strict {
					return fmt.Errorf("%d rows no longer match the recorded matrix", moved)
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&live, "live", false, "actually make the requests")
	cmd.Flags().BoolVar(&strict, "strict", false, "exit non-zero when a row moved")
	return cmd
}

func describe(o bili.Observation) string {
	s := o.Bare
	if o.WithSignature != "" {
		s += " / signed " + o.WithSignature
	}
	if o.Retried {
		s += " (second pass)"
	}
	if o.Reason != "" {
		s += "  " + o.Reason
	}
	return s
}
