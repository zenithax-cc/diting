package logger

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type LogFormat string

const (
	LogFormatText LogFormat = "text"
	LogFormatJSON LogFormat = "json"
)

type LogConfig struct {
	Dir            string
	FilenamePrefix string
	Format         LogFormat
	RetainDays     int
	Level          slog.Level
	AddSource      bool
}

var (
	loggerOnce sync.Once
	onceLogger *slog.Logger
)

func InitLogger(cfg *LogConfig) (*slog.Logger, error) {
	var initErr error
	loggerOnce.Do(func() {
		if cfg.Dir == "" {
			cfg.Dir = "/var/log/diting"
		}
		if cfg.FilenamePrefix == "" {
			cfg.FilenamePrefix = "diting"
		}
		if cfg.RetainDays <= 0 {
			cfg.RetainDays = 365
		}

		if err := os.MkdirAll(cfg.Dir, 0755); err != nil {
			initErr = fmt.Errorf("create log directory failed: %w", err)
			return
		}

		termHandler := NewColorHandler(os.Stderr, &slog.HandlerOptions{
			Level:     cfg.Level,
			AddSource: cfg.AddSource,
		})

		fileHandler, err := NewFileHandler(cfg)
		if err != nil {
			initErr = err
			return
		}

		multi := slog.New(NewMultiHandler(termHandler, fileHandler))
		onceLogger = multi
		slog.SetDefault(onceLogger)
	})

	if initErr != nil {
		return nil, initErr
	}

	return onceLogger, nil
}

func GetLogger() *slog.Logger {
	if onceLogger != nil {
		return onceLogger
	}

	return slog.Default()
}

type MultiHandler struct {
	handlers []slog.Handler
}

func NewMultiHandler(handlers ...slog.Handler) slog.Handler {
	return &MultiHandler{handlers: handlers}
}

func (mh *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range mh.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}

	return false
}

func (mh *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	var handlerErr error

	for _, h := range mh.handlers {
		if !h.Enabled(ctx, r.Level) {
			continue
		}

		if err := h.Handle(ctx, r); err != nil && handlerErr == nil {
			handlerErr = err
		}
	}

	return handlerErr
}

func (mh *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(mh.handlers))
	for i, h := range mh.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}

	return &MultiHandler{handlers: handlers}
}

func (mh *MultiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(mh.handlers))
	for i, h := range mh.handlers {
		handlers[i] = h.WithGroup(name)
	}

	return &MultiHandler{handlers: handlers}
}

type DailyFileHandler struct {
	mu       sync.Mutex
	cfg      *LogConfig
	curDate  string
	curFile  *os.File
	curInner slog.Handler
}

func NewFileHandler(cfg *LogConfig) (*DailyFileHandler, error) {
	handler := &DailyFileHandler{cfg: cfg}
	if err := handler.rotateIfNeeded(time.Now()); err != nil {
		return nil, err
	}

	return handler, nil
}

func (h *DailyFileHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.cfg.Level <= level
}

func (h *DailyFileHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if err := h.rotateIfNeeded(r.Time); err != nil {
		return err
	}

	if h.curInner == nil {
		return fmt.Errorf("file handler not initialized")
	}

	return h.curInner.Handle(ctx, r)
}

func (h *DailyFileHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()

	var newInner slog.Handler
	if h.curInner != nil {
		newInner = h.curInner.WithAttrs(attrs)
	}

	return &DailyFileHandler{
		cfg:      h.cfg,
		curDate:  h.curDate,
		curFile:  h.curFile,
		curInner: newInner,
	}
}

func (h *DailyFileHandler) WithGroup(name string) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()

	var newInner slog.Handler
	if h.curInner != nil {
		newInner = h.curInner.WithGroup(name)
	}

	return &DailyFileHandler{
		cfg:      h.cfg,
		curDate:  h.curDate,
		curFile:  h.curFile,
		curInner: newInner,
	}
}

func (h *DailyFileHandler) rotateIfNeeded(t time.Time) error {
	date := t.Format("2006-01-02")
	if date == h.curDate && h.curInner != nil {
		return nil
	}

	if h.curFile != nil {
		_ = h.curFile.Close()
		h.curFile = nil
		h.curInner = nil
	}

	filename := fmt.Sprintf("%s-%s.log", h.cfg.FilenamePrefix, date)
	path := filepath.Join(h.cfg.Dir, filename)

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open log file failed: %w", err)
	}

	opts := &slog.HandlerOptions{
		Level:     h.cfg.Level,
		AddSource: h.cfg.AddSource,
	}

	var inner slog.Handler
	switch h.cfg.Format {
	case LogFormatJSON:
		inner = slog.NewJSONHandler(file, opts)
	default:
		inner = slog.NewTextHandler(file, opts)
	}

	h.curDate = date
	h.curFile = file
	h.curInner = inner

	go h.cleanOldLogs()

	return nil
}

func (h *DailyFileHandler) cleanOldLogs() {
	if h.cfg.RetainDays <= 0 {
		return
	}

	entries, err := os.ReadDir(h.cfg.Dir)
	if err != nil {
		return
	}

	now := time.Now()
	prefix := h.cfg.FilenamePrefix + "-"
	suffix := ".log"

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if len(name) <= len(prefix)+len(suffix) || name[:len(prefix)] != prefix || name[len(name)-len(suffix):] != suffix {
			continue
		}

		dateStr := name[len(prefix) : len(name)-len(suffix)]
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		if now.Sub(date) > time.Duration(h.cfg.RetainDays)*24*time.Hour {
			_ = os.Remove(filepath.Join(h.cfg.Dir, name))
		}
	}
}

type ColorHandler struct {
	out   *os.File
	inner slog.Handler
	opts  *slog.HandlerOptions
}

func NewColorHandler(out *os.File, opts *slog.HandlerOptions) slog.Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}

	inner := slog.NewTextHandler(out, opts)
	return &ColorHandler{
		out:   out,
		inner: inner,
		opts:  opts,
	}
}

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
)

func (h *ColorHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *ColorHandler) Handle(ctx context.Context, r slog.Record) error {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, h.opts)
	if err := handler.Handle(ctx, r); err != nil {
		return err
	}

	var color string
	switch {
	case r.Level >= slog.LevelError:
		color = colorRed
	case r.Level == slog.LevelWarn:
		color = colorYellow
	case r.Level == slog.LevelInfo:
		color = ""
	default:
		color = colorGreen
	}

	if color != "" {
		_, _ = h.out.Write([]byte(color))
	}
	_, err := h.out.Write(buf.Bytes())
	if color != "" {
		_, _ = h.out.Write([]byte(colorReset))
	}

	return err
}

func (h *ColorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ColorHandler{
		out:   h.out,
		inner: h.inner.WithAttrs(attrs),
		opts:  h.opts,
	}
}

func (h *ColorHandler) WithGroup(name string) slog.Handler {
	return &ColorHandler{
		out:   h.out,
		inner: h.inner.WithGroup(name),
		opts:  h.opts,
	}
}

