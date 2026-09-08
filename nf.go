package schema

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
)

/*
NameFormProperties implements a matrix of fast lookup tables
pertaining to registered Name Form definitions.
*/
type NameFormProperties struct {
	MAP     map[string]uint             // nrml. form descriptor (v) to numeric OID (v)
	ID      map[uint][]byte             // form (k) to numeric OID (v)
	PRINC   map[uint][]byte             // form (k) to principal descriptor (v)
	TEXT    map[uint][]byte             // form (k) bears text description (v)
	STRING  map[uint]string             // form (k) in string representation (v)
	OC      map[uint]uint               // form (k) to object class (v)
	MUSTORD map[uint][]uint             // ordered collection of mandatory attribute type OIDs
	MAYORD  map[uint][]uint             // ordered collection of optional attribute type OIDs
	DESCR   map[uint][][]byte           // ordered collection of optional attribute type OIDs
	MUST    map[uint]map[uint]struct{}  // form (k) to mandatory type OIDs (k2)
	MAY     map[uint]map[uint]struct{}  // form (k) to optional type OIDs (k2)
	DS      map[uint]map[uint]struct{}  // form (k1) to dependent structure rule (k2)
	EXT     map[uint]map[uint]Extension // form (k) bears extension(s) (k2, v)
	OBS     map[uint]struct{}           // form (k) is obsolete
	IDX     map[uint]int                // form (k) resides at ORD slice N
	ORD     []uint                      // ordered collection of form numeric OIDs

	schema *SubschemaSubentry
}

func initNameFormProperties(sch *SubschemaSubentry) (nfp *NameFormProperties) {
	nfp = &NameFormProperties{
		MAP:     make(map[string]uint),
		ID:      make(map[uint][]byte),
		DESCR:   make(map[uint][][]byte),
		MUSTORD: make(map[uint][]uint),
		MAYORD:  make(map[uint][]uint),
		PRINC:   make(map[uint][]byte),
		TEXT:    make(map[uint][]byte),
		STRING:  make(map[uint]string),
		OC:      make(map[uint]uint),
		DS:      make(map[uint]map[uint]struct{}),
		MUST:    make(map[uint]map[uint]struct{}),
		MAY:     make(map[uint]map[uint]struct{}),
		EXT:     make(map[uint]map[uint]Extension),
		OBS:     make(map[uint]struct{}),
		IDX:     make(map[uint]int),
		ORD:     make([]uint, 0),
		schema:  sch,
	}

	return
}

func (r NameFormProperties) nameFormDescription(x any) (result bool, err error) {
	var input []byte
	switch tv := x.(type) {
	case []byte:
		input = tv
	case string:
		input = []byte(tv)
	default:
		err = errorBadType("nameForm")
		return
	}

	err = r.marshal(input, true)
	result = err == nil
	return
}

func (r *NameFormProperties) makeString(id uint) (err error) {
	noid := r.ID[id]
	descrs := r.DESCR[id]

	bld := &strings.Builder{}
	bld.WriteRune('(')
	bld.WriteRune(' ')
	bld.Write(noid)
	bld.Write(definitionName(descrs))

	if text, found := r.TEXT[id]; found {
		bld.Write(definitionDescription(text))
	}

	_, obs := r.OBS[id]
	bld.Write(stringBooleanClause(`OBSOLETE`, obs))

	oc, _ := r.OC[id]
	oprinc := r.schema.oc.PRINC[oc]
	bld.Write(definitionMVDescriptors(`OC`, oprinc))

	if must, found := r.MUSTORD[id]; found {
		var musts [][]byte
		for i := 0; i < len(must); i++ {
			princ := r.schema.at.PRINC[must[i]]
			musts = append(musts, princ)
		}
		bld.Write(definitionMVDescriptors(`MUST`, musts))
	}

	if may, found := r.MAYORD[id]; found {
		var mays [][]byte
		for i := 0; i < len(may); i++ {
			princ := r.schema.at.PRINC[may[i]]
			mays = append(mays, princ)
		}
		bld.Write(definitionMVDescriptors(`MAY`, mays))
	}

	if ext, found := r.EXT[id]; found {
		bld.Write(stringExtensions(ext))
	}

	bld.WriteRune(' ')
	bld.WriteRune(')')
	r.STRING[id] = bld.String()

	return
}

