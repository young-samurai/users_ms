package lgr

// Package logger
/* Система логирования*/

import (
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strings"

	cfg "users/pkg/config"

	"github.com/sirupsen/logrus"
)

/// Обертка для логгера, позволяет использовать несколько мест и несколько уровней детализации для логов
/// stdOut, stdErr, files, kafka, elasticsearch, ...

// / Структура хука для logrus
type writerHook struct {
	Writer    []io.Writer    // Слайс райтеров
	LogLevels []logrus.Level // Слайс уровней логирования
}

func (hook *writerHook) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		return err
	}
	for _, w := range hook.Writer {
		_, _ = w.Write([]byte(line))
	}
	return err
}

func (hook *writerHook) Levels() []logrus.Level {
	return hook.LogLevels
}

var e *logrus.Entry

// Logger / Создаем свою структуру
// чтобы в случае смены импортного логгера не менять его по всему коду
// поменять только тут
type Logger struct {
	*logrus.Entry
}

var LOG *Logger

func GetLogger(cfg *cfg.Config) *Logger {

	l := logrus.New()                    // создаем логгер
	l.SetReportCaller(true)              // разрешаем ему писать
	l.Formatter = &logrus.TextFormatter{ // задаем формат записей
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			fileName := path.Base(frame.File)
			return fmt.Sprintf("%s()", frame.Function), fmt.Sprintf("%s:%d", fileName, frame.Line)
		},
		DisableColors:   true,
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	}

	// Парсим output
	var w []io.Writer
	outputs := strings.Split(cfg.Log.LogOutput, "|")
	for _, out := range outputs {
		switch strings.ToLower(out) {
		case "file":
			if cfg.Log.LogPath != "" {
				f, err := os.OpenFile(cfg.Log.LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				if err == nil {
					w = append(w, f)
				}
			}
		case "stdout":
			w = append(w, os.Stdout)
		}
	}

	// Запрещаем стандартное логирование, чтобы использовать хук
	l.SetOutput(io.Discard)

	// Добавляем хук
	l.AddHook(&writerHook{
		Writer:    w,
		LogLevels: logrus.AllLevels,
	})

	// Задаем уровни логирования
	switch cfg.Log.LogLevel {
	case "panic":
		l.SetLevel(logrus.PanicLevel)
	case "fatal":
		l.SetLevel(logrus.FatalLevel)
	case "error":
		l.SetLevel(logrus.ErrorLevel)
	case "warn":
		l.SetLevel(logrus.WarnLevel)
	case "info":
		l.SetLevel(logrus.InfoLevel)
	case "debug":
		l.SetLevel(logrus.DebugLevel)
	case "trace":
		l.SetLevel(logrus.TraceLevel)
	}

	e = logrus.NewEntry(l)

	LOG = &Logger{e}
	return &Logger{e}
}

func (l *Logger) GetLoggerWithField(k string, v interface{}) *Logger {
	return &Logger{l.WithField(k, v)}
}

/// Конец обертки