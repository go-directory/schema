package schema

import (
	"bytes"
	"errors"
	"regexp"
	"strconv"
	"strings"
	//"sync"
)

func initLDAPSyntaxProperties(sch *SubschemaSubentry) (lsp *LDAPSyntaxProperties) {
	lsp = &LDAPSyntaxProperties{
		MAP:    make(map[string]uint),                  // string oid to #
		ID:     make(map[uint][]byte),                  // # to oid
		TEXT:   make(map[uint][]byte),                  // # to oid
		PRINC:  make(map[uint][]byte),                  // # to principal descr
		NHR:    make(map[uint]struct{}),                // # to nhr
		BTX:    make(map[uint]struct{}),                // # to btx
		AT:     make(map[uint]map[uint]struct{}),       // # to at deps
		MR:     make(map[uint]map[uint]struct{}),       // # to mr deps
		EXT:    make(map[uint]map[uint]Extension),      // # to extensions
		STRING: make(map[uint]string),                  // # to string representation
		IDX:    make(map[uint]int),                     // # is Nth in ORD collection
		ORD:    make([]uint, 0),                        // ordered collection of definition #s
		FUNC:   make(map[uint]func(any) (bool, error)), // # to syntax matching func
		schema: sch,
	}

	return
}

/*
LDAPSyntaxProperties implements a fast lookup matrix of properties related
to syntaxes and syntax relationships.
*/
type LDAPSyntaxProperties struct {
	MAP    map[string]uint                  // string oid or nrml. text to #
	TEXT   map[uint][]byte                  // # to text
	ID     map[uint][]byte                  // # to oid
	PRINC  map[uint][]byte                  // # to principal descr
	NHR    map[uint]struct{}                // # to nhr
	BTX    map[uint]struct{}                // # to btx
	AT     map[uint]map[uint]struct{}       // # to at deps
	MR     map[uint]map[uint]struct{}       // # to mr deps
	EXT    map[uint]map[uint]Extension      // # to extensions
	STRING map[uint]string                  // # to string representation
	IDX    map[uint]int                     // # is Nth in ORD collection
	ORD    []uint                           // ordered collection of definition #s
	FUNC   map[uint]func(any) (bool, error) // # to syntax verification func
	schema *SubschemaSubentry               // internal reference
}

/*
Resolve returns a numeric OID and a text description associated with the input
search term.

Valid input values are the numeric OID of the definition or its official text
description.

The id return value should not be read if the return noid value is zero length.

Case is not significant in the text matching process.
*/
func (r LDAPSyntaxProperties) Resolve(term []byte) (noid, description []byte, n uint) {
	if len(term) > 0 {
		var found bool
		if n, found = r.MAP[string(lc(term))]; found {
			noid, _ = r.ID[n]
			description, _ = r.TEXT[n]
		}
	}

	return
}

/*
makeString returns an error following an attempt to create the string
representation of the specified definition.

If successful, the string value is written to the underlying receiver
STRING map.

If the input index is not registered, an error is returned.
*/
func (r *LDAPSyntaxProperties) makeString(n uint) (err error) {
	noid := r.ID[n]
	text := r.TEXT[n]

	bld := &strings.Builder{}
	bld.WriteRune('(')
	bld.WriteRune(' ')
	bld.Write(noid)
	bld.Write(definitionDescription(text))

	if ext, _ := r.EXT[n]; ext != nil {
		bld.Write(stringExtensions(ext))
	}

	bld.WriteRune(' ')
	bld.WriteRune(')')
	r.STRING[n] = bld.String()

	return
}

/*
Get returns the pre-generated string representation of the definition
bearing the input id value. A zero string is returned if not found.

The id argument may be the numeric OID or descriptive text of the
desired definition.

Case is not significant in the text matching process.
*/
func (r LDAPSyntaxProperties) Get(id []byte) string {
	var s string
	if noid, _, n := r.Resolve(id); noid != nil {
		s = r.STRING[n]
	}

	return s
}

/*
HR returns a Boolean value indicative of the input id argument
being associated with a human-readable LDAP syntax. A value of
false is returned if the definition is not found.
*/
func (r LDAPSyntaxProperties) HR(id []byte) bool {
	var b bool
	if noid, _, n := r.Resolve(id); noid != nil {
		b = true // assume true for the moment.
		if _, nhr := r.NHR[n]; nhr {
			// if found, syntax is NOT human readable.
			b = false
		}
	}

	return b
}

