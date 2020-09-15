package main

import (
	"io"
	stdlog "log"
	"time"

	"github.com/wvh/urn/internal/version"

	log "github.com/go-kit/kit/log"
)

func isDev(env string) bool {
	return env == "development" || env == "dev" || env == "local"
}

// makeLogger creates a logger that writes in logfmt format to a given io.Writer stream.
// If running in a development environment as determined by the isDev function above,
// it will use a shorter time-only date format and add log caller information.
func makeLogger(out io.Writer, env string) log.Logger {
	timeFormat := log.DefaultTimestampUTC
	if isDev(env) {
		timeFormat = log.TimestampFormat(time.Now, "15:04:05.000000")
	}

	logger := log.NewLogfmtLogger(log.NewSyncWriter(out))
	logger = log.With(logger, "service", version.Id+"web")
	logger = log.With(logger, "time", timeFormat)

	/*
	wrapper := logger
	if isDev(env) {
		wrapper = log.With(logger, "caller", log.Caller(6))
		logger = log.With(logger, "caller", log.DefaultCaller)
	}
	*/
	stdwrapper := logAdapter{logger}

	// add location information in dev output
	if isDev(env) {
		logger = log.With(logger, "caller", log.DefaultCaller)
		stdwrapper.logger = log.With(stdwrapper.logger, "caller", log.Caller(6))
	}

	//stdlog.SetOutput(log.NewStdlibAdapter(logger, log.TimestampKey("time")))
	stdlog.SetFlags(0)
	stdlog.SetPrefix("")
	stdlog.SetOutput(&stdwrapper)

	return logger
}

func sublogger(logger log.Logger, name string) log.Logger {
	return log.With(logger, "component", name)
}

type logAdapter struct {
	logger log.Logger
}

func (la *logAdapter) Write(p []byte) (n int, err error) {
	// stdlib logger always adds a newline; strip it
	la.logger.Log("msg", string(p[:len(p)-1]))
	return len(p), nil
}

func newStdlogAdapter(logger log.Logger) *stdlog.Logger {
	//logger = log.With(logger, "caller", log.Caller(6))
	return stdlog.New(&logAdapter{logger}, "", 0)
}
