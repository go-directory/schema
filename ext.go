package schema

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
)

/*
Extension implements [§ 4.2 of RFC 4512] and describes a single extension
using an "xstring" ([]byte) and one or more quoted []byte values.

[§ 4.2 of RFC 4512]: https://datatracker.ietf.org/doc/html/rfc4512#section-4.2
*/
type Extension struct {
	XString []byte
	Values  [][]byte
}

func marshalExtension(token, typ string, tkz *schemaTokenizer) (ext Extension, err error) {
	if tpfx := strings.ToUpper(token); strings.HasPrefix(tpfx, "X-") {
		ext = Extension{
			XString: []byte(tpfx),
			Values:  parseMultiVal(tkz),
		}
	} else {
		err = errors.New(typ + ": Unknown token in definition: " + token)
	}

	return
}

/*
boolValue returns a Boolean value indicative of both of the following
evaluating as true:

  - Receiver contains an XString field value matching the input name, and ...
  - Receiver contains a single Values slice value that is the string representation of a Boolean
*/
func (r Extension) boolValue(name []byte) bool {
	var b bool
	if len(r.Values) == 1 && eqFoldASCII(name, r.XString) {
		b, _ = strconv.ParseBool(string(r.Values[0]))
	}

	return b
}

func stringExtensions(exts map[uint]Extension) (s []byte) {
	var ct int = len(exts)
	bld := &bytes.Buffer{}
	for i := 0; i < ct; i++ {
		if _, found := exts[uint(i)]; found {
			bld.WriteString(exts[uint(i)].string())
		} else {
			ct++
		}
	}

	s = bld.Bytes()

	return
}

/*
String returns the string representation of the receiver instance.
*/
func (r Extension) string() (ext string) {
	if len(r.XString) > 0 && len(r.Values) > 0 {
		ext = ` ` + string(r.XString) + ` ` + string(stringQuotedDescrs(r.Values))
	}

	return
}
