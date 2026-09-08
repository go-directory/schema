package schema

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
)

const (
	mrEqKind uint8 = iota + 1 // 1 EQUALITY
	mrSsKind                  // 2 SUBSTR
	mrOrKind                  // 3 ORDERING
)

func initMatchingRuleProperties(sch *SubschemaSubentry) (mrp *MatchingRuleProperties) {
	mrp = &MatchingRuleProperties{
		MAP:    make(map[string]uint),                             // string oid or nrml. text to #
		TEXT:   make(map[uint][]byte),                             // # to text
		ID:     make(map[uint][]byte),                             // # to oid
		DESCR:  make(map[uint][][]byte),                           // # to descriptors (names)
		PRINC:  make(map[uint][]byte),                             // # to principal descr
		OBS:    make(map[uint]struct{}),                           // # to btx
		SYNTAX: make(map[uint]uint),                               // # to syntax #
		EXT:    make(map[uint]map[uint]Extension),                 // # to extensions
		KIND:   make(map[uint]uint8),                              // # to mrXXKind constant (see above)
		STRING: make(map[uint]string),                             // # to string representation
		IDX:    make(map[uint]int),                                // # is Nth in ORD collection
		ORD:    make([]uint, 0),                                   // ordered collection of definition #s
		EQFUNC: make(map[uint]func(any, any) (bool, error)),       // # to equality matching func
		SSFUNC: make(map[uint]func(any, any) (bool, error)),       // # to substring matching func
		ORFUNC: make(map[uint]func(any, byte, any) (bool, error)), // # to ordering matching func
		schema: sch,
	}

	return
}

/*
MatchingRuleProperties implements a fast lookup matrix of properties related
to matching rules and matching rule relationships.
*/
type MatchingRuleProperties struct {
	MAP    map[string]uint                             // rule (k) nrml. name(s) and OID map to index (v)
	ID     map[uint][]byte                             // rule (k) bears numeric OID (v)
	TEXT   map[uint][]byte                             // rule (k) bears descriptive text (v)
	PRINC  map[uint][]byte                             // rule (k) to principal descriptor (v)
	DESCR  map[uint][][]byte                           // rule (k) bears descriptor(s) (v)
	SYNTAX map[uint]uint                               // rule (k) implements syntax (v)
	KIND   map[uint]uint8                              // rule (k) is what kind (v) [EQ:1|SS:2|OR:3]
	OBS    map[uint]struct{}                           // rule (k) is obsolete
	IDX    map[uint]int                                // rule (k) is Nth in ORD collection
	EXT    map[uint]map[uint]Extension                 // rule (k1) bears numbered (k2) extensions (v)
	STRING map[uint]string                             // rule (k) in string representation (v)
	EQFUNC map[uint]func(any, any) (bool, error)       // rule (k) extends EQUALITY func (v)
	SSFUNC map[uint]func(any, any) (bool, error)       // rule (k) extends SUBSTR func (v)
	ORFUNC map[uint]func(any, byte, any) (bool, error) // rule(k) extends ORDERING func (v)
	ORD    []uint                                      // collection of ordered rule OIDs, maps to IDX
	schema *SubschemaSubentry                          // internal reference
}

/*
Resolve returns a numeric OID, a principal descriptor and slices of all descriptors
associated with the input search term.

Valid input values are the numeric OID of the definition or any of its official
descriptors (names).

The return id value should not be read if the return noid value is zero length.

Case is not significant in the name matching process.
*/
func (r MatchingRuleProperties) Resolve(term []byte) (noid, princ []byte, descrs [][]byte, n uint) {
	if len(term) > 0 {
		var found bool
		if n, found = r.MAP[string(lc(term))]; found {
			noid, _ = r.ID[n]
			princ, _ = r.PRINC[n]
			descrs, _ = r.DESCR[n]
		}
	}

	return
}

/*
makeString returns an error following an attempt to create the string
representation of the specified definition.

If successful, the string value is written to the underlying receiver
STRING map.

If the input id (numeric OID) is not registered, an error is returned.
*/
func (r *MatchingRuleProperties) makeString(n uint) (err error) {
	noid := r.ID[n]
	descr := r.DESCR[n]

	bld := &strings.Builder{}
	bld.WriteRune('(')
	bld.WriteRune(' ')
	bld.Write(noid)
	bld.Write(definitionName(descr))

	if text, found := r.TEXT[n]; found {
		bld.Write(definitionDescription(text))
	}

	if _, found := r.OBS[n]; found {
		bld.WriteString(` OBSOLETE`)
	}

	bld.WriteString(` SYNTAX `)
	bld.Write(r.schema.ls.ID[r.SYNTAX[n]])

	if ext, found := r.EXT[n]; found {
		bld.Write(stringExtensions(ext))
	}

	bld.WriteRune(' ')
	bld.WriteRune(')')
	r.STRING[n] = bld.String()

	return
}

