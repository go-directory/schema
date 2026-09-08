package schema

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
)

/*
DITStructureRuleProperties implements a container type for
DIT Structure Rule properties.
*/
type DITStructureRuleProperties struct {
	MAP     map[string]uint             // rule string numeric OID or nrml. descriptor to #
	ID      map[uint][]byte             // rule id (k) to numeric OID (v)
	SUPRORD map[uint][]uint             // ordered collection of superior rule refs
	DESCR   map[uint][][]byte           // nrml. descriptor (k) to rule integer identifier (v)
	PRINC   map[uint][]byte             // rule integer identifier (k) to principal descriptor (v)
	TEXT    map[uint][]byte             // rule integer identifier (k) to textual description (v)
	STRING  map[uint]string             // rule integer identifier (k) in string representation (v)
	FORM    map[uint]uint               // rule integer identifier (k) to nameForm (v)
	SUBR    map[uint]map[uint]struct{}  // rule integer identifier (k1) to subordinate rules (k2)
	SUPR    map[uint]map[uint]struct{}  // rule integer identifier (k1) to superior rules (k2)
	EXT     map[uint]map[uint]Extension // rule integer identifier (k1) bears extensions (k2)
	OBS     map[uint]struct{}           // rule integer identifier (k) is obsolete
	IDX     map[uint]int                // rule integer identifier (k) is Nth in ORD collection (v)
	ORD     []uint                      // ordered collection of structure rules
	schema  *SubschemaSubentry          // internal reference
}

func initDITStructureRuleProperties(sch *SubschemaSubentry) (dsp *DITStructureRuleProperties) {
	dsp = &DITStructureRuleProperties{
		MAP:     make(map[string]uint),
		DESCR:   make(map[uint][][]byte),
		ID:      make(map[uint][]byte),
		PRINC:   make(map[uint][]byte),
		TEXT:    make(map[uint][]byte),
		FORM:    make(map[uint]uint),
		STRING:  make(map[uint]string),
		SUPRORD: make(map[uint][]uint),
		OBS:     make(map[uint]struct{}),
		SUBR:    make(map[uint]map[uint]struct{}),
		SUPR:    make(map[uint]map[uint]struct{}),
		EXT:     make(map[uint]map[uint]Extension),
		IDX:     make(map[uint]int),
		ORD:     make([]uint, 0),
		schema:  sch,
	}

	return
}

func (r DITStructureRuleProperties) dITStructureRuleDescription(x any) (result bool, err error) {
	var input []byte

	switch tv := x.(type) {
	case []byte:
		input = tv
	case string:
		input = []byte(tv)
	default:
		err = errorBadType("dITStructureRule")
		return
	}

	err = r.marshal(input, true)
	result = err == nil
	return
}

func (r *DITStructureRuleProperties) makeString(id uint) (err error) {
	rid := r.ID[id]
	descrs := r.DESCR[id]

	bld := &strings.Builder{}
	bld.WriteRune('(')
	bld.WriteRune(' ')
	bld.Write(rid)
	bld.Write(definitionName(descrs))

	if text, found := r.TEXT[id]; found {
		bld.Write(definitionDescription(text))
	}

	_, obs := r.OBS[id]
	bld.Write(stringBooleanClause(`OBSOLETE`, obs))

	fm, _ := r.FORM[id]
	fdescr := r.schema.nf.DESCR[fm]
	bld.Write(definitionMVDescriptors(`FORM`, fdescr))

	if supr, found := r.SUPRORD[id]; found {
		var sups [][]byte
		for i := 0; i < len(supr); i++ {
			mm := r.ID[supr[i]]
			sups = append(sups, mm)
		}
		bld.Write(definitionMVDescriptors(`SUP`, sups, true))
	}

	if ext, found := r.EXT[id]; found {
		bld.Write(stringExtensions(ext))
	}

	bld.WriteRune(' ')
	bld.WriteRune(')')
	r.STRING[id] = bld.String()

	return
}

