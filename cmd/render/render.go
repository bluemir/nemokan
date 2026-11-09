package render

import (
	"os"

	"github.com/alecthomas/kingpin/v2"
	"github.com/cockroachdb/errors"

	"github.com/bluemir/nemokan/internal/action"
	"github.com/bluemir/nemokan/internal/shape"
	"github.com/bluemir/nemokan/internal/state"
)

// Register adds the command drawing a .mono file as text, the same as "Export as text" of the
// editor.
func Register(cmd *kingpin.CmdClause) {
	var (
		filePath   string
		outputPath string
	)

	cmd.Arg("file", ".mono file path").
		Required().
		StringVar(&filePath)
	cmd.Flag("output", "output file path (default: stdout)").
		Short('o').
		StringVar(&outputPath)

	cmd.Action(func(*kingpin.ParseContext) error {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return errors.Wrap(err, "failed to read file")
		}

		file, err := shape.ParseFile(data)
		if err != nil {
			return errors.Wrap(err, "failed to parse file")
		}

		editor := state.New(shape.NewManager(), action.NewManager(), func(state.Change) {})
		editor.OpenFile(file)
		text := editor.ExportText()

		if outputPath == "" {
			_, err := os.Stdout.WriteString(text + "\n")
			return err
		}

		if err := os.WriteFile(outputPath, []byte(text+"\n"), 0o644); err != nil { // #nosec G306 - a rendered text for the user
			return errors.Wrap(err, "failed to write output")
		}
		return nil
	})
}
