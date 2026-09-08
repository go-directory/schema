package schema

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
)

type ObjectClassProperties struct {
	MAP     map[string]uint             // numeric OID or nrml. descriptor (k) to # (v)
	ID      map[uint][]byte             // class (k) to numeric OID (v)
	DESCR   map[uint][][]byte           // class (k) to descriptor(s) (v)
	SUPCORD map[uint][]uint             // ordered collection of super classes
	MUSTORD map[uint][]uint             // ordered collection of mandatory types
	MAYORD  map[uint][]uint             // ordered collection of optional types
	CHAINS  map[uint][][]uint           // class (k) is the base member of each super class path
	NF      map[uint]map[uint]struct{}  // class (k1) to dependent name forms (k2)
	SUPC    map[uint]map[uint]struct{}  // class (k1) to super classes (k2)
	SUBC    map[uint]map[uint]struct{}  // class (k1) to sub classes (k2)
	MUST    map[uint]map[uint]struct{}  // class (k1) to mandatory local types (k2)
	ALLMUST map[uint]map[uint]struct{}  // class (k1) to mandatory global types (k2)
	MAY     map[uint]map[uint]struct{}  // class (k1) to optional local types (k2)
	ALLMAY  map[uint]map[uint]struct{}  // class (k1) to optional global types (k2)
	EXT     map[uint]map[uint]Extension // class (k1) bears numbered (k2) extensions (v)
	OBS     map[uint]struct{}           // class (k) is obsolete
	PRIV    map[uint]struct{}           // class (k) is private
	KIND    map[uint]uint8              // class (k) to kind (v) [0=STRC/1=AUX/2=ABS]
	PRINC   map[uint][]byte             // class (k) to principal descriptor (v)
	TEXT    map[uint][]byte             // class (k) bears descriptive text (v)
	STRING  map[uint]string             // class (k) in string representation (v)
	DC      map[uint]uint               // class (k) to dependent content rule (v)
	IDX     map[uint]int                // class (k) to integer index (v)
	ORD     []uint                      // ordered collection of all classes

	schema *SubschemaSubentry // internal reference
}

func initObjectClassProperties(sch *SubschemaSubentry) (ocp *ObjectClassProperties) {
	ocp = &ObjectClassProperties{
		MAP:     make(map[string]uint),
		ID:      make(map[uint][]byte),
		DESCR:   make(map[uint][][]byte),
		DC:      make(map[uint]uint),
		PRINC:   make(map[uint][]byte),
		TEXT:    make(map[uint][]byte),
		STRING:  make(map[uint]string),
		MUSTORD: make(map[uint][]uint),
		MAYORD:  make(map[uint][]uint),
		SUPCORD: make(map[uint][]uint),
		CHAINS:  make(map[uint][][]uint),
		NF:      make(map[uint]map[uint]struct{}),
		SUPC:    make(map[uint]map[uint]struct{}),
		SUBC:    make(map[uint]map[uint]struct{}),
		MUST:    make(map[uint]map[uint]struct{}),
		MAY:     make(map[uint]map[uint]struct{}),
		ALLMUST: make(map[uint]map[uint]struct{}),
		ALLMAY:  make(map[uint]map[uint]struct{}),
		EXT:     make(map[uint]map[uint]Extension),
		OBS:     make(map[uint]struct{}),
		KIND:    make(map[uint]uint8),
		IDX:     make(map[uint]int),
		ORD:     make([]uint, 0),
		schema:  sch,
	}

	return
}

func (r ObjectClassProperties) objectClassDescription(x any) (result bool, err error) {
	var input []byte
	switch tv := x.(type) {
	case []byte:
		input = tv
	case string:
		input = []byte(tv)
	default:
		err = errorBadType("objectClass")
		return
	}

	err = r.marshal(input, true)
	result = err == nil
	return
}

type objectClassDefinition struct {
	n      uint
	noid   []byte
	descrs [][]byte
	text   []byte
	kind   uint8
	supers [][]byte
	subs   [][]byte
	amust  [][]byte
	must   [][]byte
	amay   [][]byte
	may    [][]byte
	obs    bool
	ext    map[uint]Extension
}