/*
BinaryTxReq returns a Boolean value indicative of the specified syntax
id being .
*/
func (r LDAPSyntaxProperties) BinaryTxReq(id []byte) bool {
	var b bool
	noid, _, n := r.Resolve(id)
	if noid == nil {
		return b
	}

	if _, found := r.EXT[n]; found {
		for _, v := range r.EXT[n] {
			if v.boolValue([]byte(`X-BINARY-TRANSFER-REQUIRED`)) {
				b = true
				break
			}
		}
	}

	return b
}

func (r *LDAPSyntaxProperties) lDAPSyntaxDescription(x any) (result bool, err error) {
	var input []byte

	switch tv := x.(type) {
	case []byte:
		input = tv
	case string:
		input = []byte(tv)
	default:
		err = errorBadType("ldapSyntax")
		return
	}

	err = r.marshal(input, true)
	result = err == nil
	return
}

func (r *LDAPSyntaxProperties) marshal(in []byte, checkOnly bool) (err error) {
	input := bytes.TrimSpace(trimDefinitionLabelToken(in))
	tkz := newSchemaTokenizer(input)
	if tkz.next() && tkz.eqR('(') {
		tkz.next()
	}

	noid := tkz.this()
	if ls, _, _ := r.Resolve(noid); ls != nil && !checkOnly {
		err = errors.New("LDAPSyntax OID '" + string(noid) +
			"' already registered")
		return
	}

	def := ldapSyntaxDefinition{
		n:    uint(len(r.ORD)),
		noid: noid,
		ext:  make(map[uint]Extension),
	}

	seen := make(map[string]struct{})

	for tkz.next() && err == nil {
		token := string(tkz.this())
		if _, found := seen[token]; found {
			err = errors.New("LDAPSyntax clause '" + token +
				"' appears more than once")
			break
		}

		switch token {
		case ")":
			if tkz.isFinalToken() {
				if !checkOnly {
					err = r.commit(def)
				}
				return
			}
		case "DESC":
			def.text = parseSingleVal(tkz)
			seen[token] = struct{}{}
		default:
			var ext Extension
			ext, err = marshalExtension(token, "ldapSyntax", tkz)
			def.ext[uint(len(def.ext))] = ext
			seen[token] = struct{}{}
		}
	}

	return
}

type ldapSyntaxDefinition struct {
	n    uint
	noid []byte
	text []byte
	ext  map[uint]Extension
}

func (r *LDAPSyntaxProperties) commit(def ldapSyntaxDefinition) error {
	if !r.isHR(def.n) {
		r.NHR[def.n] = struct{}{}
	}

	if r.binTxReq(def.n) {
		r.BTX[def.n] = struct{}{}
	}

	r.MAP[string(def.noid)] = def.n
	r.MAP[string(lc(def.text))] = def.n

	if def.text != nil {
		r.TEXT[def.n] = def.text
	} else {
		r.TEXT[def.n] = []byte("Undocumented Syntax")
	}

	r.ID[def.n] = def.noid
	r.PRINC[def.n] = def.text

	r.ORD = append(r.ORD, def.n)
	r.EXT[def.n] = def.ext
	r.makeString(def.n)

	// add special syntax checkers for RFC4512 definition types
	switch string(def.noid) {
	case OIDLDAPSyntaxDescription:
		r.FUNC[def.n] = r.lDAPSyntaxDescription
	case OIDMatchingRuleDescription:
		r.FUNC[def.n] = r.schema.mr.matchingRuleDescription
	case OIDAttributeTypeDescription:
		r.FUNC[def.n] = r.schema.at.attributeTypeDescription
	case OIDObjectClassDescription:
		r.FUNC[def.n] = r.schema.oc.objectClassDescription
	case OIDDITContentRuleDescription:
		r.FUNC[def.n] = r.schema.dc.dITContentRuleDescription
	case OIDNameFormDescription:
		r.FUNC[def.n] = r.schema.nf.nameFormDescription
	case OIDDITStructureRuleDescription:
		r.FUNC[def.n] = r.schema.ds.dITStructureRuleDescription
	default:
		// For everything else, provide dead syntax
		// verification placeholder for now. If the
		// definition bears an X-PATTERN signature,
		// use that instead.
		r.FUNC[def.n] = r.syntaxHandler(def.n, def.noid, def.text)
	}

	return nil
}