func (r *NameFormProperties) marshal(in []byte, checkOnly bool) (err error) {
	input := bytes.TrimSpace(trimDefinitionLabelToken(in))
	tkz := newSchemaTokenizer(input)
	tkz.startTokenParen()

	noid := tkz.this()

	if !isObjectIdentifier(noid) {
		err = errMalformedOID(`NameForm`, string(noid))
		return
	}

	if nn, _, _, _ := r.Resolve(noid); nn != nil && !checkOnly {
		err = errNFNotUnique(noid)
		return
	}

	def := nameFormDefinition{
		n:    uint(len(r.ORD)),
		noid: noid,
		ext:  make(map[uint]Extension),
	}

	seen := make(map[string]struct{})

	for tkz.next() && err == nil {
		token := string(tkz.this())
		if _, found := seen[token]; found {
			err = errors.New("NameForm clause '" + token +
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
		case "OC":
			def.oc = parseSingleVal(tkz)
			err = r.checkOCClause(noid, def.oc)
			seen[token] = struct{}{}
		case "MUST":
			def.must = parseMultiVal(tkz)
			err = r.checkMVClause(noid, def.must, `MUST`)
			seen[token] = struct{}{}
		case "MAY":
			def.may = parseMultiVal(tkz)
			err = r.checkMVClause(noid, def.may, `MAY`)
			seen[token] = struct{}{}
		default:
			var ext Extension
			ext, err = marshalExtension(token, "nameForm", tkz)
			def.ext[uint(len(def.ext))] = ext
			seen[token] = struct{}{}
		}
	}

	return
}

func (r NameFormProperties) checkOCClause(
	noid, clause []byte,
) (err error) {
	if !isAttribute(clause) {
		err = errNFBadOC(noid, clause)
	}

	return
}

type nameFormDefinition struct {
	n      uint
	noid   []byte
	text   []byte
	oc     []byte
	descrs [][]byte
	must   [][]byte
	may    [][]byte
	obs    bool
	ext    map[uint]Extension
}

func (r *NameFormProperties) check(def *nameFormDefinition) (err error) {
	for i := 0; i < len(def.descrs); i++ {
		item := lc(def.descrs[i])
		if reg, found := r.MAP[string(item)]; found {
			err = errNFNameReg(def.descrs[i], r.PRINC[reg])
			return
		}
	}

	class, _, _, classN := r.schema.oc.Resolve(def.oc)
	if class == nil {
		err = errNFUnregOC(def.noid, def.oc)
		return
	}
	def.oc = class

	var availableTypes map[uint]struct{}
	if availableTypes, err = r.gatherAvailableTypes(def, classN); err == nil {
		err = r.resolveMM(def, availableTypes)
	}

	return
}

func (r *NameFormProperties) gatherAvailableTypes(
	def *nameFormDefinition,
	classN uint,
) (
	avail map[uint]struct{},
	err error,
) {
	avail = make(map[uint]struct{})

	// add structurally-provided types first
	for at := range r.schema.oc.MUST[classN] {
		avail[at] = struct{}{}
	}

	for at := range r.schema.oc.MAY[classN] {
		avail[at] = struct{}{}
	}

	return
}

func (r *NameFormProperties) resolveMM(
	def *nameFormDefinition,
	avail map[uint]struct{},
) (err error) {

	seen := make(map[string]struct{})
	for k, v := range map[string][][]byte{
		`MUST`: def.must,
		`MAY`:  def.may,
	} {
		for i := 0; i < len(v); i++ {
			atoid, atdescr, _, atn := r.schema.at.Resolve(v[i])
			if atoid == nil {
				err = errNFUnregType(def.noid, v[i], k)
				return
			}
			if _, found := seen[string(atoid)]; found && ErrorOnDuplicateClauseMember {
				err = errNFDuplicateType(def.noid, atoid, atdescr, k)
				return
			}

			seen[string(atoid)] = struct{}{}
			if _, found := avail[atn]; !found {
				err = errNFIneligibleType(def.noid, atdescr)
				return
			}

			switch k {
			case `MUST`:
				def.must[i] = atoid
			case `MAY`:
				def.may[i] = atoid
			}
		}
	}

	return
}

func (r *NameFormProperties) commit(def nameFormDefinition, checkOnly bool) (err error) {
	if checkOnly {
		return
	}

	if err = r.check(&def); err != nil {
		return
	}

	r.IDX[def.n] = len(r.ORD)
	r.ORD = append(r.ORD, def.n)
	r.MAP[string(def.noid)] = def.n
	r.ID[def.n] = def.noid

	if def.text != nil {
		r.TEXT[def.n] = def.text
	}

	r.commitDescr(def.n, def.noid, def.descrs)

	// make a note of this name form being associated with
	// the specified STRUCTURAL class.
	oc := r.schema.oc.MAP[string(lc(def.oc))]
	if _, found := r.schema.oc.NF[oc]; !found {
		r.schema.oc.NF[oc] = make(map[uint]struct{})
	}

	r.OC[def.n] = oc
	r.schema.oc.NF[oc][def.n] = struct{}{}

	r.commitMust(def)
	r.commitMay(def)

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

func (r *NameFormProperties) commitDescr(n uint, noid []byte, descrs [][]byte) error {
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

// record MANDATORY types
func (r *NameFormProperties) commitMust(def nameFormDefinition) {
	if lmusts := len(def.must); lmusts > 0 {
		if _, found := r.MUST[def.n]; !found {
			r.MUST[def.n] = make(map[uint]struct{})
		}
		if _, found := r.MUSTORD[def.n]; !found {
			r.MUSTORD[def.n] = make([]uint, 0)
		}

		for i := 0; i < lmusts; i++ {
			id := r.schema.at.MAP[string(lc(def.must[i]))]
			if _, found := r.schema.at.NF[id]; !found {
				r.schema.at.NF[id] = make(map[uint]struct{})
			}
			r.MUSTORD[def.n] = append(r.MUSTORD[def.n], id)
			r.schema.at.NF[id][def.n] = struct{}{}
			r.MUST[def.n][id] = struct{}{}
		}
	}
}

// record OPTIONAL types
func (r *NameFormProperties) commitMay(def nameFormDefinition) {
	if lmays := len(def.may); lmays > 0 {
		if _, found := r.MAY[def.n]; !found {
			r.MAY[def.n] = make(map[uint]struct{})
		}
		if _, found := r.MAYORD[def.n]; !found {
			r.MAYORD[def.n] = make([]uint, 0)
		}

		for i := 0; i < lmays; i++ {
			id := r.schema.at.MAP[string(lc(def.may[i]))]
			if _, found := r.schema.at.NF[id]; !found {
				r.schema.at.NF[id] = make(map[uint]struct{})
			}
			r.MAYORD[def.n] = append(r.MAYORD[def.n], id)
			r.schema.at.NF[id][def.n] = struct{}{}
			r.MAY[def.n][id] = struct{}{}
		}
	}
}

func (r *NameFormProperties) unregister(id []byte) (err error) {
	noid, _, descrs, n := r.Resolve(id)
	if noid == nil {
		return // no error needed
	}

	if ds, found := r.DS[n]; found {
		if L := len(ds); L > 0 {
			err = errNFHasDependents(noid, L)
			return
		}
	}

	for i := 0; i < len(descrs); i++ {
		delete(r.MAP, string(lc(descrs[i])))
	}
	delete(r.MAP, string(noid))

	for k, v := range r.schema.at.NF {
		for k2 := range v {
			if k2 == n {
				delete(r.schema.at.NF[k], n)
				break
			}
		}
	}

	delete(r.schema.oc.NF, r.OC[n])
	delete(r.DS, n)
	//delete(r.schema.ds.FORM, n) // TODO?!??!
	delete(r.OC, n)
	delete(r.ID, n)
	delete(r.OBS, n)
	delete(r.EXT, n)
	delete(r.MAY, n)
	delete(r.MUST, n)
	delete(r.TEXT, n)
	delete(r.DESCR, n)
	delete(r.PRINC, n)
	delete(r.STRING, n)
	delete(r.MAYORD, n)
	delete(r.MUSTORD, n)

	idx := r.IDX[n]
	r.ORD = append(r.ORD[:idx], r.ORD[idx+1:]...)
	r.IDX = make(map[uint]int)
	for i := 0; i < len(r.ORD); i++ {
		r.IDX[r.ORD[i]] = i
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
func (r NameFormProperties) Resolve(term []byte) (noid, princ []byte, descrs [][]byte, n uint) {
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

func (r NameFormProperties) checkMVClause(
	noid []byte,
	clause [][]byte,
	name string,
) (err error) {
	for i := 0; i < len(clause); i++ {
		if !isAttribute(clause[i]) {
			err = errNFMalformedType(noid, clause[i], name)
			break
		}
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
func (r NameFormProperties) Get(id []byte) string {
	var s string
	if noid, _, _, n := r.Resolve(id); noid != nil {
		s = r.STRING[n]
	}

	return s
}

/*
RegisterNameForm returns an error following an attempt to parse the input
def bytes. If successful, the components of def are written to the underlying
matrices.
*/
func (r *SubschemaSubentry) RegisterNameForm(def []byte) (err error) {
	if !bytes.HasPrefix(lc(def), lc([]byte(headerTokens[13]))) {
		def = append([]byte(headerTokens[13]+": "), def...)
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	err = r.nf.marshal(def, false)

	return
}

/*
UnregisterNameForm returns an error following an attempt to remove the
specified definition from the receiver instance. Valid input values are
the numeric OID or descriptor (name) of the desired definition.

Case is not significant in the name matching process.
*/
func (r *SubschemaSubentry) UnregisterNameForm(id []byte) (err error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	err = r.nf.unregister(id)
	return
}

func errNFMalformedType(noid, descr []byte, clause string) error {
	return errors.New("NameForm '" + string(noid) +
		"' bears malformed " + clause + " clause member '" +
		string(descr) + "'")
}

func errNFIneligibleType(noid, descr []byte) error {
	return errors.New("NameForm '" + string(noid) +
		"' bears a MUST or MAY clause member '" + string(descr) +
		"' neither required nor allowed by OC")
}

func errNFDuplicateType(noid, atoid, atdescr []byte, clause string) error {
	return errors.New("NameForm '" + string(noid) +
		"' bears a duplicate " + clause + " clause member '" +
		string(atdescr) + "' (" + string(atoid) + ")")
}

func errNFUnregType(noid, atoid []byte, clause string) error {
	return errors.New("NameForm '" + string(noid) +
		"' bears an unregistered " + clause + " clause member '" +
		string(atoid) + "'")
}

func errNFBadOC(noid, oc []byte) error {
	return errors.New("NameForm '" + string(noid) +
		"' bears an invalid OC clause member '" +
		string(oc) + "'")
}

func errNFUnregOC(noid, oc []byte) error {
	return errors.New("NameForm '" + string(noid) +
		"' bears an unregistered OC clause member '" +
		string(oc) + "'")
}

func errNFNameReg(name, other []byte) error {
	return errors.New("NameForm NAME '" + string(name) +
		"' already registered to '" + string(other) + "'")
}

func errNFHasDependents(noid []byte, count int) error {
	return errors.New("NameForm '" + string(noid) + "' has " +
		strconv.Itoa(count) + " structure rule dependents")
}

func errNFNotUnique(noid []byte) error {
	return errors.New("NameForm OID '" + string(noid) + "' already registered")
}

func errNFNotFound(noid []byte) error {
	return errors.New("NameForm id '" + string(noid) + "' not found")
}
