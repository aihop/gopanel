package api

import (
	"bufio"
	"encoding/json"
	"fmt"
)

func writeSystemDiagnosticSSE(writer *bufio.Writer, event string, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if _, err = fmt.Fprintf(writer, "event: %s\ndata: %s\n\n", event, payload); err != nil {
		return err
	}
	return writer.Flush()
}
