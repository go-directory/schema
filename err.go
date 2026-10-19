package schema

import (
	"errors"
	"os"

	"github.com/go-directory/common"
)

type commonErr = common.Error

func syntaxError(msg ...string) (err commonErr) {
	err = common.NewError(uint16(common.LDAPResultInvalidAttributeSyntax), msg...)
	err.SetHeader("syntax:")
	return
}

func schemaError(msg ...string) (err commonErr) {
	err = common.NewError(uint16(common.ErrorSchema), msg...)
	err.SetHeader("schema:")
	return
}

var (
	errCollectiveSingle = schemaError("Attribute cannot be both COLLECTIVE and SINGLE-VALUE")
	nilInstanceErr      = schemaError("Nil instance error")
	errNotExist         = os.ErrNotExist
)

var errorIs = errors.Is

func errorBadLength(name string, length int) error {
	return schemaError(`Invalid length '` +
		fint(int64(length), 10) + `' for ` + name)
}

func errorBadType(name string) error {
	return schemaError(`Incompatible input type for ` + name)
}

func errMalformedOID(typ, noid string) error {
	return syntaxError(typ, " bears malformed numeric OID '", noid, "'")
}