func (r *LDAPSyntaxProperties) syntaxHandler(
	n uint,
	noid, text []byte,
) (
	funk func(any) (bool, error),
) {
	// set a dead syntax verification
	// placeholder for the moment.
	funk = func(_ any) (bool, error) {
		return false, errors.New("LDAPSyntax " +
			string(noid) + " (" + string(text) +
			") not implemented")
	}

	if _, ok := r.EXT[n]; ok {
		for _, v := range r.EXT[n] {
			if eqFoldASCII(v.XString, []byte("X-PATTERN")) &&
				len(v.Values) > 0 {
				// Found an X-PATTERN signature, so make it
				// into a handler and return it.
				funk = func(x any) (result bool, err error) {
					var val string
					switch tv := x.(type) {
					case string:
						val = tv
					case []byte:
						val = string(tv)
					default:
						return false, nil
					}
					pat := string(v.Values[0])
					result, err = regexp.MatchString(pat, val)
					return
				}
				break
			}
		}
	}

	return
}

func (r *LDAPSyntaxProperties) unregister(x []byte) (err error) {
	n, found := r.MAP[string(lc(x))]
	if !found {
		return // no error needed
	}

	id := r.ID[n]
	idx := r.IDX[n]
	text := r.TEXT[n]

	checkATMRDep := func(col map[uint]struct{}, typ string) (err error) {
		if L := len(col); L > 0 {
			err = errors.New("LDAPSyntax is a depended upon by " +
				strconv.Itoa(L) + typ)
		}
		return
	}

	// first check if any dependencies exist
	// that should preclude unregistration.
	if err = checkATMRDep(r.MR[n], " matching rules"); err != nil {
		return
	} else if err = checkATMRDep(r.AT[n], " attribute types"); err != nil {
		return
	}

	r.ORD = append(r.ORD[:idx], r.ORD[idx+1:]...)
	delete(r.AT, n)
	delete(r.MR, n)
	delete(r.ID, n)
	delete(r.MAP, string(id))
	delete(r.MAP, string(lc(text)))
	delete(r.BTX, n)
	delete(r.NHR, n)
	delete(r.EXT, n)
	delete(r.TEXT, n)
	delete(r.FUNC, n)
	delete(r.PRINC, n)
	delete(r.STRING, n)

	// rebuild indices
	r.IDX = make(map[uint]int)
	for i := 0; i < len(r.ORD); i++ {
		r.IDX[r.ORD[i]] = i
	}

	return
}

func (r LDAPSyntaxProperties) binTxReq(id uint) (btx bool) {
	ct, it := uint(0), uint(0)
	ext := r.EXT[id]
	l := uint(len(ext))
	for it = 0; ct < l && !btx; it++ {
		if m, ok := ext[it]; ok {
			ct++
			btx = m.boolValue([]byte(`X-BINARY-TRANSFER-REQUIRED`))
		}
	}

	return
}

func (r LDAPSyntaxProperties) isHR(id uint) (hr bool) {
	hr = true // assume human readable by default
	ct, it := uint(0), uint(0)
	ext := r.EXT[id]
	l := uint(len(ext))
	for it = 0; ct < l && hr; it++ {
		if m, ok := ext[it]; ok {
			ct++
			hr = !m.boolValue([]byte(`X-NOT-HUMAN-READABLE`))
		}
	}

	return
}

/*
RegisterLDAPSyntax returns an error following an attempt to parse the input
def bytes. If successful, the components of def are written to the underlying
matrices.
*/
func (r *SubschemaSubentry) RegisterLDAPSyntax(def []byte) (err error) {
	if !bytes.HasPrefix(lc(def), lc([]byte(headerTokens[1]))) {
		def = append([]byte(headerTokens[1]+": "), def...)
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	err = r.ls.marshal(def, false)

	return
}

/*
UnregisterLDAPSyntax returns an error following an attempt to remove the
specified definition from the receiver instance. Valid input values are
the numeric OID or descriptive text of the desired definition.

Case is not significant in the text matching process.

Note that it is highly unusual for there to be a legitimate need to delete
an LDAP syntax from the schema. Syntaxes are fundamental to the operation
of attribute types and matching rules.
*/
func (r *SubschemaSubentry) UnregisterLDAPSyntax(id []byte) (err error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	err = r.ls.unregister(id)
	return
}
