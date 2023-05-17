package log

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"go.elastic.co/ecslogrus"
)

const HTTPRequestTrackingHeader string = "X-Request-Id"
const RequestIDField string = "request_id"
const RuneSet string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_."
const srcDirName string = "/src/"

var logger *logrus.Logger = nil

func defaultLogger() *logrus.Entry {
	if logger == nil {
		if logger == nil {
			logger = logrus.New()
			logger.SetFormatter(&ecslogrus.Formatter{})
		}
	}

	_, fileName, lineNumber, _ := runtime.Caller(2)
	fileName = srcDirName + strings.Split(fileName, srcDirName)[1]
	directoryName := filepath.Base(filepath.Dir(fileName))
	fileName = filepath.Base(fileName)

	return logger.WithFields(logrus.Fields{
		"file_path":   fmt.Sprintf("./%s/%s", directoryName, fileName),
		"file_name":   fileName,
		"line_number": lineNumber,
	})
}

func WithField(key string, value interface{}) *logrus.Entry {
	return defaultLogger().WithField(key, value)
}

func WithFields(fields logrus.Fields) *logrus.Entry {
	return defaultLogger().WithFields(fields)
}

func WithError(err error) *logrus.Entry {
	return defaultLogger().WithError(err)
}

func WithContext(ctx context.Context) *logrus.Entry {
	return defaultLogger().WithContext(ctx)
}

func WithTime(t time.Time) *logrus.Entry {
	return defaultLogger().WithTime(t)
}

func Trace(args ...interface{}) {
	defaultLogger().Trace(args...)
}

func Debug(args ...interface{}) {
	defaultLogger().Debug(args...)
}

func Info(args ...interface{}) {
	defaultLogger().Info(args...)
}

func Print(args ...interface{}) {
	defaultLogger().Print(args...)
}

func Warn(args ...interface{}) {
	defaultLogger().Warn(args...)
}

func Warning(args ...interface{}) {
	defaultLogger().Warning(args...)
}

func Error(args ...interface{}) {
	defaultLogger().Error(args...)
}

func Fatal(args ...interface{}) {
	defaultLogger().Fatal(args...)
}

func Panic(args ...interface{}) {
	defaultLogger().Panic(args...)
}

func Logln(level logrus.Level, args ...interface{}) {
	defaultLogger().Logln(level, args...)
}

func Traceln(args ...interface{}) {
	defaultLogger().Traceln(args...)
}

func Debugln(args ...interface{}) {
	defaultLogger().Debugln(args...)
}

func Infoln(args ...interface{}) {
	defaultLogger().Infoln(args...)
}

func Println(args ...interface{}) {
	defaultLogger().Println(args...)
}

func Warnln(args ...interface{}) {
	defaultLogger().Warnln(args...)
}

func Warningln(args ...interface{}) {
	defaultLogger().Warningln(args...)
}

func Errorln(args ...interface{}) {
	defaultLogger().Errorln(args...)
}

func Fatalln(args ...interface{}) {
	defaultLogger().Fatalln(args...)
}

func Panicln(args ...interface{}) {
	defaultLogger().Panicln(args...)
}

func Logf(level logrus.Level, format string, args ...interface{}) {
	defaultLogger().Logf(level, format, args...)
}

func Tracef(format string, args ...interface{}) {
	defaultLogger().Tracef(format, args...)
}

func Debugf(format string, args ...interface{}) {
	defaultLogger().Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	defaultLogger().Infof(format, args...)
}

func Printf(format string, args ...interface{}) {
	defaultLogger().Printf(format, args...)
}

func Warnf(format string, args ...interface{}) {
	defaultLogger().Warnf(format, args...)
}

func Warningf(format string, args ...interface{}) {
	defaultLogger().Warningf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	defaultLogger().Errorf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	defaultLogger().Fatalf(format, args...)
}

func Panicf(format string, args ...interface{}) {
	defaultLogger().Panicf(format, args...)
}

func HTTPRequest(c *fiber.Ctx) {
	remoteIPList := []string{
		c.IP(),
	}
	if string(c.Request().Header.Peek("X-Forwarded-For")) != "" {
		remoteIPList = append(remoteIPList, string(c.Request().Header.Peek("X-Forwarded-For")))
	}

	cookies := map[string]string{}
	headers := map[string]string{}
	queryArgs := map[string]string{}
	c.Request().Header.VisitAll(func(key []byte, value []byte) { headers[string(key)] = string(value) })
	c.Request().Header.VisitAllCookie(func(key []byte, value []byte) { cookies[string(key)] = string(value) })
	c.Request().URI().QueryArgs().VisitAll(func(key []byte, value []byte) { queryArgs[string(key)] = string(value) })

	WithFields(logrus.Fields{
		"nginx": map[string]interface{}{
			"access": map[string]interface{}{
				"agent": string(c.Request().Header.UserAgent()),
				"body_sent": map[string]interface{}{
					"bytes": "",
				},
				"http_version":   string(c.Request().Header.Protocol()),
				"method":         string(c.Request().Header.Method()),
				"referrer":       string(c.Request().Header.Referer()),
				"remote_ip_list": remoteIPList,
				"response_code":  "",
				"url":            c.Request().URI().String(),
				"user_name":      "",
				"user_agent": map[string]interface{}{
					"device":   "",
					"name":     "",
					"original": string(c.Request().Header.UserAgent()),
					"os":       "",
					"os_name":  "",
				},
				"geoip": map[string]interface{}{
					"continent_name":   "",
					"country_iso_code": "",
					"location":         "",
					"region_name":      "",
					"city_name":        "",
					"region_iso_code":  "",
				},
			},
		},
		"request": map[string]interface{}{
			"addr":    c.IP(),
			"cookies": cookies,
			"headers": headers,
			"method":  string(c.Request().Header.Method()),
			"path":    string(c.Request().URI().Path()),
			"query":   queryArgs,
			"url":     c.Request().URI().String(),
		},
		RequestIDField: c.Request().Header.Peek(HTTPRequestTrackingHeader),
	}).Info(fmt.Sprintf("%.7s %s", string(c.Request().Header.Method()), string(c.Request().URI().Path())))
}

func TraceHTTPRequest(c *fiber.Ctx) *logrus.Entry {
	requestID := c.Request().Header.Peek(HTTPRequestTrackingHeader)
	return WithFields(logrus.Fields{
		"request.id": requestID,
	})
}
