//go:build !(godir_debug_schema || godir_debug_all)

package schema

/*
NullLogger defines a dead concrete type meant to implement the
[Logger] interface in a non-operational manner.

If and when this type is visible within a given [Go Directory]
package, this indicates that logging has not been enabled via
go build tags.

Logging can be enabled via the following process in any given
[Go Directory] package:

  - go build (or go run) with -t "godir_debug_<pkg_namespace>", or ...
  - go build (or go run) with -t "godir_debug_all"

Next, a non-zero logging level must be set in order for events to
actually appear. Each package bears its own [LogLevel] uint16 global,
which may be manipulated in an additive or subtractive manner according
to the following available "verbosity bit values":

  - 1 (error)
  - 2 (warning)
  - 4 (trace)
  - 8 (info)

To enable errors and warnings within "packageX", set its global
to three (3). Or, to enable everything except trace, set the global
to eleven (11).

To temporarily disable logging without the need to supply build
tags again, simply set this (and other) [LogLevel] globals to
zero (0) as needed.

Keep in mind, the technique of temporarily disabling the [LogLevel]
global as described above will still carry a small degree of overhead
due to conditional logging level bit value checks being conducted for
EVERY logger call present within the relevant package(s). As such, it
is not as performant as rebuilding/rerunning the application with NO
debugger build tags specified whatsoever.

When performance is a critical concern, avoid unnecessary logging
and -- especially -- unnecessary use of the more granular logging
levels such as trace.  Failure to observe this warning WILL yield
measurable performance penalties.

[Go Directory]: https://github.com/go-directory
*/
type NullLogger struct{}

func (_ NullLogger) Printf(_ string, _ ...any) {}
func (_ NullLogger) Fatalf(_ string, _ ...any) {}

func emitW(_ string, _ ...any) {}
func emitE(_ string, _ ...any) {}
func emitT(_ string, _ ...any) {}
func emitI(_ string, _ ...any) {}
func emitM(_ string, _ ...any) {}
func emitErr(_ error)          {}

func init() {
	eventLogger = NullLogger{}
	LogLevel = 0
}
