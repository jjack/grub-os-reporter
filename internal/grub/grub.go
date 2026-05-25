package grub

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/jjack/grubstation/internal/assets"
	"github.com/rs/zerolog/log"
)

var ErrConfigNotFound = errors.New("no grub config found in known locations")

var knownConfigPaths = []string{
	"/boot/grub/grub.cfg",
	"/boot/grub2/grub.cfg",
	"/boot/efi/EFI/fedora/grub.cfg",
	"/boot/efi/EFI/redhat/grub.cfg",
	"/boot/efi/EFI/ubuntu/grub.cfg",
}

// Grub represents the GRUB bootloader on this system.
type Grub struct {
	ConfigPath          string
	HassGrubStationPath string
	LookPath            func(file string) (string, error)
	Command             func(ctx context.Context, name string, arg ...string) *exec.Cmd
	GetTemplate         func(name string) (*template.Template, error)
}

func NewGrub() *Grub {
	return &Grub{
		HassGrubStationPath: "/etc/grub.d/99_grubstation",
		LookPath:            exec.LookPath,
		Command:             exec.CommandContext,
		GetTemplate:         assets.GetTemplate,
	}
}

type SetupOptions struct {
	TargetMAC       string
	TargetURL       string
	AuthToken       string
	WaitTimeSeconds int
}

// GetBootOptions parses the GRUB configuration and returns available boot options.
func (g *Grub) GetBootOptions(ctx context.Context) ([]string, error) {
	log.Debug().Msg("Parsing GRUB boot options...")

	var grubPath string
	var err error

	if g.ConfigPath != "" {
		grubPath = g.ConfigPath
		log.Debug().Str("path", grubPath).Msg("Using explicit GRUB config path")
	} else {
		grubPath, err = findConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to locate grub config: %w", err)
		}
	}

	file, err := os.Open(grubPath)
	if err != nil {
		if os.IsPermission(err) {
			return nil, fmt.Errorf("permission denied reading grub config %s (are you running as root?)", grubPath)
		}
		return nil, fmt.Errorf("failed to open grub config %s: %w", grubPath, err)
	}
	defer func() { _ = file.Close() }()

	return parseMenuEntries(file), nil
}

// DiscoverConfigPath attempts to auto-detect the GRUB config file path.
func (g *Grub) DiscoverConfigPath(ctx context.Context) (string, error) {
	return findConfig()
}

func findConfig() (string, error) {
	for _, path := range knownConfigPaths {
		if _, err := os.Stat(path); err == nil {
			log.Debug().Str("path", path).Msg("Found GRUB config")
			return path, nil
		}
	}
	return "", ErrConfigNotFound
}

// submenuContext keeps track of where we are in the nested tree
type submenuContext struct {
	depth int
	title string
}

// parseMenuEntries takes an io.Reader and returns a flat list of GRUB boot targets.
func parseMenuEntries(r io.Reader) []string {
	scanner := bufio.NewScanner(r)
	var entries []string
	var stack []submenuContext
	braceDepth := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 1. Process the line if it's a menu declaration
		if strings.HasPrefix(line, "submenu ") {
			title := extractTitle(line)
			if title != "" {
				// Record the depth *before* we enter the block
				stack = append(stack, submenuContext{depth: braceDepth, title: title})
			}
		} else if strings.HasPrefix(line, "menuentry ") {
			title := extractTitle(line)
			if title != "" {
				entries = append(entries, buildTargetString(stack, title))
			}
		}

		// 2. Update our brace depth state
		// We do this after checking the prefixes so the submenu's own opening brace
		// increases the depth to a level *deeper* than the submenu's recorded depth.
		braceDepth += strings.Count(line, "{")
		braceDepth -= strings.Count(line, "}")

		// 3. Pop the stack if we've exited a submenu block
		// If our depth drops to or below the depth where the current submenu was declared,
		// it means we have closed that submenu's bracket.
		for len(stack) > 0 && braceDepth <= stack[len(stack)-1].depth {
			stack = stack[:len(stack)-1]
		}
	}

	return entries
}

// extractTitle finds the first string wrapped in single or double quotes
func extractTitle(line string) string {
	var quoteChar rune
	start := -1

	for i, char := range line {
		if (char == '\'' || char == '"') && start == -1 {
			quoteChar = char
			start = i + 1
		} else if char == quoteChar && start != -1 {
			return line[start:i]
		}
	}
	return ""
}

// buildTargetString joins the current submenu stack with the final entry name
func buildTargetString(stack []submenuContext, entryTitle string) string {
	if len(stack) == 0 {
		return entryTitle
	}

	var parts []string
	for _, s := range stack {
		parts = append(parts, s.title)
	}
	parts = append(parts, entryTitle)

	return strings.Join(parts, ">")
}
