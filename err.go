package schema

import (
	"errors"
	"os"
	"strconv"

	"github.com/go-directory/common"
)

func syntaxError(msg ...string) error {
	m := append([]string{"Syntax error: "}, msg...)
	return common.LDAPResultInvalidAttributeSyntax.New(m...)
}

func schemaError(msg ...string) error {
	m := append([]string{"Schema error: "}, msg...)
	return common.ErrorSchema.New(m...)
}

var (
	errCollectiveSingle = schemaError("Attribute cannot be both COLLECTIVE and SINGLE-VALUE")
	nilInstanceErr      = schemaError("Nil instance error")
	errNotExist         = os.ErrNotExist
)

var errorIs = errors.Is

func errorBadLength(name string, length int) error {
	return schemaError(`Invalid length '` + strconv.FormatInt(int64(length), 10) + `' for ` + name)
}

func errorBadType(name string) error {
	return schemaError(`Incompatible input type for ` + name)
}

func errorPrimerFailed(ls, mr int) (err error) {
	if ls != 0 || mr != 0 {
		err = schemaError("Failed to prime schema: " + strconv.Itoa(ls) + " ldapSyntaxes, " +
			strconv.Itoa(mr) + " matchingRules")
	}

	return
}

func errMalformedOID(typ, noid string) error {
	return syntaxError(typ, " bears malformed numeric OID '", noid, "'")
}
