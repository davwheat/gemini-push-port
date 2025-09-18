package logging

import (
	"fmt"
	"os"
	"strings"
)

var version string

type SentryConfigInterface interface {
	GetDSN() string
	GetEnvironment() string
	GetSendWarnings() bool
	GetRelease(serviceName string) (string, error)
	GetServerName() string
	GetMode() string
}

type SentryConfig struct {
	DSN          string
	Environment  string
	SendWarnings bool
	ServiceName  string // Optional, to override using default logging one
	Mode         string // Optional, for services which can operate in multiple modes
}

func (sc SentryConfig) GetDSN() string {
	return sc.DSN
}

func (sc SentryConfig) GetEnvironment() string {
	return sc.Environment
}

func (sc SentryConfig) GetSendWarnings() bool {
	return sc.SendWarnings
}

func (sc SentryConfig) GetRelease(serviceName string) (string, error) {
	if sc.ServiceName != "" {
		serviceName = sc.ServiceName
	}

	if version == "" {
		version = "unset"
		return serviceName + "@unset", fmt.Errorf("version is not set, using 'unset'")
	}

	return serviceName + "@" + version, nil
}

func (sc SentryConfig) GetServerName() string {
	return os.Getenv("HOSTNAME")
}

func (sc SentryConfig) GetMode() string {
	return sc.Mode
}

type BindEnvInterface interface {
	BindEnv(values ...string) error
}

func BindSentryConfig(be BindEnvInterface, paths ...string) {
	dotPath := ""
	underPath := ""
	if len(paths) > 0 {
		dotPath = strings.Join(paths, ".") + "."
		underPath = strings.Join(paths, "_") + "_"
	}

	_ = be.BindEnv(dotPath+"DSN", underPath+"DSN")
	_ = be.BindEnv(dotPath+"Environment", underPath+"Environment")
	_ = be.BindEnv(dotPath+"SendWarnings", underPath+"SendWarnings")
	_ = be.BindEnv(dotPath+"ServiceName", underPath+"ServiceName")
	_ = be.BindEnv(dotPath+"Mode", underPath+"Mode")
}
