package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"
)

const (
	timeFormat   = "[15:04:05.000]"
	reset        = "\033[0m"
	black        = 30
	red          = 31
	green        = 32
	yellow       = 33
	blue         = 34
	magenta      = 35
	cyan         = 36
	lightGray    = 37
	darkGray     = 90
	lightRed     = 91
	lightGreen   = 92
	lightYellow  = 93
	lightBlue    = 94
	lightMagenta = 95
	lightCyan    = 96
	white        = 97
)

func colorizer(colorCode int, v string) string {
	return fmt.Sprintf("\033[%sm%s%s", strconv.Itoa(colorCode), v, reset)
}

// Handler is a slog.Handler implementation that outputs human-readable,
// colorized log messages for development use. It wraps the standard
// slog.JSONHandler and transforms its output into a pretty format.
type Handler struct {
	handler slog.Handler
	// Per-handler configuration
	writer io.Writer
	// Shared state across WithAttrs/WithGroup instances for output synchronization.
	// This ensures log lines from related handlers don't get interleaved.
	buffer   *bytes.Buffer
	mutex    *sync.Mutex
	groups   []string
	colorize bool
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		handler:  h.handler.WithAttrs(attrs),
		buffer:   h.buffer,
		mutex:    h.mutex,
		writer:   h.writer,
		colorize: h.colorize,
		groups:   h.groups,
	}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name
	return &Handler{
		handler:  h.handler.WithGroup(name),
		buffer:   h.buffer,
		mutex:    h.mutex,
		writer:   h.writer,
		colorize: h.colorize,
		groups:   newGroups,
	}
}

func (h *Handler) computeAttrs(ctx context.Context, r slog.Record) (map[string]any, error) {
	h.mutex.Lock()
	defer func() {
		h.buffer.Reset()
		h.mutex.Unlock()
	}()
	if err := h.handler.Handle(ctx, r); err != nil {
		return nil, fmt.Errorf("error when calling inner handler's Handle: %w", err)
	}
	var attrs map[string]any
	err := json.Unmarshal(h.buffer.Bytes(), &attrs)
	if err != nil {
		return nil, fmt.Errorf("error when unmarshaling inner handler's Handle result: %w", err)
	}
	return attrs, nil
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	colorize := func(code int, value string) string {
		return value
	}
	if h.colorize {
		colorize = colorizer
	}
	level := r.Level.String() + ":"
	switch {
	case r.Level <= slog.LevelDebug:
		level = colorize(magenta, level)
	case r.Level <= slog.LevelInfo:
		level = colorize(green, level)
	case r.Level < slog.LevelWarn:
		level = colorize(lightBlue, level)
	case r.Level < slog.LevelError:
		level = colorize(yellow, level)
	case r.Level == slog.LevelError:
		level = colorize(lightRed, level)
	default:
		level = colorize(lightMagenta, level)
	}
	timestamp := colorize(lightGray, r.Time.Format(timeFormat))
	msg := colorize(cyan, r.Message)
	// Add group prefix to message when groups exist
	var groupPrefix string
	if len(h.groups) > 0 {
		groupPrefix = colorize(magenta, "["+strings.Join(h.groups, ".")+"]")
	}
	attrs, err := h.computeAttrs(ctx, r)
	if err != nil {
		return err
	}
	var attrsAsBytes []byte
	if len(attrs) > 0 {
		attrsAsBytes, err = json.MarshalIndent(attrs, "", "  ")
		if err != nil {
			return fmt.Errorf("error when marshaling attrs: %w", err)
		}
	}
	var parts []string
	if len(timestamp) > 0 {
		parts = append(parts, timestamp)
	}
	if len(level) > 0 {
		parts = append(parts, level)
	}
	if len(groupPrefix) > 0 {
		parts = append(parts, groupPrefix)
	}
	if len(msg) > 0 {
		parts = append(parts, msg)
	}
	if len(attrsAsBytes) > 0 {
		parts = append(parts, colorize(darkGray, string(attrsAsBytes)))
	}
	out := strings.Join(parts, " ")
	if h.writer != nil {
		_, err = io.WriteString(h.writer, out+"\n")
		if err != nil {
			return fmt.Errorf("write log line: %w", err)
		}
	}
	return nil
}

// suppressDefaults drops the attrs Handle renders itself, leaving the inner
// JSON handler to emit only the caller's own attributes.
func suppressDefaults(_ []string, a slog.Attr) slog.Attr {
	switch a.Key {
	case slog.TimeKey, slog.LevelKey, slog.MessageKey:
		return slog.Attr{}
	}
	return a
}

// NewHandler builds the development handler: slog's JSON handler renders the
// record into a buffer, which Handle then reformats into a colorized line.
func NewHandler(writer io.Writer, level slog.Leveler, colorize bool) *Handler {
	buffer := &bytes.Buffer{}
	return &Handler{
		buffer:   buffer,
		writer:   writer,
		colorize: colorize,
		handler: slog.NewJSONHandler(buffer, &slog.HandlerOptions{
			Level:       level,
			ReplaceAttr: suppressDefaults,
		}),
		mutex: &sync.Mutex{},
	}
}
