package schema

import (
	"bytes"
	"errors"
	"strings"
)

/*
DITContentRuleProperties implements a matrix of fast lookup tables
pertaining to registered DIT Content Rule definitions.
*/
type DITContentRuleProperties struct {
	MAP     map[string]uint             // rule numeric OID or nrml. descriptor (k) to # (v)
	ID      map[uint][]byte             // rule (k) to numeric OID
	PRINC   map[uint][]byte             // rule (k) to principal descriptor (v)
	TEXT    map[uint][]byte             // rule (k) bears text description (v)
	STRING  map[uint]string             // rule (k) in string representation (v)
	OBS     map[uint]struct{}           // rule (k) is obsolete
	AUX     map[uint]map[uint]struct{}  // rule (k) to auxiliary classes (k2)
	MUST    map[uint]map[uint]struct{}  // rule (k) to mandatory types (k2)
	MAY     map[uint]map[uint]struct{}  // rule (k) to optional types (k2)
	NOT     map[uint]map[uint]struct{}  // rule (k) to prohibited types (k2)
	DESCR   map[uint][][]byte           // rule (k) to principal descriptor (v)
	AUXORD  map[uint][]uint             // ordered collection of auxiliary object classes
	MUSTORD map[uint][]uint             // ordered collection of mandatory attribute types
	MAYORD  map[uint][]uint             // ordered collection of optional attribute types
	NOTORD  map[uint][]uint             // ordered collection of prohibited attribute types
	IDX     map[uint]int                // rule (k) resides at ORD slice N
	EXT     map[uint]map[uint]Extension // rule (k) bears extension(s) (k2, v)
	ORD     []uint                      // ordered collection of content rule OIDs

	schema *SubschemaSubentry
}

func initDITContentRuleProperties(sch *SubschemaSubentry) (dcp *DITContentRuleProperties) {
	dcp = &DITContentRuleProperties{
		MAP:     make(map[string]uint),
		ID:      make(map[uint][]byte),
		DESCR:   make(map[uint][][]byte),
		PRINC:   make(map[uint][]byte),
		STRING:  make(map[uint]string),
		TEXT:    make(map[uint][]byte),
		OBS:     make(map[uint]struct{}),
		AUXORD:  make(map[uint][]uint),
		MUSTORD: make(map[uint][]uint),
		MAYORD:  make(map[uint][]uint),
		NOTORD:  make(map[uint][]uint),
		IDX:     make(map[uint]int),
		AUX:     make(map[uint]map[uint]struct{}),
		MUST:    make(map[uint]map[uint]struct{}),
		MAY:     make(map[uint]map[uint]struct{}),
		NOT:     make(map[uint]map[uint]struct{}),
		EXT:     make(map[uint]map[uint]Extension),
		ORD:     make([]uint, 0),
		schema:  sch,
	}

	return
}

