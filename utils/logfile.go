package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type DailyFileLogger struct {
	mu         sync.Mutex
	dir        string
	base       string
	ext        string
	loc        *time.Location
	curDate    string
	file       *os.File
	stopCh     chan struct{}
	doneCh     chan struct{}
	alsoStdout bool
	stdout     io.Writer
	dateFmt    string // по умолчанию 2006-01-02
	bindStd    bool
	stdPrefix  string
	stdFlags   int
}

// Опции
type LogOption func(*DailyFileLogger)

// С локальным временем (по умолчанию — UTC)
func WithLocalTime() LogOption {
	return func(l *DailyFileLogger) { l.loc = time.Local }
}

// С UTC временем (по умолчанию уже UTC)
func WithUTC() LogOption {
	return func(l *DailyFileLogger) { l.loc = time.UTC }
}

// Дублировать вывод в stdout
func WithAlsoStdout() LogOption {
	return func(l *DailyFileLogger) { l.alsoStdout = true; l.stdout = os.Stdout }
}

// Кастомный формат даты (по умолчанию "2006-01-02")
func WithDateFormat(layout string) LogOption {
	return func(l *DailyFileLogger) { l.dateFmt = layout }
}

// NewDailyFileLogger создает ротацию по датам. path может быть:
// - "logAPI.log" → "logAPI-YYYY-MM-DD.log"
// - "logAPI"     → "logAPI-YYYY-MM-DD.log"
// - "logs/api.log" → "logs/api-YYYY-MM-DD.log"
func NewDailyFileLogger(path string, opts ...LogOption) (*DailyFileLogger, error) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if name == "" {
		name = "app"
	}
	if ext == "" {
		ext = ".log"
	}
	if dir == "." {
		dir = "" // текущая папка
	}

	l := &DailyFileLogger{
		dir:     dir,
		base:    name,
		ext:     ext,
		loc:     time.UTC,
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
		stdout:  io.Discard,
		dateFmt: "2006-01-02",
	}
	for _, opt := range opts {
		opt(l)
	}
	if l.alsoStdout && l.stdout == nil {
		l.stdout = os.Stdout
	}

	if err := l.rotateIfNeededLocked(time.Now().In(l.loc)); err != nil {
		return nil, err
	}
	go l.rotationLoop()

	return l, nil
}

func WithBindStdLogger(prefix string, flags int) LogOption {
	return func(l *DailyFileLogger) {
		// пометим в объекте, что надо привязать stdlog
		l.bindStd = true
		l.stdPrefix = prefix
		l.stdFlags = flags
	}
}

func InitAndBindStdLogger(path, prefix string, flags int, opts ...LogOption) (*DailyFileLogger, error) {
	opts = append(opts, WithBindStdLogger(prefix, flags))
	return InitGlobalLogger(path, "", opts...)
}

// InitGlobalLogger упрощает подключение логгера к стандартному log.
// Пример использования в main:
//
//	lf, err := utils.InitGlobalLogger("logAPI.log", "[api] ", utils.WithUTC(), utils.WithAlsoStdout())
//	if err != nil { log.Panic(err) }
//	defer lf.Close()
//	log.SetFlags(log.LstdFlags)
func InitGlobalLogger(path, unusedPrefix string, opts ...LogOption) (*DailyFileLogger, error) {
	lf, err := NewDailyFileLogger(path, opts...)
	if err != nil {
		return nil, err
	}
	if lf.bindStd {
		log.SetOutput(lf)
		if lf.stdPrefix != "" {
			log.SetPrefix(lf.stdPrefix)
		}
		if lf.stdFlags != 0 {
			log.SetFlags(lf.stdFlags)
		}
	}
	return lf, nil
}

func (l *DailyFileLogger) fileNameForDate(d string) string {
	filename := fmt.Sprintf("%s-%s%s", l.base, d, l.ext)
	if l.dir != "" {
		return filepath.Join(l.dir, filename)
	}
	return filename
}

func (l *DailyFileLogger) rotateIfNeededLocked(now time.Time) error {
	dateStr := now.Format(l.dateFmt)
	if l.file != nil && dateStr == l.curDate {
		return nil
	}

	// закрыть старый файл
	if l.file != nil {
		_ = l.file.Close()
		l.file = nil
	}

	// создать директорию при необходимости
	if l.dir != "" {
		if err := os.MkdirAll(l.dir, 0o755); err != nil {
			return fmt.Errorf("create log dir: %w", err)
		}
	}

	// открыть новый файл
	full := l.fileNameForDate(dateStr)
	f, err := os.OpenFile(full, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	l.file = f
	l.curDate = dateStr
	return nil
}

func (l *DailyFileLogger) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		// safety: попытаться открыть
		_ = l.rotateIfNeededLocked(time.Now().In(l.loc))
		if l.file == nil {
			return 0, fmt.Errorf("logger not initialized")
		}
	}
	// проверяем необходимость ротации (если запись после смены даты)
	_ = l.rotateIfNeededLocked(time.Now().In(l.loc))

	// запись в файл
	n, err := l.file.Write(p)
	// дублирование в stdout, если включено
	if l.alsoStdout {
		_, _ = l.stdout.Write(p)
	}
	return n, err
}

func (l *DailyFileLogger) rotationLoop() {
	defer close(l.doneCh)
	for {
		select {
		case <-l.stopCh:
			return
		case <-time.After(l.durationUntilNextMidnight()):
			l.mu.Lock()
			_ = l.rotateIfNeededLocked(time.Now().In(l.loc))
			l.mu.Unlock()
		}
	}
}

func (l *DailyFileLogger) durationUntilNextMidnight() time.Duration {
	now := time.Now().In(l.loc)
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, l.loc)
	return time.Until(next)
}

func (l *DailyFileLogger) Close() error {
	close(l.stopCh)
	<-l.doneCh

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		err := l.file.Close()
		l.file = nil
		return err
	}
	return nil
}
