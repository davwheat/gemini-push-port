package logging

import (
	"fmt"
	"gemini-push-port/generic"
	"net/http"
	"os"
	"strings"
	"time"

	"context"

	"cloud.google.com/go/logging"

	glogging "github.com/TV4/logrus-stackdriver-formatter"
	"github.com/getsentry/sentry-go"
	"github.com/sirupsen/logrus"
)

var (
	Logger    LogInterface
	gcpLogger *logging.Logger
	gcpClient *logging.Client
)

// LogInterface differs from the usual method signatures for Error and Fatal, where the default methods require an error
// to be passed in. This prevents it being forgotten as can often happen

type LogInterface interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})

	Info(args ...interface{})
	Infof(format string, args ...interface{})

	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	WarnE(prefix string, err error)

	Error(err error, args ...interface{})
	Errorf(err error, format string, args ...interface{})
	ErrorE(prefix string, err error)

	// ErrorMsg is logrus's Error method, for the occasion an error log has no error associated
	ErrorMsg(args ...interface{})
	// ErrorMsgf is logrus's Errorf method, for the occasion an error log has no error associated
	ErrorMsgf(format string, args ...interface{})

	Fatal(err error, args ...interface{})
	Fatalf(err error, format string, args ...interface{})
	FatalE(prefix string, err error)

	WithField(key string, value interface{}) LogInterface
	WithFields(fields logrus.Fields) LogInterface

	CloneForID(id string, req *http.Request) LogInterface
	CloneForThread(thread string) LogInterface
}

// LogrusLogger is a standard logger which outputs to the console
type LogrusLogger logrus.Entry

func (l *LogrusLogger) Debug(args ...interface{}) {
	l.toLogrusEntry().Debug(args...)
}

func (l *LogrusLogger) Debugf(format string, args ...interface{}) {
	l.toLogrusEntry().Debugf(format, args...)
}

func (l *LogrusLogger) Info(args ...interface{}) {
	l.toLogrusEntry().Info(args...)
}

func (l *LogrusLogger) Infof(format string, args ...interface{}) {
	l.toLogrusEntry().Infof(format, args...)
}

func (l *LogrusLogger) Warn(args ...interface{}) {
	l.toLogrusEntry().Warn(args...)
}

func (l *LogrusLogger) Warnf(format string, args ...interface{}) {
	l.toLogrusEntry().Warnf(format, args...)
}

func (l *LogrusLogger) toLogrusEntry() *logrus.Entry {
	return (*logrus.Entry)(l)
}

func (l *LogrusLogger) WarnE(prefix string, err error) {
	l.toLogrusEntry().WithError(err).Warnf("%s: %v", prefix, err)
}

func (l *LogrusLogger) ErrorE(prefix string, err error) {
	l.toLogrusEntry().WithError(err).Errorf("%s: %v", prefix, err)
}

func (l *LogrusLogger) FatalE(prefix string, err error) {
	l.toLogrusEntry().WithError(err).Fatalf("%s: %v", prefix, err)
}

func (l *LogrusLogger) Error(err error, args ...interface{}) {
	l.toLogrusEntry().WithError(err).Error(args...)
}

func (l *LogrusLogger) Errorf(err error, format string, args ...interface{}) {
	l.toLogrusEntry().WithError(err).Errorf(format, args...)
}

func (l *LogrusLogger) ErrorMsg(args ...interface{}) {
	l.toLogrusEntry().Error(args...)
}

func (l *LogrusLogger) ErrorMsgf(format string, args ...interface{}) {
	l.toLogrusEntry().Errorf(format, args...)
}

func (l *LogrusLogger) Fatal(err error, args ...interface{}) {
	l.toLogrusEntry().WithError(err).Fatal(args...)
}

func (l *LogrusLogger) Fatalf(err error, format string, args ...interface{}) {
	l.toLogrusEntry().WithError(err).Fatalf(format, args...)
}

func (l *LogrusLogger) withField(key string, value interface{}) *LogrusLogger {
	return (*LogrusLogger)(l.toLogrusEntry().WithField(key, value))
}

func (l *LogrusLogger) WithField(key string, value interface{}) LogInterface {
	return l.withField(key, value)
}

func (l *LogrusLogger) withFields(fields logrus.Fields) *LogrusLogger {
	return (*LogrusLogger)(l.toLogrusEntry().WithFields(fields))
}

