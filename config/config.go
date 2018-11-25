package config

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"gopkg.in/jcmturner/gokrb5.v6/config"
)

const (
	logPrefix = "Auth Envoy: "
	accessLog = "access.log"
	appLog    = "application.log"
)

// Config holds the application's configuration values and loggers.
type Config struct {
	Port     int
	LogPath  string
	Loggers  Loggers
	KRB5Conf *config.Config
}

// Loggers holds the logging configuration for the application.
type Loggers struct {
	Event             string
	EventWriter       *json.Encoder
	Application       string        `json:"Application"`
	ApplicationWriter *log.Logger   `json:"-"`
	Access            string        `json:"Access"`
	AccessWriter      *json.Encoder `json:"-"`
}

// NewConfig returns a new Config instance.
func New(port int, krbconf, lp string) (*Config, error) {
	lp = strings.TrimSuffix(lp, "/") + "/"
	k, err := config.Load(krbconf)
	if err != nil {
		return &Config{}, fmt.Errorf("could not load krb5.conf: %v", err)
	}
	c := &Config{
		Port:     port,
		LogPath:  lp,
		KRB5Conf: k,
	}
	//Default logging to stdout
	c.SetApplicationLog(lp + appLog).
		SetAccessLog(lp + accessLog)
	return c, nil
}

func (c *Config) logWriter(p string) (w io.Writer, err error) {
	switch strings.ToLower(p) {
	case "":
		w = os.Stdout
	case "stdout":
		w = os.Stdout
	case "stderr":
		w = os.Stderr
	case "null":
		w = ioutil.Discard
	default:
		w, err = os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	}
	return
}

// SetApplicationLogWriter sets the application log lines to be written to the Logger provided.
func (c *Config) SetApplicationLogWriter(l *log.Logger) *Config {
	c.Loggers.ApplicationWriter = l
	return c
}

// SetApplicationLog sets the application log to the file path specified.
// The following values special values can be used:
//
// stdout
//
// stderr
//
// null - discard log lines
func (c *Config) SetApplicationLog(p string) *Config {
	w, err := c.logWriter(p)
	if err != nil {
		c.ApplicationLogf("could not open application log file: %v\n", err)
	}
	c.Loggers.Application = p
	l := log.New(w, logPrefix, log.Ldate|log.Ltime)
	c.SetApplicationLogWriter(l)
	return c
}

// ApplicationLogf formats according to a format specifier and writes the value to the application log.
func (c Config) ApplicationLogf(format string, v ...interface{}) {
	if c.Loggers.ApplicationWriter == nil {
		l := log.New(os.Stdout, logPrefix, log.Ldate|log.Ltime)
		c.Loggers.ApplicationWriter = l
	}
	if len(v) > 0 {
		c.Loggers.ApplicationWriter.Printf(format, v...)
	} else {
		c.Loggers.ApplicationWriter.Print(format)
	}
}

// SetAccessLogWriter sets the access log lines to be written to the JSON encoder provided.
func (c *Config) SetAccessLogWriter(e *json.Encoder) *Config {
	c.Loggers.AccessWriter = e
	return c
}

// SetAccessLog sets the access log to the file path specified.
// The following values special values can be used:
//
// stdout
//
// stderr
//
// null - discard log lines
func (c *Config) SetAccessLog(p string) *Config {
	w, err := c.logWriter(p)
	if err != nil {
		c.ApplicationLogf("could not open access log file: %v\n", err)
	}
	c.Loggers.Access = p
	enc := json.NewEncoder(w)
	c.SetAccessLogWriter(enc)
	return c
}

// AccessLog write the value provided to the access log.
func (c Config) AccessLog(v interface{}) {
	if c.Loggers.AccessWriter != nil {
		err := c.Loggers.AccessWriter.Encode(v)
		if err != nil {
			c.ApplicationLogf("could not log access event: %+v - Error: %v\n", err)
		}
	}
}

// SetEventLog sets the access log to the file path specified.
// The following values special values can be used:
//
// stdout
//
// stderr
//
// null - discard log lines
func (c *Config) SetEventLog(p string) *Config {
	w, err := c.logWriter(p)
	if err != nil {
		c.ApplicationLogf("could not open event log file: %v\n", err)
	}
	c.Loggers.Event = p
	enc := json.NewEncoder(w)
	c.SetEventLogWriter(enc)
	return c
}

// SetEventLogWriter sets the event log lines to be written to the JSON encoder provided.
func (c *Config) SetEventLogWriter(e *json.Encoder) *Config {
	c.Loggers.EventWriter = e
	return c
}

// AccessLog write the value provided to the access log.
func (c Config) EventLog(v interface{}) {
	if c.Loggers.EventWriter != nil {
		err := c.Loggers.EventWriter.Encode(v)
		if err != nil {
			c.ApplicationLogf("could not log event: %+v - Error: %v\n", err)
		}
	}
}
