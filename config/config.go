package config

import (
	"encoding/json"
	"errors"
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
	//AccessLog is the access log file name
	AccessLog = "access.log"
	//AppLog is the application log file name
	AppLog = "authenvoy.log"
	//EventLog is the event log file name
	EventLog = "event.log"
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
	Event             string        `json:"Event"`
	EventWriter       *json.Encoder `json:"-"`
	Application       string        `json:"Application"`
	ApplicationWriter *log.Logger   `json:"-"`
	Access            string        `json:"Access"`
	AccessWriter      *json.Encoder `json:"-"`
}

// New returns a new Config instance.
func New(port int, krbconf, lp string) (*Config, error) {
	if port > 65535 || port < 1 {
		return &Config{}, errors.New("port number invalid")
	}
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
	err = c.SetApplicationLog(lp)
	if err != nil {
		return &Config{}, err
	}
	err = c.SetAccessLog(lp)
	if err != nil {
		return &Config{}, err
	}
	err = c.SetEventLog(lp)
	if err != nil {
		return &Config{}, err
	}
	return c, nil
}

func (c *Config) logWriter(p string, f string) (w io.Writer, wp string, err error) {
	wp = strings.TrimSuffix(p, "/")
	switch strings.ToLower(wp) {
	case "":
		wp = "stdout"
		w = os.Stdout
	case "stdout":
		wp = "stdout"
		w = os.Stdout
	case "stderr":
		wp = "stderr"
		w = os.Stderr
	case "null":
		wp = "null"
		w = ioutil.Discard
	default:
		wp = p + "/" + f
		w, err = os.OpenFile(wp, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
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
func (c *Config) SetApplicationLog(p string) error {
	w, wp, err := c.logWriter(p, AppLog)
	if err != nil {
		c.ApplicationLogf("could not open application log file: %v\n", err)
		return err
	}
	c.Loggers.Application = wp
	l := log.New(w, logPrefix, log.Ldate|log.Ltime)
	c.SetApplicationLogWriter(l)
	return nil
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
func (c *Config) SetAccessLog(p string) error {
	w, wp, err := c.logWriter(p, AccessLog)
	if err != nil {
		c.ApplicationLogf("could not open access log file: %v\n", err)
		return err
	}
	c.Loggers.Access = wp
	enc := json.NewEncoder(w)
	c.SetAccessLogWriter(enc)
	return nil
}

// AccessLog write the value provided to the access log.
func (c Config) AccessLog(v interface{}) {
	if c.Loggers.AccessWriter != nil {
		err := c.Loggers.AccessWriter.Encode(v)
		if err != nil {
			c.ApplicationLogf("could not log access event: %v\n", err)
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
func (c *Config) SetEventLog(p string) error {
	w, wp, err := c.logWriter(p, EventLog)
	if err != nil {
		c.ApplicationLogf("could not open event log file: %v\n", err)
		return err
	}
	c.Loggers.Event = wp
	enc := json.NewEncoder(w)
	c.SetEventLogWriter(enc)
	return nil
}

// SetEventLogWriter sets the event log lines to be written to the JSON encoder provided.
func (c *Config) SetEventLogWriter(e *json.Encoder) *Config {
	c.Loggers.EventWriter = e
	return c
}

// EventLog write the value provided to the access log.
func (c *Config) EventLog(v interface{}) {
	if c.Loggers.EventWriter != nil {
		err := c.Loggers.EventWriter.Encode(v)
		if err != nil {
			c.ApplicationLogf("could not log event: %v\n", err)
		}
	}
}