func (r *ObjectClassProperties) marshal(in []byte, checkOnly bool) (err error) {
	input := bytes.TrimSpace(trimDefinitionLabelToken(in))
	tkz := newSchemaTokenizer(input)
	tkz.startTokenParen()

	noid := tkz.this()
	if _, found := r.MAP[string(noid)]; found && !checkOnly {
		err = errOCNotUnique(noid)
		return
	}

	def := objectClassDefinition{
		n:    uint(len(r.ORD)),
		noid: noid,
		ext:  make(map[uint]Extension),
	}

	seen := make(map[string]struct{})

	for tkz.next() && err == nil {
		token := string(tkz.this())
		if _, found := seen[token]; found {
			err = errors.New("ObjectClass clause '" + token +
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
		case "STRUCTURAL", "AUXILIARY", "ABSTRACT":
			def.kind, err = parseClassKind(token)
			seen[token] = struct{}{}
		case "OBSOLETE":
			def.obs = true
			seen[token] = struct{}{}
		case "SUP":
			def.supers = parseMultiVal(tkz)
			err = r.checkSupClause(def.noid, def.supers)
			seen[token] = struct{}{}
		case "MUST":
			def.must = parseMultiVal(tkz)
			err = r.checkMustClause(def.noid, def.must)
			seen[token] = struct{}{}
		case "MAY":
			def.may = parseMultiVal(tkz)
			err = r.checkMayClause(def.noid, def.may)
			seen[token] = struct{}{}
		default:
			var ext Extension
			ext, err = marshalExtension(token, "objectClass", tkz)
			def.ext[uint(len(def.ext))] = ext
			seen[token] = struct{}{}
		}
	}

	return
}

func (r ObjectClassProperties) buildChains(id uint) [][]uint {
	if _, found := r.ID[id]; !found {
		return nil
	}

	parents := r.SUPCORD[id]
	if len(parents) == 0 {
		return [][]uint{{id}}
	}

	var out [][]uint

	for _, p := range parents {
		sub := r.buildChains(p)
		for _, path := range sub {
			cp := make([]uint, 0, len(path)+1)
			cp = append(cp, id)
			cp = append(cp, path...)
			out = append(out, cp)
		}
	}

	return out
}

func zeroChain(chains [][]uint) bool {
	return len(chains) == 1 && len(chains[0]) == 0
}

func (r *ObjectClassProperties) buildAllMust(id uint) (out map[uint]struct{}) {
	_, found := r.ID[id]
	if !found {
		out = map[uint]struct{}{}
		return
	}

	out = make(map[uint]struct{})
	visited := make(map[uint]struct{})

	var dfs func(uint)
	dfs = func(n uint) {
		if _, ok := visited[n]; ok {
			return
		}
		visited[n] = struct{}{}

		// local MUST
		if m, found := r.MUST[n]; found {
			for x := range m {
				out[x] = struct{}{}
			}
		}

		// inherited MUST via chains
		if chains, found := r.CHAINS[n]; found {
			for _, path := range chains {
				for _, super := range path {
					dfs(super)
				}
			}
		}
	}

	dfs(id)
	return
}

func (r *ObjectClassProperties) buildAllMay(id uint) (out map[uint]struct{}) {
	_, found := r.ID[id]
	if !found {
		return
	}

	out = make(map[uint]struct{})
	visited := make(map[uint]struct{})

	var dfs func(uint)
	dfs = func(n uint) {
		if _, ok := visited[n]; ok {
			return
		}
		visited[n] = struct{}{}

		// local MUST
		if m, found := r.MAY[n]; found {
			for x := range m {
				out[x] = struct{}{}
			}
		}

		// inherited MAY via chains
		if chains, found := r.CHAINS[n]; found {
			for _, path := range chains {
				for _, super := range path {
					dfs(super)
				}
			}
		}
	}

	dfs(id)
	return
}

func (r ObjectClassProperties) checkSupClause(
	noid []byte,
	clause [][]byte,
) (err error) {
	for i := 0; i < len(clause); i++ {
		if !isAttribute(clause[i]) {
			err = errOCMalformedType(noid, clause[i], `SUP`)
			break
		}
	}

	return
}

func (r ObjectClassProperties) checkMustClause(
	noid []byte,
	clause [][]byte,
) (err error) {
	for i := 0; i < len(clause); i++ {
		if !isAttribute(clause[i]) {
			err = errOCMalformedType(noid, clause[i], `MUST`)
			break
		}
	}

	return
}

func (r ObjectClassProperties) checkMayClause(
	noid []byte,
	clause [][]byte,
) (err error) {
	for i := 0; i < len(clause); i++ {
		if !isAttribute(clause[i]) {
			err = errOCMalformedType(noid, clause[i], `MAY`)
			break
		}
	}

	return
}

func parseClassKind(token string) (kind uint8, err error) {
	switch token {
	case `STRUCTURAL`:
	case `AUXILIARY`:
		kind = uint8(1)
	case `ABSTRACT`:
		kind = uint8(2)
	default:
		err = errOCIllegalKind(token)
	}
	return
}

func classKind(kind uint8) (k string) {
	k = ` STRUCTURAL`
	if kind == 1 {
		k = ` AUXILIARY`
	} else if kind == 2 {
		k = ` ABSTRACT`
	}
	return
}

func (r *ObjectClassProperties) check(def *objectClassDefinition) (err error) {
	for i := 0; i < len(def.descrs); i++ {
		item := lc(def.descrs[i])
		if reg, found := r.MAP[string(item)]; found {
			err = errOCNameReg(def.noid, r.PRINC[reg])
			return
		}
	}

	// we do separate seen tables here to
	// account for certain classes which
	// foolishly put a particular attribute
	// in both MUST and MAY.
	seenMust := make(map[uint]struct{})
	seenMay := make(map[uint]struct{})

	if err = r.checkSup(def); err == nil {
		for i := 0; i < len(def.must); i++ {
			atoid, atdescr, _, atn := r.schema.at.Resolve(def.must[i])
			if atoid == nil {
				err = errOCUnregType(def.noid, def.must[i], `MUST`)
				return
			}
			if _, found := seenMust[atn]; found && ErrorOnDuplicateClauseMember {
				err = errOCDuplicateType(def.noid, atoid, atdescr, `MUST`)
				return
			}
			seenMust[atn] = struct{}{}
			def.must[i] = atoid
		}

		for i := 0; i < len(def.may); i++ {
			atoid, atdescr, _, atn := r.schema.at.Resolve(def.may[i])
			if atoid == nil {
				err = errOCUnregType(def.noid, def.may[i], `MAY`)
				return
			}
			if _, found := seenMay[atn]; found && ErrorOnDuplicateClauseMember {
				err = errOCDuplicateType(def.noid, atoid, atdescr, `MAY`)
				return
			}
			seenMay[atn] = struct{}{}
			def.may[i] = atoid
		}
	}

	return
}

func (r *ObjectClassProperties) checkSup(def *objectClassDefinition) (err error) {
	seen := make(map[uint]struct{})
	for i := 0; i < len(def.supers); i++ {
		// ensure each SUP exists
		res, resd, _, resn := r.Resolve(def.supers[i])
		if res == nil {
			err = errOCUnregSup(def.noid, def.supers[i])
			break
		}
		if _, found := seen[resn]; found && ErrorOnDuplicateClauseMember {
			err = errOCDuplicateType(def.noid, res, resd, `SUP`)
			break
		}
		seen[resn] = struct{}{}

		supk := r.KIND[resn]
		// if new OC is 'STRUCTURAL' kind, ensure none of
		// its SUP clause members are of kind 'AUXILIARY'.
		if supk == 0x1 && def.kind == 0x0 {
			err = errOCStructAux(def.noid, res)
			break
		}
		def.supers[i] = res
	}

	return
}

func (r *ObjectClassProperties) unregister(id []byte) (err error) {
	noid, _, descrs, n := r.Resolve(id)
	if noid == nil {
		return // no error needed
	}

	if dc, found := r.DC[n]; found {
		_, dep, _, _ := r.schema.dc.Resolve(r.PRINC[dc])
		err = errOCDepDCR(noid, dep)
		return

	}

	if nf, found := r.NF[n]; found {
		if L := len(nf); L > 0 {
			err = errOCDepNF(noid, L)
			return
		}
	}

	if subc, found := r.SUBC[n]; found {
		if L := len(subc); L > 0 {
			err = errOCDepOC(noid, L)
			return
		}
	}

	for i := 0; i < len(descrs); i++ {
		delete(r.MAP, string(lc(descrs[i])))
	}
	delete(r.MAP, string(noid))

	delete(r.ALLMUST, n)
	delete(r.MUSTORD, n)
	delete(r.SUPCORD, n)
	delete(r.STRING, n)
	delete(r.MAYORD, n)
	delete(r.ALLMAY, n)
	delete(r.CHAINS, n)
	delete(r.PRINC, n)
	delete(r.DESCR, n)
	delete(r.KIND, n)
	delete(r.MUST, n)
	delete(r.TEXT, n)
	delete(r.SUPC, n)
	delete(r.SUBC, n)
	delete(r.MAY, n)
	delete(r.EXT, n)
	delete(r.OBS, n)
	delete(r.DC, n)
	delete(r.NF, n)
	delete(r.ID, n)

	idx := r.IDX[n]
	r.ORD = append(r.ORD[:idx], r.ORD[idx+1:]...)

	r.IDX = make(map[uint]int)
	for i := 0; i < len(r.ORD); i++ {
		r.IDX[r.ORD[i]] = i
	}

	return
}

func (r *ObjectClassProperties) commit(def objectClassDefinition, checkOnly bool) (err error) {
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
	r.commitSupers(def.n, def.supers)
	r.commitMust(def.n, def.must)
	r.commitMay(def.n, def.may)

	r.KIND[def.n] = def.kind

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

	r.ALLMUST[def.n] = r.buildAllMust(def.n)
	r.ALLMAY[def.n] = r.buildAllMay(def.n)

	return
}

func (r *ObjectClassProperties) commitDescr(n uint, noid []byte, descrs [][]byte) error {
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

func (r *ObjectClassProperties) commitSupers(n uint, supers [][]byte) {
	if L := len(supers); L > 0 {
		if _, found := r.SUPC[n]; !found {
			r.SUPC[n] = make(map[uint]struct{})
		}
		if _, found := r.SUPCORD[n]; !found {
			r.SUPCORD[n] = make([]uint, 0)
		}

		for i := 0; i < L; i++ {
			supn := r.MAP[string(lc(supers[i]))]

			// forward mapping
			r.SUPCORD[n] = append(r.SUPCORD[n], supn) // record order of sup oid(s)
			r.SUPC[n][supn] = struct{}{}              // oid -> sup oid

			// reverse mapping
			if _, found := r.SUBC[supn]; !found { // check if sup oid has sub class map
				r.SUBC[supn] = make(map[uint]struct{}) // init sup oid's sub class map
			}
			r.SUBC[supn][n] = struct{}{} // sup oid -> sub class noid
		}

		r.CHAINS[n] = r.buildChains(n)
	}
}

func (r *ObjectClassProperties) commitMust(n uint, must [][]byte) {
	if L := len(must); L > 0 {
		if _, found := r.MUST[n]; !found {
			r.MUST[n] = make(map[uint]struct{})
		}
		if _, found := r.MUSTORD[n]; !found {
			r.MUSTORD[n] = make([]uint, 0)
		}

		for i := 0; i < L; i++ {
			mustn := r.schema.at.MAP[string(lc(must[i]))]

			if _, found := r.schema.at.OC[mustn]; !found {
				r.schema.at.OC[mustn] = make(map[uint]struct{})
			}
			r.MUSTORD[n] = append(r.MUSTORD[n], mustn)
			r.schema.at.OC[mustn][n] = struct{}{}
			r.MUST[n][mustn] = struct{}{}
		}
	}
}

func (r *ObjectClassProperties) commitMay(n uint, may [][]byte) {
	if L := len(may); L > 0 {
		if _, found := r.MAY[n]; !found {
			r.MAY[n] = make(map[uint]struct{})
		}
		if _, found := r.MAYORD[n]; !found {
			r.MAYORD[n] = make([]uint, 0)
		}

		for i := 0; i < L; i++ {
			mayn := r.schema.at.MAP[string(lc(may[i]))]

			if _, found := r.schema.at.OC[mayn]; !found {
				r.schema.at.OC[mayn] = make(map[uint]struct{})
			}
			r.MAYORD[n] = append(r.MAYORD[n], mayn)
			r.schema.at.OC[mayn][n] = struct{}{}
			r.MAY[n][mayn] = struct{}{}
		}
	}
}

/*
Resolve returns a numeric OID, principal descriptor (name) and slices of
all descriptors associated with the input search term.

Valid input values are the numeric OID of the definition or one of its
official descriptors.

Case is not significant in the name matching process.
*/
func (r ObjectClassProperties) Resolve(term []byte) (noid, princ []byte, descrs [][]byte, n uint) {
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
SetPrivate classifies the specified object class as private. This is useful
if the schema administrator wishes to prevent disclosure of the specified
class in a given subschemaSubentry.

The class can still be interrogated directly like any other when this
package is being interacted with directly. However, when tied into a DSA,
end users ostensibly shall not be able to interrogate or interact with
a private class through conventional means.

Valid input values are the numeric OID or descriptor (name) associated
with the desired definition.

Case is not significant in the name matching process.

See also [AttributeTypeProperties.SetPrivate].
*/
func (r *ObjectClassProperties) SetPrivate(id []byte) {
	res, _, _, n := r.Resolve(id)
	if res != nil {
		r.schema.lock.Lock()
		r.PRIV[n] = struct{}{}
		r.schema.lock.Unlock()
	}
}

func (r *ObjectClassProperties) makeString(n uint) (err error) {
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

	_, obs := r.OBS[n]
	bld.Write(stringBooleanClause(`OBSOLETE`, obs))

	if super, found := r.SUPCORD[n]; found {
		var sups [][]byte
		for i := 0; i < len(super); i++ {
			sups = append(sups, r.PRINC[super[i]])
		}
		bld.Write(definitionMVDescriptors(`SUP`, sups))
	}

	bld.WriteString(classKind(r.KIND[n]))

	if must, found := r.MUSTORD[n]; found {
		var musts [][]byte
		for i := 0; i < len(must); i++ {
			musts = append(musts, r.schema.at.PRINC[must[i]])
		}
		bld.Write(definitionMVDescriptors(`MUST`, musts))
	}

	if may, found := r.MAYORD[n]; found {
		var mays [][]byte
		for i := 0; i < len(may); i++ {
			mays = append(mays, r.schema.at.PRINC[may[i]])
		}
		bld.Write(definitionMVDescriptors(`MAY`, mays))
	}

	if ext, found := r.EXT[n]; found {
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

The id argument may be the numeric OID or the descriptor (name) of the
desired definition.

Case is not significant in the name matching process.

Note that class definitions classified private WILL be returned, if
found.
*/
func (r ObjectClassProperties) Get(id []byte) string {
	var s string
	if noid, _, _, n := r.Resolve(id); noid != nil {
		s = r.STRING[n]
	}

	return s
}

/*
RegisterObjectClass returns an error following an attempt to parse the input
def bytes. If successful, the components of def are written to the underlying
matrices.
*/
func (r *SubschemaSubentry) RegisterObjectClass(def []byte) (err error) {
	if !bytes.HasPrefix(lc(def), lc([]byte(headerTokens[9]))) {
		def = append([]byte(headerTokens[9]+": "), def...)
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	err = r.oc.marshal(def, false)

	return
}

/*
UnregisterObjectClass returns an error following an attempt to remove the
specified definition from the receiver instance. Valid input values are
the numeric OID or descriptor (name) of the desired definition.

Case is not significant in the name matching process.
*/
func (r *SubschemaSubentry) UnregisterObjectClass(id []byte) (err error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	err = r.oc.unregister(id)
	return
}

func errOCDuplicateType(noid, atoid, atdescr []byte, clause string) error {
	return errors.New("ObjectClass '" + string(noid) +
		"' bears a duplicate " + clause + " clause member '" +
		string(atdescr) + "' (" + string(atoid) + ")")
}

func errOCStructAux(noid, resd []byte) error {
	return errors.New("STRUCTURAL ObjectClass '" + string(noid) +
		"' bears a SUP clause member of 'AUXILIARY' kind '" +
		string(resd) + "'")
}

func errOCUnregSup(noid, sup []byte) error {
	return errors.New("ObjectClass '" + string(noid) +
		"' bears an unregistered SUP clause member '" +
		string(sup) + "'")
}

func errOCUnregType(noid, atoid []byte, clause string) error {
	return errors.New("ObjectClass '" + string(noid) +
		"' bears an unregistered " + clause + " clause member '" +
		string(atoid) + "'")
}

func errOCNotFound(noid []byte) error {
	return errors.New("ObjectClass id '" + string(noid) + "' not found")
}

func errOCDepDCR(noid, dep []byte) error {
	return errors.New("ObjectClass '" + string(noid) +
		" is depended upon by content rule '" +
		string(dep) + "'")
}

func errOCDepNF(noid []byte, count int) error {
	return errors.New("ObjectClass '" + string(noid) +
		" is depended upon by " + strconv.Itoa(count) +
		" name forms")
}

func errOCDepOC(noid []byte, count int) error {
	return errors.New("ObjectClass '" + string(noid) +
		" is depended upon by " + strconv.Itoa(count) +
		" sub classes")
}

func errOCNameReg(name, other []byte) error {
	return errors.New("ObjectClass NAME '" + string(name) +
		"' already registered to '" + string(other) + "'")
}

func errOCMalformedType(noid, descr []byte, clause string) error {
	return errors.New("ObjectClass '" + string(noid) +
		"' bears malformed " + clause + " clause member '" +
		string(descr) + "'")
}

func errOCNotUnique(noid []byte) error {
	return errors.New("ObjectClass OID '" + string(noid) +
		"' already registered")
}

func errOCIllegalKind(token string) error {
	return errors.New("ObjectClass bears illegal KIND '" +
		token + "'; want 'STRUCTURAL', 'AUXILIARY' or 'ABSTRACT'")
}
