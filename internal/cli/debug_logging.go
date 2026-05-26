package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// debugWriter is an io.Writer that captures logs in a buffer and optionally writes to another writer.
type debugWriter struct {
	mu     sync.Mutex
	buf    *bytes.Buffer
	parent io.Writer
}

func (w *debugWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 1. Capture in memory buffer
	_, _ = w.buf.Write(p)

	// 2. Write to parent (usually console)
	return w.parent.Write(p)
}

// setupDebugLogging sets up an in-memory logger and returns a function to dump logs on error.
func setupDebugLogging() (dumpFunc func(err error)) {
	var buf bytes.Buffer
	originalLogger := log.Logger

	// Zerolog doesn't provide a clean way to extract the current writer from a Logger instance
	// without using internal/unstable APIs. Since we mostly control the global logger in this app,
	// we'll assume os.Stderr as the fallback parent writer.
	parent := io.Writer(os.Stderr)

	dw := &debugWriter{
		buf:    &buf,
		parent: parent,
	}

	// Create a new logger that writes to our debugWriter
	// We want to capture DEBUG logs even if the parent was set to INFO.
	// So we set the global level to Debug for the duration of the capture.
	originalLevel := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	log.Logger = zerolog.New(dw).With().Timestamp().Logger()

	return func(err error) {
		// Restore original logger and level
		log.Logger = originalLogger
		zerolog.SetGlobalLevel(originalLevel)

		if err == nil {
			return
		}

		// If there was an error, dump the buffer to a temp file
		tmpFile, createErr := os.CreateTemp("", "grubstation-setup-*.log")
		if createErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "\nFailed to create debug log file: %v\n", createErr)
			return
		}
		defer func() { _ = tmpFile.Close() }()

		if _, writeErr := tmpFile.Write(buf.Bytes()); writeErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "\nFailed to write to debug log file: %v\n", writeErr)
			return
		}

		_, _ = fmt.Fprintf(os.Stderr, "\nAn error occurred during setup. Detailed debug logs have been saved to:\n%s\n", tmpFile.Name())
	}
}
