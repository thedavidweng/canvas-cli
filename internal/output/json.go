package output

import (
	"encoding/json"
	"io"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
)

// WriteJSON serializes an Envelope to the writer as JSON.
// When pretty is true the output is indented; otherwise it is compact.
func WriteJSON(w io.Writer, envelope canvas.Envelope, pretty bool) error {
	enc := json.NewEncoder(w)
	if pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(envelope)
}