func (r *DITStructureRuleProperties) marshal(in []byte, checkOnly bool) (err error) {
	input := bytes.TrimSpace(trimDefinitionLabelToken(in))
	tkz := newSchemaTokenizer(input)
	tkz.startTokenParen()

	id := tkz.this()

	if !isUnsignedNumber(id) {
		err = errDSRMalformedID(id, nil)
		return
	}

	if _, found := r.MAP[string(lc(id))]; found && !checkOnly {
		err = errDSRNotUnique(id)
		return
	}

	def := dITStructureRuleDefinition{
		n:   uint(len(r.ORD)),
		id:  id,
		ext: make(map[uint]Extension),
	}

	seen := make(map[string]struct{})

	for tkz.next() && err == nil {
		token := string(tkz.this())
		if _, found := seen[token]; found {
			err = errors.New("DITStructureRule clause '" + token +
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
			err = checkDescriptors(def.descrs, def.id)
			seen[token] = struct{}{}
		case "DESC":
			def.text = parseSingleVal(tkz)
			seen[token] = struct{}{}
		case "OBSOLETE":
			def.obs = true
			seen[token] = struct{}{}
		case "SUP":
			def.sup = parseMultiVal(tkz)
			err = r.checkSupClause(def.id, def.sup)
			seen[token] = struct{}{}
		case "FORM":
			def.form = parseSingleVal(tkz)
			err = r.checkFormClause(def.id, def.form)
			seen[token] = struct{}{}
		default:
			var ext Extension
			ext, err = marshalExtension(token, "dITStructureRule", tkz)
			def.ext[uint(len(def.ext))] = ext
			seen[token] = struct{}{}
		}
	}

	return
}

func (r *DITStructureRuleProperties) check(def *dITStructureRuleDefinition) (err error) {
	for i := 0; i < len(def.descrs); i++ {
		item := lc(def.descrs[i])
		if reg, found := r.MAP[string(item)]; found {
			err = errDSRNameNotUnique(def.descrs[i], r.PRINC[reg])
			return
		}
	}

	form, _, _, _ := r.schema.nf.Resolve(def.form)
	if form == nil {
		err = errDSRUnregForm(def.id, def.form)
		return
	}
	def.form = form

	seen := make(map[string]struct{})
	for i := 0; i < len(def.sup); i++ {
		if _, found := seen[string(def.sup[i])]; found && ErrorOnDuplicateClauseMember {
			err = errDSRDuplicate(def.id, def.sup[i])
			break
		}
		seen[string(def.sup[i])] = struct{}{}

		if bytes.Equal(def.sup[i], def.id) {
			// rule is recursive (it is a member of its own SUP
			// clause), so don't bother trying to resolve it.
			// We haven't finished adding it in the first place.
			continue
		}

		if rid, _, _, _ := r.Resolve(def.sup[i]); rid == nil {
			err = errDSRUnregSup(def.id, def.sup[i])
			break
		}
	}

	return
}

func (r *DITStructureRuleProperties) checkFormClause(
	id []byte,
	clause []byte,
) (err error) {
	if !isAttribute(clause) {
		err = errDSRMalformedForm(id, clause)
	}

	return
}

func (r DITStructureRuleProperties) checkSupClause(
	id []byte,
	clause [][]byte,
) (err error) {
	for i := 0; i < len(clause); i++ {
		if !isUnsignedNumber(clause[i]) {
			err = errDSRMalformedID(id, clause[i])
			break
		}
	}

	return
}

type dITStructureRuleDefinition struct {
	n      uint
	id     []byte
	text   []byte
	form   []byte
	descrs [][]byte
	sup    [][]byte
	sub    [][]byte
	obs    bool
	ext    map[uint]Extension
}

func (r *DITStructureRuleProperties) unregister(id []byte) (err error) {
	ii, _, descrs, n := r.Resolve(id)
	if ii == nil {
		return // error not needed
	}

	if subs, ok := r.SUBR[n]; ok {
		if L := len(subs); L > 0 {
			err = errDSRHasDependents(ii, L)
			return
		}
	}

	for i := 0; i < len(descrs); i++ {
		delete(r.MAP, string(lc(descrs[i])))
	}
	delete(r.MAP, string(ii))

	delete(r.SUPRORD, n)
	delete(r.STRING, n)
	delete(r.DESCR, n)
	delete(r.PRINC, n)
	delete(r.TEXT, n)
	delete(r.FORM, n)
	delete(r.SUBR, n)
	delete(r.SUPR, n)
	delete(r.OBS, n)
	delete(r.EXT, n)

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

func (r *DITStructureRuleProperties) commit(def dITStructureRuleDefinition) (err error) {
	if err = r.check(&def); err != nil {
		return
	}

	r.IDX[def.n] = len(r.ORD)
	r.ORD = append(r.ORD, def.n)
	r.MAP[string(def.id)] = def.n
	r.ID[def.n] = def.id

	r.commitDescr(def.n, def.id, def.descrs)

	// make a note of this dependency in the DS lookup
	// table within NameFormProperties. Ordering is not
	// necessary.
	_, _, _, formn := r.schema.nf.Resolve(def.form)
	r.FORM[def.n] = formn
	if _, found := r.schema.nf.DS[formn]; !found {
		r.schema.nf.DS[formn] = make(map[uint]struct{})
	}
	r.schema.nf.DS[formn][def.n] = struct{}{}

	// record SUP/SUB relationships
	if lsup := len(def.sup); lsup > 0 {
		if _, found := r.SUBR[def.n]; !found {
			r.SUBR[def.n] = make(map[uint]struct{})
		}
		if _, found := r.SUPR[def.n]; !found {
			r.SUPR[def.n] = make(map[uint]struct{})
		}
		if _, found := r.SUPRORD[def.n]; !found {
			r.SUPRORD[def.n] = make([]uint, 0)
		}

		for i := 0; i < lsup; i++ {
			_, _, _, lln := r.schema.ds.Resolve(def.sup[i])
			r.SUPRORD[def.n] = append(r.SUPRORD[def.n], lln)
			if _, found := r.SUBR[lln][def.n]; !found {
				r.SUBR[lln] = make(map[uint]struct{})
			}
			r.SUBR[lln][def.n] = struct{}{}
			r.SUPR[def.n][lln] = struct{}{}
		}
	}

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

func (r *DITStructureRuleProperties) commitDescr(n uint, id []byte, descrs [][]byte) error {
	if len(descrs) > 0 {
		for i := 0; i < len(descrs); i++ {
			r.MAP[string(lc(descrs[i]))] = n
		}
		r.PRINC[n] = descrs[0]
		r.DESCR[n] = descrs
	} else {
		r.PRINC[n] = id
	}

	return nil
}

/*
Resolve returns a integer identifier, a principal descriptor and slices of all
descriptors associated with the input search term.

Valid input values are the integer identifier of the definition or any of its
official descriptors (names).

Case is not significant in the name matching process.
*/
func (r DITStructureRuleProperties) Resolve(term []byte) (id, princ []byte, descrs [][]byte, n uint) {
	if len(term) > 0 {
		var found bool
		if n, found = r.MAP[string(lc(term))]; found {
			id, _ = r.ID[n]
			princ, _ = r.PRINC[n]
			descrs, _ = r.DESCR[n]
		}
	}

	return
}

func (r DITStructureRuleProperties) NamedObjectClass(term []byte) (
	id, princ []byte,
	descrs [][]byte,
	n uint,
) {

	var t []byte
	if t, _, _, n = r.Resolve(term); t != nil {
		form := r.FORM[n]
		oc := r.schema.nf.OC[form]

		id = r.schema.oc.ID[oc]
		princ = r.schema.oc.PRINC[oc]
		descrs = r.schema.oc.DESCR[oc]
	}

	return
}

/*
Get returns the pre-generated string representation of the definition
bearing the input id value. A zero string is returned if not found.

The id argument may be the integer identifier or the descriptor (name)
of the desired definition.

Case is not significant in the name matching process.
*/
func (r DITStructureRuleProperties) Get(id []byte) string {
	var s string
	if noid, _, _, n := r.Resolve(id); noid != nil {
		s = r.STRING[n]
	}

	return s
}

/*
RegisterDITStructureRule returns an error following an attempt to parse the input
def bytes. If successful, the components of def are written to the underlying
matrices.
*/
func (r *SubschemaSubentry) RegisterDITStructureRule(def []byte) (err error) {
	if !bytes.HasPrefix(lc(def), lc([]byte(headerTokens[15]))) {
		def = append([]byte(headerTokens[15]+": "), def...)
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	err = r.ds.marshal(def, false)

	return
}

/*
UnregisterDITStructureRule returns an error following an attempt to remove the
specified definition from the receiver instance. Valid input values are
the integer identifier or descriptor (name) of the desired definition.

Case is not significant in the name matching process.
*/
func (r *SubschemaSubentry) UnregisterDITStructureRule(id []byte) (err error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	err = r.ds.unregister(id)
	return
}

func errDSRMalformedID(id, sup []byte) error {
	if sup != nil {
		return errors.New("DITStructureRule '" + string(id) +
			"' bears malformed superior integer identifier '" +
			string(sup) + "'")
	}
	return errors.New("DITStructureRule bears a malformed integer identifier '" +
		string(id) + "'")
}

func errDSRMalformedForm(id, form []byte) error {
	return errors.New("DITStructureRule '" + string(id) +
		"' bears malformed FORM clause member '" +
		string(form) + "'")
}

func errDSRUnregSup(id, sup []byte) error {
	return errors.New("DITStructureRule '" + string(id) +
		"' bears unregistered SUP clause member '" +
		string(sup) + "'")
}

func errDSRDuplicate(id, sup []byte) error {
	return errors.New("DITStructureRule '" + string(id) +
		"' bears a duplicate SUP clause member '" +
		string(sup) + "'")
}

func errDSRUnregForm(id, form []byte) error {
	return errors.New("DITStructureRule '" + string(id) +
		"' bears unregistered FORM clause member '" +
		string(form) + "'")
}

func errDSRNameNotUnique(descr, other []byte) error {
	return errors.New("DITStructureRule NAME '" + string(descr) +
		"' already registered to '" + string(other) + "'")
}

func errDSRNotUnique(id []byte) error {
	return errors.New("DITStructureRule integer identifier '" +
		string(id) + "' already registered")
}

func errDSRNotFound(id []byte) error {
	return errors.New("DITStructureRule integer identifier '" +
		string(id) + "' not found")
}

func errDSRHasDependents(id []byte, count int) error {
	return errors.New("DITStructureRule '" + string(id) +
		"' has " + strconv.Itoa(count) +
		" dependent structure rules")
}