func (l *LogrusLogger) WithFields(fields logrus.Fields) LogInterface {
	return l.withFields(fields)
}

func (l *LogrusLogger) cloneForID(id string, r *http.Request) *LogrusLogger {
	return (*LogrusLogger)(l.toLogrusEntry().WithField("requestID", id).WithField("requestUri", r.RequestURI))
}

func (l *LogrusLogger) CloneForID(id string, r *http.Request) LogInterface {
	return l.cloneForID(id, r)
}

func (l *LogrusLogger) cloneForThread(thread string) *LogrusLogger {
	return (*LogrusLogger)(l.toLogrusEntry().WithField("thread", thread))
}

func (l *LogrusLogger) CloneForThread(thread string) LogInterface {
	return l.cloneForThread(thread)
}

// LoggerWithSentry is a logger which outputs to the console and sends errors to Sentry. Sentry do provide a Logrus hook,
// but its lacks the ability to handle HTTP requests in the way we want, so we do it manually instead.
//
// We could inherit from LogrusLogger, but that makes it easy to forget to override a method
type LoggerWithSentry struct {
	BaseLogger   *LogrusLogger
	SentryHub    *sentry.Hub
	SendWarnings bool
}

func (l *LoggerWithSentry) sendToSentry(level sentry.Level, err error, message string, data logrus.Fields) {
	l.SentryHub.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(level)

		if data != nil && len(data) > 0 {
			scope.SetContext("logData", data)
		}

		if message != "" && err != nil {
			scope.SetExtra("message", message)
		}

		if err != nil {
			l.SentryHub.CaptureException(err)
		} else {
			l.SentryHub.CaptureMessage(message)
		}
	})
}

func argsToMessage(args []interface{}) string {
	argStrings := generic.Map(args, func(x interface{}) string { return fmt.Sprint(x) })
	return strings.Join(argStrings, " ")
}

func (l *LoggerWithSentry) Debug(args ...interface{}) {
	l.BaseLogger.Debug(args...)
}

func (l *LoggerWithSentry) Debugf(format string, args ...interface{}) {
	l.BaseLogger.Debugf(format, args...)
}

func (l *LoggerWithSentry) Info(args ...interface{}) {
	l.BaseLogger.Info(args...)
}

func (l *LoggerWithSentry) Infof(format string, args ...interface{}) {
	l.BaseLogger.Infof(format, args...)
}

func (l *LoggerWithSentry) Warn(args ...interface{}) {
	message := argsToMessage(args)

	entry := l.BaseLogger.toLogrusEntry()

	if l.SendWarnings {
		l.sendToSentry(sentry.LevelWarning, nil, message, entry.Data)
	}

	entry.Warn(message)
}

func (l *LoggerWithSentry) Warnf(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)

	entry := l.BaseLogger.toLogrusEntry()

	if l.SendWarnings {
		l.sendToSentry(sentry.LevelWarning, nil, message, entry.Data)
	}

	entry.Warn(message)
}

func (l *LoggerWithSentry) ErrorE(prefix string, err error) {
	message := fmt.Sprintf("%s: %v", prefix, err)

	entry := l.BaseLogger.toLogrusEntry().WithError(err)

	l.sendToSentry(sentry.LevelError, err, message, entry.Data)

	entry.Error(message)
}

func (l *LoggerWithSentry) WarnE(prefix string, err error) {
	message := fmt.Sprintf("%s: %v", prefix, err)

	entry := l.BaseLogger.toLogrusEntry().WithError(err)

	if l.SendWarnings {
		l.sendToSentry(sentry.LevelWarning, err, message, entry.Data)
	}

	entry.Warn(message)
}

func (l *LoggerWithSentry) FatalE(prefix string, err error) {
	message := fmt.Sprintf("%s: %v", prefix, err)

	entry := l.BaseLogger.toLogrusEntry().WithError(err)

	l.sendToSentry(sentry.LevelFatal, err, message, entry.Data)

	l.SentryHub.Flush(2 * time.Second)

	entry.Fatal(message)
}

func (l *LoggerWithSentry) Error(err error, args ...interface{}) {
	message := argsToMessage(args)

	entry := l.BaseLogger.toLogrusEntry().WithError(err)

	l.sendToSentry(sentry.LevelError, err, message, entry.Data)

	entry.Error(message)
}

