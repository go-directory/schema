package schema

import (
	"github.com/go-directory/common"
)

/*
Logger aliases the [common.Logger] interface.
*/
type Logger = common.Logger

/*
eventLogger stores the [Logger] instance meant to handle or discard
log events. Users may supply their own concrete [Logger] type if
desired.
*/
var eventLogger Logger

var genTimeNow = common.GenTimeNow

/*
[LogLevel] defines the desired degree of verbosity.

The value should be set in an additive or subtractive manner
according to the following available "verbosity bit values":

  - 1 (error) -- intended for very serious events
  - 2 (warning) -- intended for potentially serious events
  - 4 (trace) -- intended for deep-level events
  - 8 (info) -- intended for succinct informational events

For instance, to enable errors and info, set the value to nine
(9).

To test whether a particular logging level is set, evaluate via:

	LogLevel&uint16(8) != 0 // is "info" logging enabled?

A return value of true indicates the specified uint16 bit is set.
*/
var LogLevel uint16 = 65535

const (
	// not to exceed uint16
	logError   uint16 = 1 << iota // 1
	logWarning                    // 2
	logTrace                      // 4
	logInfo                       // 8
	logMoreInfo                   // 16
)
