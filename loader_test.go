package confetti

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testLogLevel uint8

const (
	logLevelDebug testLogLevel = iota + 1
	logLevelInfo
	logLevelWarn
	logLevelError
	logLevelFatal
)

func (l testLogLevel) String() string {
	return fmt.Sprintf("testLogLevel(%d)", l)
}

func (l *testLogLevel) Set(s string) (err error) {
	switch strings.ToLower(s) {
	case "debug":
		*l = logLevelDebug
	case "info":
		*l = logLevelInfo
	case "warn":
		*l = logLevelWarn
	case "error":
		*l = logLevelError
	case "fatal":
		*l = logLevelFatal
	default:
		err = fmt.Errorf("unsupported log level value: %s", s)
	}
	return
}

type testLogOutput string

func (l *testLogOutput) Scan(src any) error {
	path, ok := src.(string)
	if !ok {
		return fmt.Errorf("testLogOutput must be string, not %T", src)
	}

	if path == "stdout" || path == "stderr" {
		return nil
	}

	fd, err := os.Open(path)
	if err != nil {
		return err
	}
	*l = testLogOutput(path)
	return fd.Close()
}

type testBackend struct {
	kv map[string]any
}

func (b *testBackend) From(key string) func(context.Context, any) error {
	return func(_ context.Context, field any) error {
		val, ok := b.kv[key]
		if !ok {
			return nil
		}
		if vs, ok := field.(interface{ Scan(src any) error }); ok {
			return vs.Scan(val)
		}

		fv := reflect.ValueOf(field)
		if fv.Kind() != reflect.Ptr {
			return errors.New("value must be a pointer")
		}
		elem := fv.Elem()
		elem.Set(reflect.ValueOf(val))
		return nil
	}
}

func TestLoader(t *testing.T) {
	type logging struct {
		Level  testLogLevel
		Output testLogOutput
	}

	type config struct {
		User     string
		Password string
		Host     string
		Port     uint16
		MaxConns int32
		Verbose  bool
		Logging  logging
	}

	tmpFd, err := os.CreateTemp(os.TempDir(), "")
	assert.NoError(t, err)
	defer tmpFd.Close()

	tmpFilepath, err := filepath.Abs(filepath.Dir(tmpFd.Name()))
	assert.NoError(t, err)

	tb := &testBackend{
		kv: map[string]any{
			"host":       "db.server01.local",
			"port":       uint16(27017),
			"user":       "admin",
			"password":   "sHimbaBo0mbA",
			"log_level":  logLevelDebug,
			"log_output": tmpFilepath,
		},
	}

	cfg := config{
		MaxConns: 42,
		Logging: logging{
			Level: logLevelInfo,
		},
	}

	err = NewLoader().
		Load(context.Background(),
			To(&cfg.User, tb.From("user")),
			To(&cfg.Password, tb.From("password")),
			To(&cfg.Host, tb.From("host")),
			To(&cfg.Port, tb.From("port")),
			To(&cfg.MaxConns, tb.From("max_conns")),
			To(&cfg.Verbose, tb.From("verbose")),
			To(&cfg.Logging.Level, tb.From("log_level")),
			To(&cfg.Logging.Output, tb.From("log_output")),
		)
	assert.NoError(t, err)

	expected := config{
		User:     "admin",
		Password: "sHimbaBo0mbA",
		Host:     "db.server01.local",
		Port:     27017,
		MaxConns: 42,
		Verbose:  false,
		Logging: logging{
			Level:  logLevelDebug,
			Output: testLogOutput(tmpFilepath),
		},
	}
	assert.Equal(t, expected, cfg)
}