func (r *DITContentRuleProperties) marshal(in []byte, checkOnly bool) (err error) {
	input := bytes.TrimSpace(trimDefinitionLabelToken(in))
	tkz := newSchemaTokenizer(input)
	tkz.startTokenParen()

	noid := tkz.this()
	if !isObjectIdentifier(noid) {
		err = errDCRMalformedOID(noid)
		return
	}

	def := dITContentRuleDefinition{
		// use same uint as counterpart structural class
		noid: noid,
		ext:  make(map[uint]Extension),
	}

	seen := make(map[string]struct{})

	for tkz.next() && err == nil {
		token := string(tkz.this())
		if _, found := seen[token]; found {
			err = errors.New("DITContentRule clause '" + token +
				"' appears more than once")
			break
		}

		switch token {
		case ")":
			if tkz.isFinalToken() {
				err = r.commit(def, checkOnly)
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
		case "AUX":
			def.aux = parseMultiVal(tkz)
			err = r.checkMVClause(def.noid, `AUX`, def.aux)
			seen[token] = struct{}{}
		case "MUST":
			def.must = parseMultiVal(tkz)
			err = r.checkMVClause(def.noid, `MUST`, def.must)
			seen[token] = struct{}{}
		case "MAY":
			def.may = parseMultiVal(tkz)
			err = r.checkMVClause(def.noid, `MAY`, def.may)
			seen[token] = struct{}{}
		case "NOT":
			def.not = parseMultiVal(tkz)
			err = r.checkMVClause(def.noid, `NOT`, def.not)
			seen[token] = struct{}{}
		default:
			var ext Extension
			ext, err = marshalExtension(token, "dITContentRule", tkz)
			def.ext[uint(len(def.ext))] = ext
			seen[token] = struct{}{}
		}
	}

	return
}

func (r *DITContentRuleProperties) check(def *dITContentRuleDefinition) (err error) {
	for i := 0; i < len(def.descrs); i++ {
		item := lc(def.descrs[i])
		if reg, found := r.MAP[string(item)]; found {
			err = errDCRNotUnique(def.descrs[i], r.PRINC[reg])
			return
		}
	}

	class, _, _, classN := r.schema.oc.Resolve(def.noid)
	if class == nil {
		err = errDCRNoClass(def.noid)
		return
	} else if ooid, princ, _, _ := r.schema.dc.Resolve(def.noid); ooid != nil {
		err = errDCRNotUnique(princ, nil)
		return
	} else if k, _ := r.schema.oc.KIND[classN]; k != 0x0 {
		err = errDCRBadStrcKind(def.noid, k)
		return
	}
	def.noid = class
	def.n = classN

	var availableTypes map[uint]struct{}
	if availableTypes, err = r.gatherAvailableTypes(def); err == nil {
		err = r.resolveMMN(def, availableTypes)
	}

	return
}

func (r *DITContentRuleProperties) resolveMMN(
	def *dITContentRuleDefinition,
	avail map[uint]struct{},
) (err error) {

	seen := make(map[uint]struct{})
	for k, v := range map[string][][]byte{
		`MUST`: def.must,
		`MAY`:  def.may,
		`NOT`:  def.not,
	} {
		for i := 0; i < len(v); i++ {
			atoid, atdescr, _, atn := r.schema.at.Resolve(v[i])
			if atoid == nil {
				err = errDCRUnregType(def.noid, v[i], k)
				return
			}

			if _, found := seen[atn]; found && ErrorOnDuplicateClauseMember {
				err = errDCRDuplicateType(def.noid, atoid, atdescr, k)
				return
			}
			seen[atn] = struct{}{}

			if _, found := avail[atn]; !found && !def.exten {
				err = errDCRIneligibleType(def.noid, atdescr, k)
				return
			}

			switch k {
			case `MUST`:
				def.must[i] = atoid
			case `MAY`:
				def.may[i] = atoid
			case `NOT`:
				def.not[i] = atoid
			}
		}
	}

	return
}

// create an index of all available types for use in MUST/MAY/NOT.
func (r *DITContentRuleProperties) gatherAvailableTypes(
	def *dITContentRuleDefinition,
) (
	avail map[uint]struct{},
	err error,
) {
	avail = make(map[uint]struct{})

	// add structurally-provided types first
	for at := range r.schema.oc.MUST[def.n] {
		avail[at] = struct{}{}
	}

	for at := range r.schema.oc.MAY[def.n] {
		avail[at] = struct{}{}
	}

	// add auxiliary-provided types
	for i := 0; i < len(def.aux); i++ {
		auxc, _, _, auxn := r.schema.oc.Resolve(def.aux[i])
		if auxc == nil {
			err = errDCRUnregType(def.noid, def.aux[i], `AUX`)
			return
		} else if k, _ := r.schema.oc.KIND[auxn]; k != 0x1 {
			err = errDCRBadAuxKind(def.noid, k)
			return
		}

		// if we encounter the extensibleObject class,
		// then make a note of it. It will simplify
		// upcoming attribute type clearance checks.
		if string(auxc) == `1.3.6.1.4.1.1466.101.120.111` {
			def.exten = true
		}
		def.aux[i] = auxc

		for k := range r.schema.oc.MUST[auxn] {
			avail[k] = struct{}{}
		}

		for k := range r.schema.oc.MAY[auxn] {
			avail[k] = struct{}{}
		}
	}

	return
}

func (r DITContentRuleProperties) checkMVClause(
	noid []byte,
	name string,
	clause [][]byte,
) (err error) {
	for i := 0; i < len(clause); i++ {
		if !isAttribute(clause[i]) {
			err = errDCRMalformedType(noid, clause[i], name)
			break
		}
	}

	return
}

/*
Resolve returns a numeric OID, a principal descriptor and slices of all descriptors
associated with the input search term.

Valid input values are the numeric OID of the definition or any of its official
descriptors (names).

Case is not significant in the name matching process.
*/
func (r DITContentRuleProperties) Resolve(term []byte) (noid, princ []byte, descrs [][]byte, n uint) {
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

func (r DITContentRuleProperties) dITContentRuleDescription(x any) (result bool, err error) {
	var input []byte
	switch tv := x.(type) {
	case []byte:
		input = tv
	case string:
		input = []byte(tv)
	default:
		err = errorBadType("dITContentRule")
		return
	}

	err = r.marshal(input, true)
	result = err == nil
	return
}

type dITContentRuleDefinition struct {
	n      uint
	noid   []byte
	text   []byte
	descrs [][]byte
	aux    [][]byte
	must   [][]byte
	may    [][]byte
	not    [][]byte
	exten  bool
	obs    bool
	ext    map[uint]Extension
}

func (r *DITContentRuleProperties) makeString(n uint) (err error) {
	noid := r.ID[n]
	descrs := r.DESCR[n]

	bld := &strings.Builder{}
	bld.WriteRune('(')
	bld.WriteRune(' ')
	bld.Write(noid)
	bld.Write(definitionName(descrs))

	if text, found := r.TEXT[n]; found {
		bld.Write(definitionDescription(text))
	}

	_, obs := r.OBS[n]
	bld.Write(stringBooleanClause(`OBSOLETE`, obs))

	if aux, found := r.AUXORD[n]; found {
		var auxs [][]byte
		for i := 0; i < len(aux); i++ {
			princ := r.schema.oc.PRINC[aux[i]]
			auxs = append(auxs, princ)
		}
		bld.Write(definitionMVDescriptors(`AUX`, auxs))
	}

	if must, found := r.MUSTORD[n]; found {
		var musts [][]byte
		for i := 0; i < len(must); i++ {
			princ := r.schema.at.PRINC[must[i]]
			musts = append(musts, princ)
		}
		bld.Write(definitionMVDescriptors(`MUST`, musts))
	}

	if may, found := r.MAYORD[n]; found {
		var mays [][]byte
		for i := 0; i < len(may); i++ {
			princ := r.schema.at.PRINC[may[i]]
			mays = append(mays, princ)
		}
		bld.Write(definitionMVDescriptors(`MAY`, mays))
	}

	if not, found := r.NOTORD[n]; found {
		var nots [][]byte
		for i := 0; i < len(not); i++ {
			princ := r.schema.at.PRINC[not[i]]
			nots = append(nots, princ)
		}
		bld.Write(definitionMVDescriptors(`NOT`, nots))
	}

	if ext, found := r.EXT[n]; found {
		bld.Write(stringExtensions(ext))
	}

	bld.WriteRune(' ')
	bld.WriteRune(')')
	r.STRING[n] = bld.String()

	return
}

func (r *DITContentRuleProperties) commit(def dITContentRuleDefinition, checkOnly bool) (err error) {
	if checkOnly {
		return
	}

	if err = r.check(&def); err != nil {
		return
	}

	r.IDX[def.n] = len(r.ORD)
	r.ORD = append(r.ORD, def.n)
	r.ID[def.n] = def.noid
	r.MAP[string(def.noid)] = def.n

	r.commitDescr(def.n, def.noid, def.descrs)
	r.commitAux(def)
	r.commitMust(def)
	r.commitMay(def)
	r.commitNot(def)

	if def.text != nil {
		r.TEXT[def.n] = def.text
	}

	if def.obs {
		r.OBS[def.n] = struct{}{}
	}

	if len(def.ext) > 0 {
		r.EXT[def.n] = def.ext
	}

	r.makeString(def.n)

	return
}

func (r *DITContentRuleProperties) commitDescr(n uint, noid []byte, descrs [][]byte) error {
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

func (r *DITContentRuleProperties) commitAux(def dITContentRuleDefinition) {
	if lauxs := len(def.aux); lauxs > 0 {
		if _, found := r.AUX[def.n]; !found {
			r.AUX[def.n] = make(map[uint]struct{})
		}
		if _, found := r.AUXORD[def.n]; !found {
			r.AUXORD[def.n] = make([]uint, 0)
		}

		for i := 0; i < lauxs; i++ {
			if aoid, _, _, auxn := r.schema.oc.Resolve(def.aux[i]); aoid != nil {
				r.AUXORD[def.n] = append(r.AUXORD[def.n], auxn)
				r.schema.oc.DC[auxn] = def.n
				r.AUX[def.n][auxn] = struct{}{}
			}
		}
	}
}

func (r *DITContentRuleProperties) commitMust(def dITContentRuleDefinition) {
	if lmusts := len(def.must); lmusts > 0 {
		if _, found := r.MUST[def.n]; !found {
			r.MUST[def.n] = make(map[uint]struct{})
		}
		if _, found := r.MUSTORD[def.n]; !found {
			r.MUSTORD[def.n] = make([]uint, 0)
		}

		for i := 0; i < lmusts; i++ {
			_, _, _, mn := r.schema.at.Resolve(def.must[i])
			if _, found := r.schema.at.DC[mn]; !found {
				r.schema.at.DC[mn] = make(map[uint]struct{})
			}
			r.MUSTORD[def.n] = append(r.MUSTORD[def.n], mn)
			r.schema.at.DC[mn][def.n] = struct{}{}
			r.MUST[def.n][mn] = struct{}{}
		}
	}
}

func (r *DITContentRuleProperties) commitMay(def dITContentRuleDefinition) {
	if lmays := len(def.may); lmays > 0 {
		if _, found := r.MAY[def.n]; !found {
			r.MAY[def.n] = make(map[uint]struct{})
		}
		if _, found := r.MAYORD[def.n]; !found {
			r.MAYORD[def.n] = make([]uint, 0)
		}

		for i := 0; i < lmays; i++ {
			_, _, _, mn := r.schema.at.Resolve(def.may[i])
			if _, found := r.schema.at.DC[mn]; !found {
				r.schema.at.DC[mn] = make(map[uint]struct{})
			}
			r.MAYORD[def.n] = append(r.MAYORD[def.n], mn)
			r.schema.at.DC[mn][def.n] = struct{}{}
			r.MAY[def.n][mn] = struct{}{}
		}
	}
}

func (r *DITContentRuleProperties) commitNot(def dITContentRuleDefinition) {
	if lnots := len(def.not); lnots > 0 {
		if _, found := r.NOT[def.n]; !found {
			r.NOT[def.n] = make(map[uint]struct{})
		}
		if _, found := r.NOTORD[def.n]; !found {
			r.NOTORD[def.n] = make([]uint, 0)
		}

		for i := 0; i < lnots; i++ {
			_, _, _, mn := r.schema.at.Resolve(def.not[i])
			if _, found := r.schema.at.DC[mn]; !found {
				r.schema.at.DC[mn] = make(map[uint]struct{})
			}
			r.NOTORD[def.n] = append(r.NOTORD[def.n], mn)
			r.schema.at.DC[mn][def.n] = struct{}{}
			r.NOT[def.n][mn] = struct{}{}
		}
	}
}

func (r *DITContentRuleProperties) unregister(id []byte) (err error) {
	noid, _, descrs, n := r.Resolve(id)
	if noid == nil {
		return // no error needed
	}

	for i := 0; i < len(descrs); i++ {
		delete(r.MAP, string(lc(descrs[i])))
	}
	delete(r.MAP, string(noid))

	delete(r.MUSTORD, n)
	delete(r.MAYORD, n)
	delete(r.NOTORD, n)
	delete(r.AUXORD, n)
	delete(r.STRING, n)
	delete(r.DESCR, n)
	delete(r.PRINC, n)
	delete(r.TEXT, n)
	delete(r.MUST, n)
	delete(r.MAY, n)
	delete(r.NOT, n)
	delete(r.AUX, n)
	delete(r.OBS, n)
	delete(r.EXT, n)
	delete(r.ID, n)

	idx := r.IDX[n]
	r.ORD = append(r.ORD[:idx], r.ORD[idx+1:]...)
	r.IDX = make(map[uint]int)

	// rebuild indices map, as it would no longer
	// be accurate following this unregistration.
	for i := 0; i < len(r.ORD); i++ {
		r.IDX[r.ORD[i]] = i
	}

	return
}

/*
Get returns the pre-generated string representation of the definition
bearing the input id value. A zero string is returned if not found.

The id argument may be the numeric OID or the descriptor (name) of the
desired definition.

Case is not significant in the name matching process.
*/
func (r DITContentRuleProperties) Get(id []byte) string {
	var s string
	if noid, _, _, n := r.Resolve(id); noid != nil {
		s = r.STRING[n]
	}

	return s
}

/*
RegisterDITContentRule returns an error following an attempt to parse the input
def bytes. If successful, the components of def are written to the underlying
matrices.
*/
func (r *SubschemaSubentry) RegisterDITContentRule(def []byte) (err error) {
	if !bytes.HasPrefix(lc(def), lc([]byte(headerTokens[11]))) {
		def = append([]byte(headerTokens[11]+": "), def...)
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	err = r.dc.marshal(def, false)

	return
}

/*
UnregisterDITContentRule returns an error following an attempt to remove the
specified definition from the receiver instance. Valid input values are
the numeric OID or descriptor (name) of the desired definition.

Case is not significant in the name matching process.
*/
func (r *SubschemaSubentry) UnregisterDITContentRule(id []byte) (err error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	err = r.dc.unregister(id)
	return
}

func errDCRIneligibleType(noid, descr []byte, clause string) error {
	return errors.New("DITContentRule '" + string(noid) +
		"' bears a " + clause + " clause member '" + string(descr) +
		"' neither required nor allowed by any invoked class")
}

func errDCRUnregType(noid, descr []byte, clause string) error {
	return errors.New("DITContentRule '" + string(noid) +
		"' bears unregistered " + clause + " clause member '" +
		string(descr) + "'")
}

func errDCRMalformedType(noid, descr []byte, clause string) error {
	return errors.New("DITContentRule '" + string(noid) +
		"' bears malformed " + clause +
		" clause member '" + string(descr) + "'")
}

func errDCRBadAuxKind(noid []byte, kind uint8) error {
	return errors.New("DITContentRule '" + string(noid) +
		"' bears AUX clause member of an inappropriate kind '" +
		classKind(kind) + "'")
}

func errDCRBadStrcKind(noid []byte, kind uint8) error {
	return errors.New("DITContentRule '" + string(noid) +
		"' bears a class OID of inappropriate kind '" +
		classKind(kind) + "' (must be 'STRUCTURAL')")
}

func errDCRNotUnique(noid, other []byte) error {
	if other != nil {
		return errors.New("DITContentRule NAME '" + string(noid) +
			"' already registered to '" + string(other) + "'")
	}
	return errors.New("DITContentRule '" + string(noid) +
		"' already registered")
}

func errDCRNoClass(noid []byte) error {
	return errors.New("DITContentRule '" + string(noid) +
		"' bears an unregistered 'STRUCTURAL' class OID")
}

func errDCRMalformedOID(noid []byte) error {
	return errors.New("DITContentRule bears an malformed numeric OID '" +
		string(noid) + "'")
}

func errDCRDuplicateType(noid, atoid, atdescr []byte, clause string) error {
	return errors.New("DITContentRule '" + string(noid) +
		"' bears a duplicate " + clause + " clause member '" +
		string(atdescr) + "' (" + string(atoid) + ")")
}

func errDCRNotFound(id []byte) error {
	return errors.New("DITContentRule id '" +
		string(id) + "' not found")
}
