package schema

import (
	"bufio"
	"bytes"
	"errors"
	"regexp"
	"strconv"
	"unicode"
)

func definitionDescription(desc []byte) (def []byte) {
	if len(desc) > 0 {
		def = append(def, []byte(" DESC '")...)
		def = append(def, desc...)
		def = append(def, '\'')
	}
	return
}

func definitionName(name [][]byte) (def []byte) {
	switch len(name) {
	case 0:
	default:
		if len(name) > 0 {
			def = append(def, []byte(" NAME ")...)
			def = append(def, stringQuotedDescrs(name)...)
		}
	}
	return
}

func definitionMVDescriptors(key string, src any, dsr ...bool) (clause []byte) {
	var isDsr bool
	if len(dsr) > 0 {
		isDsr = dsr[0]
	}
	switch tv := src.(type) {
	case []byte:
		if len(tv) > 0 {
			clause = append(clause, ' ')
			clause = append(clause, bytes.ToUpper([]byte(key))...)
			clause = append(clause, ' ')
			clause = append(clause, tv...)
		}
	case [][]byte:
		if len(tv) > 0 {
			delim := []byte(" $ ")
			if isDsr {
				delim = []byte(" ")
			}
			clause = append(clause, ' ')
			clause = append(clause, bytes.ToUpper([]byte(key))...)
			clause = append(clause, ' ')
			clause = append(clause, stringDescrs(tv, delim)...)
		}
	}
	return
}

func (r *schemaTokenizer) startTokenParen() {
	if r.next() && bytes.Equal(r.this(), []byte("(")) {
		r.next()
	}
}

func parseMultiVal(tkz *schemaTokenizer) (values [][]byte) {
	token := tkz.nextToken()
	if bytes.Equal(token, []byte("(")) {
		for tkz.next() {
			token := tkz.this()
			if bytes.Equal(token, []byte(")")) {
				break
			} else if bytes.Equal(token, []byte("$")) {
				continue
			}
			values = append(values, bytes.Trim(token, "'"))
		}
	} else {
		values = append(values, bytes.Trim(token, "'"))
	}
	return
}

func parseSingleVal(tkz *schemaTokenizer) (val []byte) {
	return bytes.Trim(tkz.nextToken(), "'")
}

type schemaTokenizer struct {
	input []byte
	pos   int
	cur   []byte
}

func newSchemaTokenizer(input []byte) *schemaTokenizer {
	return &schemaTokenizer{input: input, pos: 0}
}

func (r *schemaTokenizer) next() bool {
	r.skipWhitespace()
	if r.pos >= len(r.input) {
		return false
	}

	start := r.pos

	if r.input[r.pos] == '\'' {
		r.pos++
		for r.pos < len(r.input) && (r.input[r.pos] != '\'' ||
			(r.pos > start && r.input[r.pos-1] == '\\')) {
			r.pos++
		}
		r.pos++
	} else if r.input[r.pos] == '(' || r.input[r.pos] == ')' {
		r.pos++
	} else {
		for r.pos < len(r.input) &&
			!unicode.IsSpace(rune(r.input[r.pos])) &&
			r.input[r.pos] != '(' &&
			r.input[r.pos] != ')' {
			r.pos++
		}
	}

	r.cur = r.input[start:r.pos]
	return true
}

func (r *schemaTokenizer) this() []byte {
	return r.cur
}

// eq returns a Boolean indicative of the input
// byte slice being equal to the current token.
func (r *schemaTokenizer) eq(in []byte) bool {
	return bytes.Equal(r.cur, in)
}

func (r *schemaTokenizer) eqFold(in []byte) bool {
	return eqFoldASCII(r.cur, in)
}

func (r *schemaTokenizer) eqR(in rune) (is bool) {
	if in < 0x007F {
		is = r.eq([]byte{byte(in)})
	}

	return
}

func (r *schemaTokenizer) isFinalToken() bool {
	r.skipWhitespace()
	return r.pos >= len(r.input)
}

func (r *schemaTokenizer) nextToken() []byte {
	r.next()
	return r.cur
}

func (r *schemaTokenizer) skipWhitespace() {
	for r.pos < len(r.input) && unicode.IsSpace(rune(r.input[r.pos])) {
		r.pos++
	}
}

func stringDescrs(x [][]byte, delim []byte) (descrs []byte) {
	if len(x) == 1 {
		descrs = append(descrs, x[0]...)
	} else if len(x) > 1 {
		descrs = append(descrs, []byte("( ")...)
		for i := 0; i < len(x); i++ {
			descrs = append(descrs, x[i]...)
			if i < len(x)-1 {
				descrs = append(descrs, delim...)
			}
		}
		descrs = append(descrs, []byte(" )")...)
	}
	return
}

