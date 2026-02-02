package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Dir            string
	RetentionHours int
	Prefix         string
}

type HourlyFileWriter struct {
	mu          sync.Mutex
	dir         string
	retention   int
	prefix      string
	currentHour time.Time
	file        *os.File
}

func NewHourlyFileWriter(cfg Config) (*HourlyFileWriter, error) {
	if cfg.Dir == "" {
		cfg.Dir = "logs"
	}
	if cfg.Prefix == "" {
		cfg.Prefix = "api"
	}
	if cfg.RetentionHours <= 0 {
		cfg.RetentionHours = 24
	}

	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		return nil, err
	}

	writer := &HourlyFileWriter{
		dir:       cfg.Dir,
		retention: cfg.RetentionHours,
		prefix:    cfg.Prefix,
	}
	if err := writer.rotateIfNeeded(time.Now().UTC()); err != nil {
		return nil, err
	}
	return writer, nil
}

func (w *HourlyFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.rotateIfNeeded(time.Now().UTC()); err != nil {
		return 0, err
	}
	return w.file.Write(p)
}

func (w *HourlyFileWriter) rotateIfNeeded(now time.Time) error {
	hour := now.Truncate(time.Hour)
	if w.file != nil && w.currentHour.Equal(hour) {
		return nil
	}

	if w.file != nil {
		_ = w.file.Close()
	}

	filename := fmt.Sprintf("%s-%s.log", w.prefix, hour.Format("20060102-15"))
	path := filepath.Join(w.dir, filename)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	w.file = file
	w.currentHour = hour
	w.cleanupOldFiles(now)
	return nil
}

func (w *HourlyFileWriter) cleanupOldFiles(now time.Time) {
	if w.retention <= 0 {
		return
	}
	cutoff := now.Add(-time.Duration(w.retention) * time.Hour)
	pattern := filepath.Join(w.dir, w.prefix+"-*.log")
	matches, _ := filepath.Glob(pattern)
	for _, path := range matches {
		base := filepath.Base(path)
		ts := strings.TrimSuffix(strings.TrimPrefix(base, w.prefix+"-"), ".log")
		t, err := time.Parse("20060102-15", ts)
		if err != nil {
			continue
		}
		if t.Before(cutoff) {
			_ = os.Remove(path)
		}
	}
}
