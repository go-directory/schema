package schema

import (
	"errors"
	"os"
	"strconv"
)

var (
	errCollectiveSingle = errors.New("Attribute cannot be both COLLECTIVE and SINGLE-VALUE")
	nilInstanceErr      = errors.New("Nil instance error")
	errNotExist         = os.ErrNotExist
)

func errorBadLength(name string, length int) error {
	return errors.New(`Invalid length '` + strconv.FormatInt(int64(length), 10) + `' for ` + name)
}

func errorBadType(name string) error {
	return errors.New(`Incompatible input type for ` + name)
}

func errorPrimerFailed(ls, mr int) (err error) {
	if ls != 0 || mr != 0 {
		err = errors.New("Failed to prime schema: " + strconv.Itoa(ls) + " ldapSyntaxes, " +
			strconv.Itoa(mr) + " matchingRules")
	}

	return
}

func errMalformedOID(typ, noid string) error {
	return errors.New(typ + " bears malformed numeric OID '" + noid + "'")
}