func (l *LoggerWithSentry) Errorf(err error, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)

	entry := l.BaseLogger.toLogrusEntry().WithError(err)

	l.sendToSentry(sentry.LevelError, err, message, entry.Data)

	entry.Error(message)
}

func (l *LoggerWithSentry) ErrorMsg(args ...interface{}) {
	message := argsToMessage(args)

	entry := l.BaseLogger.toLogrusEntry()

	l.sendToSentry(sentry.LevelError, nil, message, entry.Data)

	entry.Error(message)
}

func (l *LoggerWithSentry) ErrorMsgf(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)

	entry := l.BaseLogger.toLogrusEntry()

	l.sendToSentry(sentry.LevelError, nil, message, entry.Data)

	entry.Error(message)
}

func (l *LoggerWithSentry) Fatal(err error, args ...interface{}) {
	message := argsToMessage(args)

	entry := l.BaseLogger.toLogrusEntry().WithError(err)

	l.sendToSentry(sentry.LevelFatal, err, message, entry.Data)

	l.SentryHub.Flush(2 * time.Second)

	entry.Fatal(message)
}

func (l *LoggerWithSentry) Fatalf(err error, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)

	entry := l.BaseLogger.toLogrusEntry().WithError(err)

	l.sendToSentry(sentry.LevelFatal, err, message, entry.Data)

	l.SentryHub.Flush(2 * time.Second)

	entry.Fatal(message)
}

func (l *LoggerWithSentry) WithField(key string, value interface{}) LogInterface {
	return &LoggerWithSentry{
		BaseLogger:   l.BaseLogger.withField(key, value),
		SentryHub:    l.SentryHub,
		SendWarnings: l.SendWarnings,
	}
}

func (l *LoggerWithSentry) WithFields(fields logrus.Fields) LogInterface {
	return &LoggerWithSentry{
		BaseLogger:   l.BaseLogger.withFields(fields),
		SentryHub:    l.SentryHub,
		SendWarnings: l.SendWarnings,
	}
}

func (l *LoggerWithSentry) CloneForID(id string, req *http.Request) LogInterface {
	newHub := l.SentryHub.Clone()

	newHub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("requestID", id)
		scope.SetRequest(req)
	})

	return &LoggerWithSentry{
		BaseLogger:   l.BaseLogger.cloneForID(id, req),
		SentryHub:    newHub,
		SendWarnings: l.SendWarnings,
	}
}

func (l *LoggerWithSentry) CloneForThread(thread string) LogInterface {
	newHub := l.SentryHub.Clone()

	newHub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("thread", thread)
	})

	return &LoggerWithSentry{
		BaseLogger:   l.BaseLogger.cloneForThread(thread),
		SentryHub:    newHub,
		SendWarnings: l.SendWarnings,
	}
}

type GCPLogger struct {
	logger *logging.Logger
	ctx    context.Context
}

func (l *GCPLogger) log(severity logging.Severity, msg string, err error, fields logrus.Fields) {
	entry := logging.Entry{
		Severity: severity,
		Payload:  msg,
	}
	if err != nil {
		if entry.Payload == "" {
			entry.Payload = err.Error()
		} else {
			entry.Payload = fmt.Sprintf("%s: %v", msg, err)
		}
	}
	if fields != nil && len(fields) > 0 {
		entry.Labels = make(map[string]string)
		for k, v := range fields {
			entry.Labels[k] = fmt.Sprint(v)
		}
	}
	l.logger.Log(entry)
}