func stringQuotedDescrs(x [][]byte) (descrs []byte) {
	if len(x) == 1 {
		descrs = append(descrs, '\'')
		descrs = append(descrs, x[0]...)
		descrs = append(descrs, '\'')
	} else if len(x) > 1 {
		descrs = append(descrs, '(')
		for i := 0; i < len(x); i++ {
			descrs = append(descrs, ' ')
			descrs = append(descrs, '\'')
			descrs = append(descrs, x[i]...)
			descrs = append(descrs, '\'')
		}
		descrs = append(descrs, []byte(" )")...)
	}
	return
}

func stringBooleanClause(token string, b bool) (clause []byte) {
	if b {
		clause = append(clause, ' ')
		clause = append(clause, []byte(token)...)
	}
	return
}

func trimDefinitionLabelToken(input []byte) []byte {
	low := lc(input)
	for _, token := range headerTokens {
		if bytes.HasPrefix(low, lc([]byte(token))) {
			rest := input[len(token):]
			rest = bytes.TrimSpace(bytes.TrimLeft(rest, ":"))
			if idx := bytes.Index(rest, []byte("(")); idx != -1 {
				return rest
			}
		}
	}
	return input
}

func removeBashComments(input []byte) (output []byte) {
	stripComments := func(line []byte) []byte {
		re := regexp.MustCompile("#.*")
		return re.ReplaceAll(line, []byte{})
	}

	scanner := bufio.NewScanner(bytes.NewReader(input))
	for scanner.Scan() {
		line := scanner.Bytes()
		stripped := stripComments(line)
		if len(stripped) > 0 {
			output = append(output, stripped...)
			output = append(output, '\n')
		}
	}
	return
}

func condenseWHSP(b []byte) (a []byte) {
	b = bytes.TrimSpace(b)
	b = bytes.ReplaceAll(b, []byte{10}, []byte{32})

	var last bool
	for i := 0; i < len(b); i++ {
		c := b[i]
		switch c {
		case 9, 10, 32:
			if !last {
				last = true
				a = append(a, 32)
			}
		default:
			if last {
				last = false
			}
			a = append(a, c)
		}
	}

	a = bytes.TrimSpace(a)
	return
}

func eqFoldASCII(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca := a[i]
		cb := b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func removeWHSP(a []byte) []byte {
	return bytes.ReplaceAll(a, []byte(" "), []byte{})
}

func isAlnum(r rune) bool {
	return isAlpha(r) || isDigit(r)
}

func isAlpha(r rune) bool {
	return isLower(r) || isUpper(r)
}

func isLower(r rune) bool {
	return 'a' <= r && r <= 'z'
}

func isUpper(r rune) bool {
	return 'A' <= r && r <= 'Z'
}

func isDigit(r rune) bool {
	return '0' <= r && r <= '9'
}

func isUnsignedNumber(x []byte) bool {
	return isNumber(x) && !bytes.HasPrefix(x, []byte("-"))
}

func lc(in []byte) []byte {
	bld := &bytes.Buffer{}
	for i := 0; i < len(in); i++ {
		c := in[i]
		if c >= 'A' && c <= 'Z' {
			bld.WriteByte(c + 32)
		} else {
			bld.WriteByte(c)
		}
	}
	return bld.Bytes()
}

func isNumber(x []byte) bool {
	x = bytes.TrimLeft(x, "-")
	ct := 0
	for _, c := range x {
		if isDigit(rune(c)) {
			ct++
		}
	}
	return ct == len(x)
}

func isAttribute(val []byte) (is bool) {
	if is = isObjectIdentifier(val); !is {
		is = isAttributeDescriptor(val)
	}
	return
}

func checkDescriptors(descrs [][]byte, noid []byte) (err error) {
	for i := 0; i < len(descrs); i++ {
		if !isAttributeDescriptor(descrs[i]) {
			err = errors.New("Definition '" +
				string(noid) +
				"' bears malformed descriptor '" +
				string(descrs[i]) + "'")
			break
		}
	}
	return
}

func isObjectIdentifier(o []byte) bool {
	O := bytes.Split(o, []byte("."))
	if len(O) < 2 {
		return false
	}

	validArc := func(arc []byte) bool {
		if arc[0] == '-' {
			return false
		}
		if len(arc) > 1 && arc[0] == '0' {
			return false
		}
		for i := 0; i < len(arc); i++ {
			if !isDigit(rune(arc[i])) {
				return false
			}
		}
		return true
	}

	switch string(O[0]) {
	case "0", "1":
		if i, err := strconv.Atoi(string(O[1])); err != nil {
			return false
		} else if !(0 <= i && i <= 39) {
			return false
		}
	case "2":
	default:
		return false
	}

	for i := 1; i < len(O[1:]); i++ {
		if !validArc(O[i]) {
			return false
		}
	}

	return true
}

func isAttributeDescriptor(val []byte) bool {
	if len(val) == 0 {
		return false
	}
	if !isAlpha(rune(val[0])) {
		return false
	}
	if !isAlnum(rune(val[len(val)-1])) {
		return false
	}

	for i := 0; i < len(val); i++ {
		ch := rune(val[i])
		switch {
		case isAlnum(ch):
		case ch == ';', ch == '-':
		default:
			return false
		}
	}
	return true
}
