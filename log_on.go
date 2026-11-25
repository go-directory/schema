//go:build godir_debug_schema || godir_debug_all

package schema

import (
	"log"
	"sync"
)

type consoleLogger struct{}

func (_ consoleLogger) Printf(s string, args ...any) { log.Printf(s, args...) }
func (_ consoleLogger) Fatalf(s string, args ...any) { log.Fatalf(s, args...) }

var setEvtLogLocker = &sync.Mutex{}

/*
SetEventLogger activates the provided [Logger] instance.

The variadic emitter input argument, when specified, allows the override
of the event emission using a user-authored closure instance of the
signature shown.
*/
func SetEventLogger(l Logger, emitter ...func(string, ...any)) {
	setEvtLogLocker.Lock()
	defer setEvtLogLocker.Unlock()

	if l != nil {
		eventLogger = l
		emit = defaultEmitter
		if len(emitter) > 0 && emitter[0] != nil {
			emit = emitter[0]
		}
	}
}

/*
emitW emits warning events.
*/
func emitW(m string, v ...any) {
	if LogLevel&logWarning != 0 {
		emit(m, v...)
	}
}

/*
emitE emits error events.
*/
func emitE(m string, v ...any) {
	if LogLevel&logError != 0 {
		emit(m, v...)
	}
}

/*
emitT emits trace events. USE WITH CAUTION.
*/
func emitT(m string, v ...any) {
	if LogLevel&logTrace != 0 {
		emit(m, v...)
	}
}

/*
emitI emits informational events.
*/
func emitI(m string, v ...any) {
	if LogLevel&logInfo != 0 {
		emit(m, v...)
	}
}

/*
emitM emits MOAR informational events.
*/
func emitM(m string, v ...any) {
	if LogLevel&logMoreInfo != 0 {
		emit(m, v...)
	}
}

func emitErr(err error) {
	if err != nil {
		emitE(err.Error())
	}
}

/*
emit is the core function that actually emits events
to the configured [Logger]. This variable is set via
the [SetEventLogger] function.
*/
var emit func(string, ...any)

// default emitter
func defaultEmitter(m string, v ...any) {
	ts := string(genTimeNow(true))
	eventLogger.Printf(ts+" [SCHEMA} "+m, v...)
}

func init() {
	log.SetFlags(0)
	SetEventLogger(consoleLogger{})
}
