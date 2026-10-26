package schema

import (
	"bytes"
	"strconv"
)

var (
	itoa = strconv.Itoa
	atoi = strconv.Atoi
	fint = strconv.FormatInt
)

var (
	bLC  = bytes.ToLower
	bEQ  = bytes.Equal
	bEQF = bytes.EqualFold
)