func deadEQ(noid []byte) func(any, any) (bool, error) {
	return func(_, _ any) (bool, error) {
		return false, errors.New("No EQUALITY function defined for '" +
			string(noid) + "'")
	}
}

func deadSS(noid []byte) func(any, any) (bool, error) {
	return func(_, _ any) (bool, error) {
		return false, errors.New("No SUBSTR function defined for '" +
			string(noid) + "'")
	}
}

func deadOR(noid []byte) func(any, byte, any) (bool, error) {
	return func(_ any, _ byte, _ any) (bool, error) {
		return false, errors.New("No ORDERING function defined for '" +
			string(noid) + "'")
	}
}

func kindOfMatchingRule(descr []byte) (kind uint8) {
	d := lc(descr)
	switch {
	case bytes.Contains(d, []byte("substring")):
		kind = mrSsKind
	case bytes.Contains(d, []byte("ordering")):
		kind = mrOrKind
	case bytes.HasSuffix(d, []byte("match")):
		kind = mrEqKind
	}

	return
}

type matchingRuleDefinition struct {
	n      uint
	syntax []byte
	noid   []byte
	text   []byte
	descrs [][]byte
	obs    bool
	ext    map[uint]Extension
}

func (r *MatchingRuleProperties) commit(def matchingRuleDefinition) error {
	r.MAP[string(def.noid)] = def.n
	r.ID[def.n] = def.noid

	r.commitDescr(def.n, def.noid, def.descrs)

	r.KIND[def.n] = kindOfMatchingRule(r.PRINC[def.n])
	if def.text != nil {
		r.TEXT[def.n] = def.text
	}

	syn, _, synid := r.schema.ls.Resolve(def.syntax)
	if syn == nil {
		return errors.New("MatchingRule '" + string(def.noid) +
			"' bears unregistered SYNTAX '" +
			string(def.syntax) + "'")
	}
	r.SYNTAX[def.n] = synid
	if _, found := r.schema.ls.MR[synid]; !found {
		r.schema.ls.MR[synid] = make(map[uint]struct{})
	}
	r.schema.ls.MR[synid][def.n] = struct{}{}

	if def.obs {
		r.OBS[def.n] = struct{}{}
	}

	r.EXT[def.n] = def.ext
	r.IDX[def.n] = len(r.ORD)
	r.ORD = append(r.ORD, def.n)

	// put a placeholder matching rule function
	// in place for now; if it is never populated
	// with a real matcher, the end-user will get
	// a "not implemented" error as opposed to a
	// panic.
	switch r.KIND[def.n] {
	case mrEqKind:
		r.EQFUNC[def.n] = deadEQ(def.noid)
	case mrSsKind:
		r.SSFUNC[def.n] = deadSS(def.noid)
	case mrOrKind:
		r.ORFUNC[def.n] = deadOR(def.noid)
	}

	r.makeString(def.n)

	return nil
}

func (r *MatchingRuleProperties) commitDescr(n uint, noid []byte, descrs [][]byte) error {
	if len(descrs) > 0 {
		for i := 0; i < len(descrs); i++ {
			r.MAP[string(lc(descrs[i]))] = n
		}
		r.PRINC[n] = descrs[0]
		r.DESCR[n] = descrs
	} else {
		r.PRINC[n] = noid
	}

	return nil
}

func (r *MatchingRuleProperties) unregister(x []byte) (err error) {

	noid, _, descrs, n := r.Resolve(x)
	if noid == nil {
		return // no error needed
	}

	// first check if any dependencies exist
	// that should preclude unregistration.
	if _, found := r.schema.mu.APPORD[n]; found {
		appl, _ := r.schema.mu.APPORD[n]
		if L := len(appl); L > 0 {
			err = errors.New("MatchingRule '" + string(noid) +
				"' is depended upon by " + strconv.Itoa(L) +
				"attribute types")
		}
		return
	}

	delete(r.MAP, string(noid))
	for i := 0; i < len(descrs); i++ {
		delete(r.MAP, string(lc(descrs[i])))
	}

	delete(r.STRING, n)
	delete(r.schema.ls.MR[r.SYNTAX[n]], n)
	delete(r.SYNTAX, n)
	delete(r.PRINC, n)
	delete(r.KIND, n)
	delete(r.EQFUNC, n)
	delete(r.SSFUNC, n)
	delete(r.ORFUNC, n)
	delete(r.TEXT, n)
	delete(r.EXT, n)
	delete(r.OBS, n)

	idx := r.IDX[n]
	r.ORD = append(r.ORD[:idx], r.ORD[idx+1:]...)

	// rebuild indices map, since it will no longer
	// be accurate following this unregistration.
	r.IDX = make(map[uint]int)
	for i := 0; i < len(r.ORD); i++ {
		r.IDX[uint(r.ORD[i])] = i
	}

	// delete counterpart mru
	delete(r.schema.mu.APPLIES, n)
	delete(r.schema.mu.APPORD, n)

	return
}