func (l *GCPLogger) Debug(args ...interface{}) {
	l.log(logging.Debug, argsToMessage(args), nil, nil)
}
func (l *GCPLogger) Debugf(format string, args ...interface{}) {
	l.log(logging.Debug, fmt.Sprintf(format, args...), nil, nil)
}
func (l *GCPLogger) Info(args ...interface{}) {
	l.log(logging.Info, argsToMessage(args), nil, nil)
}
func (l *GCPLogger) Infof(format string, args ...interface{}) {
	l.log(logging.Info, fmt.Sprintf(format, args...), nil, nil)
}
func (l *GCPLogger) Warn(args ...interface{}) {
	l.log(logging.Warning, argsToMessage(args), nil, nil)
}
func (l *GCPLogger) Warnf(format string, args ...interface{}) {
	l.log(logging.Warning, fmt.Sprintf(format, args...), nil, nil)
}
func (l *GCPLogger) WarnE(prefix string, err error) {
	l.log(logging.Warning, prefix, err, nil)
}
func (l *GCPLogger) Error(err error, args ...interface{}) {
	l.log(logging.Error, argsToMessage(args), err, nil)
}
func (l *GCPLogger) Errorf(err error, format string, args ...interface{}) {
	l.log(logging.Error, fmt.Sprintf(format, args...), err, nil)
}
func (l *GCPLogger) ErrorE(prefix string, err error) {
	l.log(logging.Error, prefix, err, nil)
}
func (l *GCPLogger) ErrorMsg(args ...interface{}) {
	l.log(logging.Error, argsToMessage(args), nil, nil)
}
func (l *GCPLogger) ErrorMsgf(format string, args ...interface{}) {
	l.log(logging.Error, fmt.Sprintf(format, args...), nil, nil)
}
func (l *GCPLogger) Fatal(err error, args ...interface{}) {
	l.log(logging.Critical, argsToMessage(args), err, nil)
	os.Exit(1)
}
func (l *GCPLogger) Fatalf(err error, format string, args ...interface{}) {
	l.log(logging.Critical, fmt.Sprintf(format, args...), err, nil)
	os.Exit(1)
}
func (l *GCPLogger) FatalE(prefix string, err error) {
	l.log(logging.Critical, prefix, err, nil)
	os.Exit(1)
}
func (l *GCPLogger) WithField(key string, value interface{}) LogInterface {
	// For simplicity, just return same logger (fields not tracked)
	return l
}
func (l *GCPLogger) WithFields(fields logrus.Fields) LogInterface {
	return l
}
func (l *GCPLogger) CloneForID(id string, req *http.Request) LogInterface {
	return l
}
func (l *GCPLogger) CloneForThread(thread string) LogInterface {
	return l
}

// MultiLogger sends logs to multiple loggers
type MultiLogger struct {
	loggers []LogInterface
}

func (m *MultiLogger) Debug(args ...interface{}) {
	for _, l := range m.loggers {
		l.Debug(args...)
	}
}
func (m *MultiLogger) Debugf(format string, args ...interface{}) {
	for _, l := range m.loggers {
		l.Debugf(format, args...)
	}
}
func (m *MultiLogger) Info(args ...interface{}) {
	for _, l := range m.loggers {
		l.Info(args...)
	}
}
func (m *MultiLogger) Infof(format string, args ...interface{}) {
	for _, l := range m.loggers {
		l.Infof(format, args...)
	}
}
func (m *MultiLogger) Warn(args ...interface{}) {
	for _, l := range m.loggers {
		l.Warn(args...)
	}
}
func (m *MultiLogger) Warnf(format string, args ...interface{}) {
	for _, l := range m.loggers {
		l.Warnf(format, args...)
	}
}
func (m *MultiLogger) WarnE(prefix string, err error) {
	for _, l := range m.loggers {
		l.WarnE(prefix, err)
	}
}
func (m *MultiLogger) Error(err error, args ...interface{}) {
	for _, l := range m.loggers {
		l.Error(err, args...)
	}
}
func (m *MultiLogger) Errorf(err error, format string, args ...interface{}) {
	for _, l := range m.loggers {
		l.Errorf(err, format, args...)
	}
}
func (m *MultiLogger) ErrorE(prefix string, err error) {
	for _, l := range m.loggers {
		l.ErrorE(prefix, err)
	}
}
func (m *MultiLogger) ErrorMsg(args ...interface{}) {
	for _, l := range m.loggers {
		l.ErrorMsg(args...)
	}
}
func (m *MultiLogger) ErrorMsgf(format string, args ...interface{}) {
	for _, l := range m.loggers {
		l.ErrorMsgf(format, args...)
	}
}
func (m *MultiLogger) Fatal(err error, args ...interface{}) {
	for _, l := range m.loggers {
		l.Fatal(err, args...)
	}
}
func (m *MultiLogger) Fatalf(err error, format string, args ...interface{}) {
	for _, l := range m.loggers {
		l.Fatalf(err, format, args...)
	}
}
func (m *MultiLogger) FatalE(prefix string, err error) {
	for _, l := range m.loggers {
		l.FatalE(prefix, err)
	}
}
func (m *MultiLogger) WithField(key string, value interface{}) LogInterface {
	newLoggers := make([]LogInterface, len(m.loggers))
	for i, l := range m.loggers {
		newLoggers[i] = l.WithField(key, value)
	}
	return &MultiLogger{loggers: newLoggers}
}
func (m *MultiLogger) WithFields(fields logrus.Fields) LogInterface {
	newLoggers := make([]LogInterface, len(m.loggers))
	for i, l := range m.loggers {
		newLoggers[i] = l.WithFields(fields)
	}
	return &MultiLogger{loggers: newLoggers}
}
func (m *MultiLogger) CloneForID(id string, req *http.Request) LogInterface {
	newLoggers := make([]LogInterface, len(m.loggers))
	for i, l := range m.loggers {
		newLoggers[i] = l.CloneForID(id, req)
	}
	return &MultiLogger{loggers: newLoggers}
}
func (m *MultiLogger) CloneForThread(thread string) LogInterface {
	newLoggers := make([]LogInterface, len(m.loggers))
	for i, l := range m.loggers {
		newLoggers[i] = l.CloneForThread(thread)
	}
	return &MultiLogger{loggers: newLoggers}
}

