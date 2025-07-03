// Copyright 2025 Rubrik, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/log"
)

// subSystem is the subsystem name used by the API logger.
const subSystem = "api"

type apiLogger struct {
	// tflog embedds the logger in a context that must be passed to each log call.
	ctx context.Context
}

var _ = log.Logger(&apiLogger{})

// newAPILogger returns a new logger that logs to the Terraform log system.
// The log level is read from the TF_LOG_PROVIDER_POLARIS_API environment.
func newAPILogger(ctx context.Context) *apiLogger {
	return &apiLogger{
		ctx: tflog.NewSubsystem(ctx, subSystem, tflog.WithLevelFromEnv("TF_LOG_PROVIDER_POLARIS_API")),
	}
}

func (l *apiLogger) Print(lvl log.LogLevel, args ...any) {
	l.print(lvl, fmt.Sprint(args...))
}

func (l *apiLogger) Printf(lvl log.LogLevel, format string, args ...any) {
	l.print(lvl, fmt.Sprintf(format, args...))
}

func (l *apiLogger) print(level log.LogLevel, msg string) {
	var fields map[string]any

	// Extract caller similar to how rubrik-polaris-sdk-for-go/pkg/polaris/log does it.
	if n := log.PkgFuncName(3); n != "" {
		fields = map[string]any{"call": n}
	}

	switch level {
	case log.Trace:
		tflog.SubsystemTrace(l.ctx, subSystem, msg, fields)
	case log.Debug:
		tflog.SubsystemDebug(l.ctx, subSystem, msg, fields)
	case log.Info:
		tflog.SubsystemInfo(l.ctx, subSystem, msg, fields)
	case log.Warn:
		tflog.SubsystemWarn(l.ctx, subSystem, msg, fields)
	case log.Error:
		tflog.SubsystemError(l.ctx, subSystem, msg, fields)
	case log.Fatal:
		tflog.SubsystemError(l.ctx, subSystem, msg, fields)
		os.Exit(1)
	}
}

func (l *apiLogger) SetLogLevel(level log.LogLevel) {
	// We don't need to change log levels dynamically.
	panic("not implemented")
}