func (r *MatchingRuleProperties) marshal(in []byte, checkOnly bool) (err error) {
	if r.schema.ls == nil || r.schema.mr == nil {
		err = errors.New("Schema not initialized")
		return
	}

	input := bytes.TrimSpace(trimDefinitionLabelToken(in))
	tkz := newSchemaTokenizer(input)
	tkz.startTokenParen()

	noid := tkz.this()
	if mr, _, _, _ := r.Resolve(noid); mr != nil && !checkOnly {
		err = errors.New("MatchingRule OID '" + string(noid) +
			"' already registered")
		return
	}

	def := matchingRuleDefinition{
		n:    uint(len(r.ORD)),
		noid: noid,
		ext:  make(map[uint]Extension),
	}

	seen := make(map[string]struct{})

	for tkz.next() && err == nil {
		token := string(tkz.this())
		if _, found := seen[token]; found {
			err = errors.New("MatchingRule clause '" + token +
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
		case "NAME":
			def.descrs = parseMultiVal(tkz)
			err = checkDescriptors(def.descrs, def.noid)
			seen[token] = struct{}{}
		case "DESC":
			def.text = parseSingleVal(tkz)
			seen[token] = struct{}{}
		case "OBSOLETE":
			def.obs = true
			seen[token] = struct{}{}
		case "SYNTAX":
			def.syntax = tkz.nextToken()
			err = r.checkSyntaxClause(def.noid, def.syntax)
			seen[token] = struct{}{}
		default:
			var ext Extension
			ext, err = marshalExtension(token, "matchingRule", tkz)
			def.ext[uint(len(def.ext))] = ext
			seen[token] = struct{}{}
		}
	}

	return
}

/*
func (r *MatchingRuleProperties) checkDescriptor(
	descrs [][]byte,
	noid []byte,
	checkOnly bool,
) (err error) {
	if err = checkDescriptors(descrs, noid); err != nil {
		return
	}
	if checkOnly {
		return
	}
	for i := 0; i < len(descrs); i++ {
		item := string(lc(descrs[i]))
		if n, found := r.MAP[item]; found {
			err = errors.New("MatchingRule NAME '" + string(descrs[i]) +
				"' already registered to '" + string(r.PRINC[n]) + "'")
			break
		}
	}

	return
}
*/

func (r *MatchingRuleProperties) checkSyntaxClause(
	noid, clause []byte,
) (err error) {
	if !isObjectIdentifier(clause) {
		err = errors.New("MatchingRule '" + string(noid) +
			"' bears malformed SYNTAX '" + string(clause) + "'")
		return
	}

	return

}

func (r *MatchingRuleProperties) matchingRuleDescription(x any) (result bool, err error) {
	var input []byte
	switch tv := x.(type) {
	case []byte:
		input = tv
	case string:
		input = []byte(tv)
	default:
		err = errorBadType("matchingRuleUse")
		return
	}

	err = r.marshal(input, true)
	result = err == nil
	return
}

/*
Get returns the pre-generated string representation of the definition
bearing the input id value. A zero string is returned if not found.

The id argument may be the numeric OID or the descriptor (name) of the
desired definition.

Case is not significant in the name matching process.
*/
func (r MatchingRuleProperties) Get(id []byte) string {
	var s string
	if noid, _, _, n := r.Resolve(id); noid != nil {
		s = r.STRING[n]
	}

	return s
}

/*
RegisterMatchingRule returns an error following an attempt to parse the input
def bytes. If successful, the components of def are written to the underlying
matrices.
*/
func (r *SubschemaSubentry) RegisterMatchingRule(def []byte) (err error) {
	if !bytes.HasPrefix(lc(def), lc([]byte(headerTokens[3]))) {
		def = append([]byte(headerTokens[3]+": "), def...)
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	err = r.mr.marshal(def, false)

	return
}

/*
UnregisterMatchingRule returns an error following an attempt to remove the
specified definition from the receiver instance. Valid input values are
the numeric OID or descriptor (name) of the desired definition.

Case is not significant in the name matching process.

Note that it is highly unusual for there to be a legitimate need to delete
a matching rule from the schema. Matching rules are fundamental to the act
of searching for entries in a directory.
*/
func (r *SubschemaSubentry) UnregisterMatchingRule(id []byte) (err error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	err = r.mr.unregister(id)
	return
}