func InitialiseLogging(serviceName string, includeDebug bool, sentryConfig SentryConfigInterface) func() {
	var loggers []LogInterface
	var cleanupFuncs []func()

	logrusLogger := logrus.New()
	logrusLogger.ReportCaller = true

	// Use JSON logging format if running in K8S
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" || os.Getenv("FUNCTION_TARGET") != "" {
		logrusLogger.Formatter = glogging.NewFormatter(glogging.WithService(serviceName))
	}

	if includeDebug {
		logrusLogger.SetLevel(logrus.DebugLevel)
	}

	var baseLogger LogInterface = (*LogrusLogger)(logrus.NewEntry(logrusLogger))

	if sentryConfig != nil && sentryConfig.GetDSN() != "" {
		release, err := sentryConfig.GetRelease(serviceName)
		if err != nil {
			logrusLogger.Errorf("Error initialising Sentry: %v", err)
		}

		err = sentry.Init(sentry.ClientOptions{
			Dsn:         sentryConfig.GetDSN(),
			Debug:       includeDebug,
			ServerName:  sentryConfig.GetServerName(),
			Release:     release,
			Environment: sentryConfig.GetEnvironment(),
		})
		if err != nil {
			logrusLogger.Fatalf("Error initialising sentry: %v", err)
		}

		sentry.ConfigureScope(func(scope *sentry.Scope) {
			if sentryConfig.GetMode() != "" {
				scope.SetTag("mode", sentryConfig.GetMode())
			}
		})

		baseLogger = &LoggerWithSentry{
			BaseLogger:   (*LogrusLogger)(logrus.NewEntry(logrusLogger)),
			SentryHub:    sentry.CurrentHub(),
			SendWarnings: sentryConfig.GetSendWarnings(),
		}
		cleanupFuncs = append(cleanupFuncs, func() { sentry.Flush(2 * time.Second) })
	}
	loggers = append(loggers, baseLogger)

	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID != "" {
		logrusLogger.Debugf("Initialising GCP logging for project %s", projectID)
		ctx := context.Background()
		client, err := logging.NewClient(ctx, projectID)
		if err != nil {
			panic(fmt.Sprintf("Failed to create GCP logging client: %v", err))
		}
		gcpClient = client

		var loggerOpts []logging.LoggerOption
		if hostname, err := os.Hostname(); err == nil {
			loggerOpts = append(loggerOpts, logging.CommonLabels(map[string]string{"hostname": hostname}))
		}
		gcpLogger = client.Logger(serviceName, loggerOpts...)
		gcpLog := &GCPLogger{
			logger: gcpLogger,
			ctx:    ctx,
		}
		loggers = append(loggers, gcpLog)
		cleanupFuncs = append(cleanupFuncs, func() { _ = client.Close() })
	}

	if len(loggers) == 1 {
		Logger = loggers[0]
	} else {
		Logger = &MultiLogger{loggers: loggers}
	}

	return func() {
		for _, f := range cleanupFuncs {
			f()
		}
	}
}

type BasicErrorLogger struct{}

func (_ BasicErrorLogger) Println(args ...interface{}) {
	Logger.ErrorMsg(args...)
}

func GetBasicErrorLogger() BasicErrorLogger {
	return BasicErrorLogger{}
}
