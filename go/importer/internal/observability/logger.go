package observability

import (
	"fmt"
	"log/slog"
	"os"
)

type Logger struct {
	logger *slog.Logger
}

func NewLogger(component string) Logger {
	return Logger{logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})).With("component", component)}
}

func (l Logger) Infof(format string, args ...any) {
	l.logger.Info("info", "message", fmt.Sprintf(format, args...))
}

func (l Logger) Errorf(format string, args ...any) {
	l.logger.Error("error", "message", fmt.Sprintf(format, args...))
}
