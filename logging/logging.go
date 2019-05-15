// Package logging implements improved logging for Google Cloud Functions.
//
// Improvements include support for log levels as well as execution ids.
//
// Usage:
//    func HelloWorld(w http.ResponseWriter, r *http.Request) {
//        ctx := logging.ForRequest(r)
//        // ...
//        logging.Info(ctx).Println("Hello logs")
//        logging.Error(ctx).Println("Hello logs")
//    }
package logging

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"cloud.google.com/go/functions/metadata"
	"cloud.google.com/go/logging"

	"google.golang.org/genproto/googleapis/api/monitoredres"
)

var logger *logging.Logger

func init() {
	project := os.Getenv("GCP_PROJECT")
	function := os.Getenv("FUNCTION_NAME")
	region := os.Getenv("FUNCTION_REGION")

	if project == "" {
		fmt.Fprintln(os.Stderr, "Failed to create logging client:", "GCP_PROJECT environment variable unset or missing")
		return
	}
	if function == "" {
		fmt.Fprintln(os.Stderr, "Failed to create logging client:", "FUNCTION_NAME environment variable unset or missing")
		return
	}
	if region == "" {
		fmt.Fprintln(os.Stderr, "Failed to create logging client:", "FUNCTION_REGION environment variable unset or missing")
		return
	}

	ctx := context.Background()
	client, err := logging.NewClient(ctx, project)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create logging client:", err)
		return
	}

	res := monitoredres.MonitoredResource{
		Type:   "cloud_function",
		Labels: map[string]string{"region": region, "function_name": function},
	}

	logger = client.Logger("cloudfunctions.googleapis.com/cloud-functions", logging.CommonResource(&res))
}

type contextKey struct{}

// ForRequest creates a logging Context for the Request.
func ForRequest(r *http.Request) context.Context {
	ctx := r.Context()
	id := r.Header.Get("Function-Execution-Id")
	if id != "" {
		ctx = context.WithValue(ctx, contextKey{}, id)
	}
	return ctx
}

// Flush all loggers. Blocking.
func Flush() error {
	if logger != nil {
		return logger.Flush()
	}
	return nil
}

// A Logger represents an contextualized logging object that pushes entries to Stackdriver.
type Logger struct {
	s  logging.Severity
	id string
}

func (l Logger) log(s string) {
	s = strings.TrimRight(s, "\n")

	if logger != nil {
		entry := logging.Entry{
			Severity: l.s,
			Payload:  s,
		}

		if l.id != "" {
			entry.Labels = map[string]string{"execution_id": l.id}
		}

		logger.Log(entry)
		return
	}

	if l.s >= logging.Error {
		fmt.Fprintln(os.Stderr, s)
	} else {
		fmt.Println(s)
	}
}

// Print logs using the default formats for its operands.
// Spaces are added between operands when neither is a string.
func (l Logger) Print(v ...interface{}) {
	l.log(fmt.Sprint(v...))
}

// Println logs using the default formats for its operands.
// Spaces are always added between operands and a newline is appended.
func (l Logger) Println(v ...interface{}) {
	l.log(fmt.Sprintln(v...))
}

// Printf logs according to a format specifier.
func (l Logger) Printf(format string, v ...interface{}) {
	l.log(fmt.Sprintf(format, v...))
}

func newLogger(ctx context.Context, s logging.Severity) Logger {
	l := Logger{s: s}
	if ctx != nil {
		if meta, _ := metadata.FromContext(ctx); meta != nil {
			l.id = meta.EventID
		} else {
			l.id, _ = ctx.Value(contextKey{}).(string)
		}
	}
	return l
}

// Default gets a Logger with no assigned severity level.
func Default(ctx context.Context) Logger {
	return newLogger(ctx, logging.Default)
}

// Debug gets a Logger for debug or trace information.
func Debug(ctx context.Context) Logger {
	return newLogger(ctx, logging.Debug)
}

// Info gets a Logger for routine information, such as ongoing status or performance.
func Info(ctx context.Context) Logger {
	return newLogger(ctx, logging.Info)
}

// Notice gets a Logger for normal but significant events, such as start up, shut down, or configuration.
func Notice(ctx context.Context) Logger {
	return newLogger(ctx, logging.Notice)
}

// Warning gets a Logger for events that might cause problems.
func Warning(ctx context.Context) Logger {
	return newLogger(ctx, logging.Warning)
}

// Error gets a Logger for events that are likely to cause problems.
func Error(ctx context.Context) Logger {
	return newLogger(ctx, logging.Error)
}

// Critical gets a Logger for events that cause more severe problems or brief outages.
func Critical(ctx context.Context) Logger {
	return newLogger(ctx, logging.Critical)
}

// Alert gets a Logger for when a person must take an action immediately.
func Alert(ctx context.Context) Logger {
	return newLogger(ctx, logging.Alert)
}

// Emergency gets a Logger for when one or more systems are unusable.
func Emergency(ctx context.Context) Logger {
	return newLogger(ctx, logging.Emergency)
}
