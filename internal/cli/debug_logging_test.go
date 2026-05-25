package cli

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
)

func TestMemHandler(t *testing.T) {
	buf := &bytes.Buffer{}
	var parentRecorded bool
	parent := &mockHandler{
		enabled: func(l slog.Level) bool { return l >= slog.LevelInfo },
		handle:  func(r slog.Record) error { parentRecorded = true; return nil },
	}

	h := newMemHandler(buf, parent)

	t.Run("Enabled", func(t *testing.T) {
		if !h.Enabled(context.Background(), slog.LevelDebug) {
			t.Error("expected Enabled to always return true")
		}
	})

	t.Run("Handle_CapturedAndDelegated", func(t *testing.T) {
		buf.Reset()
		parentRecorded = false
		r := slog.Record{Time: time.Now(), Level: slog.LevelInfo, Message: "info msg"}
		err := h.Handle(context.Background(), r)
		if err != nil {
			t.Errorf("Handle failed: %v", err)
		}

		if !strings.Contains(buf.String(), "INFO: info msg") {
			t.Errorf("expected buffer to contain log, got: %s", buf.String())
		}
		if !parentRecorded {
			t.Error("expected parent to be called for INFO level")
		}
	})

	t.Run("Handle_CapturedNotDelegated", func(t *testing.T) {
		buf.Reset()
		parentRecorded = false
		r := slog.Record{Time: time.Now(), Level: slog.LevelDebug, Message: "debug msg"}
		err := h.Handle(context.Background(), r)
		if err != nil {
			t.Errorf("Handle failed: %v", err)
		}

		if !strings.Contains(buf.String(), "DEBUG: debug msg") {
			t.Errorf("expected buffer to contain log, got: %s", buf.String())
		}
		if parentRecorded {
			t.Error("expected parent NOT to be called for DEBUG level")
		}
	})

	t.Run("Handle_WithAttrs", func(t *testing.T) {
		buf.Reset()
		r := slog.Record{Time: time.Now(), Level: slog.LevelInfo, Message: "attr msg"}
		r.AddAttrs(slog.String("foo", "bar"), slog.Int("baz", 42))
		err := h.Handle(context.Background(), r)
		if err != nil {
			t.Errorf("Handle failed: %v", err)
		}

		if !strings.Contains(buf.String(), "foo=bar") || !strings.Contains(buf.String(), "baz=42") {
			t.Errorf("expected buffer to contain attributes, got: %s", buf.String())
		}
	})

	t.Run("WithAttrs", func(t *testing.T) {
		h2 := h.WithAttrs([]slog.Attr{slog.String("key", "val")})
		if h2 == nil {
			t.Fatal("expected WithAttrs to return a handler")
		}
		mh2, ok := h2.(*memHandler)
		if !ok {
			t.Fatal("expected returned handler to be *memHandler")
		}
		if mh2.buf != h.buf {
			t.Error("expected shared buffer")
		}
		if mh2.mu != h.mu {
			t.Error("expected shared mutex pointer")
		}
	})

	t.Run("WithGroup", func(t *testing.T) {
		h2 := h.WithGroup("group")
		if h2 == nil {
			t.Fatal("expected WithGroup to return a handler")
		}
		mh2, ok := h2.(*memHandler)
		if !ok {
			t.Fatal("expected returned handler to be *memHandler")
		}
		if mh2.buf != h.buf {
			t.Error("expected shared buffer")
		}
		if mh2.mu != h.mu {
			t.Error("expected shared mutex pointer")
		}
	})
}

type mockHandler struct {
	enabled func(slog.Level) bool
	handle  func(slog.Record) error
}

func (m *mockHandler) Enabled(_ context.Context, l slog.Level) bool  { return m.enabled(l) }
func (m *mockHandler) Handle(_ context.Context, r slog.Record) error { return m.handle(r) }
func (m *mockHandler) WithAttrs(_ []slog.Attr) slog.Handler          { return m }
func (m *mockHandler) WithGroup(_ string) slog.Handler               { return m }

func TestSetupDebugLogging(t *testing.T) {
	// We need to be careful with global state here.
	// We'll restore it at the end.
	originalDefault := slog.Default()
	defer slog.SetDefault(originalDefault)

	t.Run("NoError", func(t *testing.T) {
		dump := setupDebugLogging()
		if slog.Default().Handler() == originalDefault.Handler() {
			t.Error("expected default logger handler to be changed")
		}

		dump(nil)
	})

	t.Run("WithError", func(t *testing.T) {
		// Instead of os.Pipe, we'll just check if the file is created.
		// To avoid cluttering stderr during tests, we can temporarily redirect it to a dummy file
		// but since we want to avoid hangs, we'll just not capture it and trust the return.

		dump := setupDebugLogging()
		log.Info().Msg("test log message")

		// Create a temp file to capture stderr if we really want to, but let's try WITHOUT it first
		// to see if the hang was indeed os.Pipe

		err := errors.New("test error")

		// To avoid printing to real stderr during tests, we can swap it with a simple file
		f, _ := os.CreateTemp("", "stderr-capture-*.log")
		defer func() { _ = os.Remove(f.Name()) }()
		oldStderr := os.Stderr
		os.Stderr = f

		dump(err)

		os.Stderr = oldStderr
		_ = f.Close()

		// Read the captured stderr
		captured, _ := os.ReadFile(f.Name())
		output := string(captured)

		if !strings.Contains(output, "An error occurred during setup. Detailed debug logs have been saved to:") {
			t.Errorf("expected stderr to contain dump message, got: %s", output)
		}

		// Find the filename in the output
		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) > 0 {
			lastLine := lines[len(lines)-1]
			// The output might have a newline at the end, so let's be careful
			if _, statErr := os.Stat(lastLine); statErr == nil {
				// Verify file content
				content, _ := os.ReadFile(lastLine)
				if !strings.Contains(string(content), "test log message") {
					t.Errorf("expected log file to contain message, got: %s", string(content))
				}
				_ = os.Remove(lastLine)
			} else {
				// Maybe it's on a different line?
				found := false
				for _, line := range lines {
					if _, statErr := os.Stat(line); statErr == nil {
						content, _ := os.ReadFile(line)
						if strings.Contains(string(content), "test log message") {
							found = true
							_ = os.Remove(line)
							break
						}
					}
				}
				if !found {
					t.Errorf("expected log file to be created and contain message, but couldn't find it in output: %s", output)
				}
			}
		}
	})
}
