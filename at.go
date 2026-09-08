package schema

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
)

/*
bit values used in AttributeTypeProperties.FLAGS map.
*/
const (
	flagSingle     uint8 = 1 << iota // 1
	flagCollective                   // 2
	flagNoUserMod                    // 4
	flagObsolete                     // 8
)

const (
	usageUser uint8 = iota // 0x0
	usageDir               // 0x1
	usageDist              // 0x2
	usageDSA               // 0x3
)

var usages = map[uint8]string{
	usageUser: "userApplications",
	usageDist: "distributedOperation",
	usageDir:  "directoryOperation",
	usageDSA:  "dSAOperation",
}

/*
AttributeTypeProperties implements a fast lookup matrix of properties related
to attribute types and attribute type relationships.
*/
type AttributeTypeProperties struct {
	MAP    map[string]uint             // type (k) nrml. name(s) and OID map to index (v)
	ID     map[uint][]byte             // type (k) bears numeric OID (v)
	TEXT   map[uint][]byte             // type (k) bears descriptive text (v)
	PRINC  map[uint][]byte             // type (k) to principal descriptor (v)
	DESCR  map[uint][][]byte           // type (k) bears descriptor(s) (v)
	SYNTAX map[uint]uint               // type (k) implements syntax (v)
	KIND   map[uint]uint8              // type (k) is what kind (v) [EQ:1|SS:2|OR:3]
	OBS    map[uint]struct{}           // type (k) is obsolete
	STRING map[uint]string             // type (k) in string representation (v)
	EQMR   map[uint]uint               // type (k) uses equality matching rule (v)
	SSMR   map[uint]uint               // type (k) uses substring matching rule (v)
	ORMR   map[uint]uint               // type (k) uses ordering matching rule (v)
	USAGE  map[uint]uint8              // type (k) [dirOp:1|distOp:2|dSAOp|3]; excl. userApps:0
	SUPT   map[uint]uint               // type (k) has super type (v)
	SUBT   map[uint]map[uint]struct{}  // type (k1) has dependent sub types (k2)
	OC     map[uint]map[uint]struct{}  // type (k1) has dependent classes (k2)
	DC     map[uint]map[uint]struct{}  // type (k1) has dependent content rules (k2)
	NF     map[uint]map[uint]struct{}  // type (k1) has dependent name forms (k2)
	EXT    map[uint]map[uint]Extension // type (k1) bears numbered (k2) extensions (v)
	CHAIN  map[uint][]uint             // type (k) is base type in chain
	PRIV   map[uint]struct{}           // type (k) is private
	FLAGS  map[uint]uint8              // type (k) has bool flags (v) [SNGL:1|COL:2|RO:4|OBS:8]
	UB     map[uint]uint               // type (k) is constrained by upper bounds >0
	IDX    map[uint]int                // type (k) is Nth in ORD collection
	ORD    []uint                      // collection of ordered type indices, maps to IDX

	schema *SubschemaSubentry
}

func initAttributeTypeProperties(sch *SubschemaSubentry) (atp *AttributeTypeProperties) {
	atp = &AttributeTypeProperties{
		MAP:    make(map[string]uint),
		ID:     make(map[uint][]byte),
		DESCR:  make(map[uint][][]byte),
		TEXT:   make(map[uint][]byte),
		PRINC:  make(map[uint][]byte),
		SYNTAX: make(map[uint]uint),
		FLAGS:  make(map[uint]uint8),
		EXT:    make(map[uint]map[uint]Extension),
		STRING: make(map[uint]string),
		IDX:    make(map[uint]int),
		EQMR:   make(map[uint]uint),
		SSMR:   make(map[uint]uint),
		ORMR:   make(map[uint]uint),
		SUPT:   make(map[uint]uint),
		USAGE:  make(map[uint]uint8),
		SUBT:   make(map[uint]map[uint]struct{}),
		OC:     make(map[uint]map[uint]struct{}),
		DC:     make(map[uint]map[uint]struct{}),
		NF:     make(map[uint]map[uint]struct{}),
		PRIV:   make(map[uint]struct{}),
		CHAIN:  make(map[uint][]uint),
		UB:     make(map[uint]uint),
		ORD:    make([]uint, 0),
		schema: sch,
	}

	return
}

/*
SetPrivate classifies the specified attribute as private. This is useful
if the schema administrator wishes to prevent disclosure of the specified
type in a given subschemaSubentry.

The attribute can still be interrogated directly like any other when this
package is being interacted with directly. However, when tied into a DSA,
end users ostensibly shall not be able to interrogate or interact with
a private type through conventional means.

Valid input values are the numeric OID or descriptor (name) associated
with the desired definition.

Case is not significant in the name matching process.

See also [ObjectClassProperties.SetPrivate].
*/
func (r *AttributeTypeProperties) SetPrivate(id []byte) {
	res, _, _, n := r.Resolve(id)
	if res != nil {
		r.schema.lock.Lock()
		r.PRIV[n] = struct{}{}
		r.schema.lock.Unlock()
	}
}

func (r AttributeTypeProperties) attributeTypeDescription(x any) (result bool, err error) {
	var input []byte
	switch tv := x.(type) {
	case []byte:
		input = tv
	case string:
		input = []byte(tv)
	default:
		err = errorBadType("attributeType")
		return
	}

	err = r.marshal(input, true)
	result = err == nil
	return
}

type attributeTypeDefinition struct {
	n         uint
	noid      []byte
	descrs    [][]byte
	text      []byte
	princ     []byte
	super     []byte
	syntax    []byte
	equality  []byte
	substring []byte
	ordering  []byte
	usage     uint8
	flags     uint8
	ub        uint
	ext       map[uint]Extension
}

/*
buildChain returns an instance of [][]byte following an attempt to
traverse the lookup tables and build a type chain.

If the return value is zero, the type has no super type and, thus,
no chain.
*/
func (r AttributeTypeProperties) buildChain(id uint) (chain []uint) {
	chain = []uint{}

	res := r.ID[id]
	if res != nil {
		chain = append(chain, id)
		if sup, found := r.SUPT[id]; found {
			chain = append(chain, r.buildChain(sup)...)
		}
		return
	}

	if len(chain) == 1 && chain[0] == id {
		// there is no chain, so we'll zero
		// out the return value.
		chain = []uint{}
	}

	return
}

/*
String returns the ordered string representation for every
registered attribute type that is not classified as private.
*/
func (r AttributeTypeProperties) String() string {
	var s string

	if L := len(r.ORD); L > 0 {
		bld := &strings.Builder{}

		for i := 0; i < L; i++ {
			if _, priv := r.PRIV[r.ORD[i]]; !priv {
				bld.WriteString(r.STRING[r.ORD[i]])
				bld.WriteRune(10)
			}
		}

		s = bld.String()
	}

	return s
}

/*
EffectiveSyntax returns the numeric OID ([]byte) associated with the effective
syntax of the attribute type indicated by id. The effective syntax shall be the
first concrete syntax reference encountered in the super chain of which the
desired type is the base member.

The return efs value contains the numeric OID, while the funk value contains the
verification function. If the numeric OID is nil, the function should not be called.

Valid input values are the numeric OID or descriptor (name) of the desired type.
Case is not significant in the name matching process.
*/
func (r AttributeTypeProperties) EffectiveSyntax(id []byte) (
	efs []byte,
	funk func(any) (bool, error),
) {
	noid, _, _, n := r.Resolve(id)
	if noid == nil {
		return
	}

	if syn, found := r.SYNTAX[n]; !found {
		// id does not bear an explicit SYNTAX clause value.
		var supn uint
		// See if a super type is defined for id.
		if supn, found = r.SUPT[n]; found {
			// recurse to id's super type and repeat.
			efs, funk = r.EffectiveSyntax(r.ID[supn])
		}
	} else {
		// syntax is local to id
		efs = r.schema.ls.ID[syn]
		funk = r.schema.ls.FUNC[syn]
	}

	return
}

/*
EffectiveEquality returns the numeric OID ([]byte) associated with the effective
equality matching rule of the attribute type indicated by id. The effective rule
shall be the first concrete equality reference encountered in the super chain of
which the desired type is the base member.

The return emr value contains the numeric OID, while the funk value contains the
matching function. If the numeric OID is nil, the function should not be called.

Valid input values are the numeric OID or descriptor (name) of the desired type.
Case is not significant in the name matching process.
*/
func (r AttributeTypeProperties) EffectiveEquality(id []byte) (
	emr []byte,
	funk func(any, any) (bool, error),
) {
	noid, _, _, n := r.Resolve(id)
	if noid == nil {
		return
	}

	if eql, found := r.EQMR[n]; !found {
		// id does not bear an explicit EQUALITY clause value.
		var supn uint
		// See if a super type is defined for id.
		if supn, found = r.SUPT[n]; found {
			// recurse to id's super type and repeat.
			emr, funk = r.EffectiveEquality(r.ID[supn])
		}
	} else {
		// rule is local to id
		emr = r.schema.mr.ID[eql]
		funk = r.schema.mr.EQFUNC[eql]
	}

	return
}

/*
EffectiveSubstring returns the numeric OID ([]byte) associated with the effective
substring matching rule of the attribute type indicated by id. The effective rule
shall be the first concrete substring reference encountered in the super chain of
which the desired type is the base member.

The return smr value contains the numeric OID, while the funk value contains the
matching function. If the numeric OID is nil, the function should not be called.

Valid input values are the numeric OID or descriptor (name) of the desired type.
Case is not significant in the name matching process.
*/
func (r AttributeTypeProperties) EffectiveSubstring(id []byte) (
	smr []byte,
	funk func(any, any) (bool, error),
) {
	noid, _, _, n := r.Resolve(id)
	if noid == nil {
		return
	}

	if sst, found := r.SSMR[n]; !found {
		// id does not bear an explicit SUBSTR clause value.
		var supn uint
		// See if a super type is defined for id.
		if supn, found = r.SUPT[n]; found {
			// recurse to id's super type and repeat.
			smr, funk = r.EffectiveSubstring(r.ID[supn])
		}
	} else {
		// rule is local to id
		smr = r.schema.mr.ID[sst]
		funk = r.schema.mr.SSFUNC[sst]
	}

	return
}

/*
EffectiveOrdering returns the numeric OID ([]byte) associated with the effective
ordering matching rule of the attribute type indicated by id. The effective rule
shall be the first concrete ordering reference encountered in the super chain of
which the desired type is the base member.

The return omr value contains the numeric OID, while the funk value contains the
matching function. If the numeric OID is nil, the function should not be called.

Valid input values are the numeric OID or descriptor (name) of the desired type.
Case is not significant in the name matching process.
*/
func (r AttributeTypeProperties) EffectiveOrdering(id []byte) (
	omr []byte,
	funk func(any, byte, any) (bool, error),
) {
	noid, _, _, n := r.Resolve(id)
	if noid == nil {
		return
	}

	if ord, found := r.ORMR[n]; !found {
		// id does not bear an explicit ORDERING clause value.
		var supn uint
		// See if a super type is defined for id.
		if supn, found = r.SUPT[n]; found {
			// recurse to id's super type and repeat.
			omr, funk = r.EffectiveOrdering(r.ID[supn])
		}
	} else {
		// rule is local to id
		omr = r.schema.mr.ID[ord]
		funk = r.schema.mr.ORFUNC[ord]
	}

	return
}

func (r *AttributeTypeProperties) commit(def attributeTypeDefinition, checkOnly bool) (err error) {
	if checkOnly {
		return
	}

	if err = r.check(&def); err != nil {
		return
	}

	r.MAP[string(def.noid)] = def.n
	r.ID[def.n] = def.noid

	r.commitDescr(def.n, def.noid, def.descrs)

	if def.ub > 0 {
		r.UB[def.n] = def.ub
	}

	if def.syntax != nil {
		_, _, synid := r.schema.ls.Resolve(def.syntax)
		if _, found := r.schema.ls.AT[synid]; !found {
			r.schema.ls.AT[synid] = make(map[uint]struct{})
		}
		r.schema.ls.AT[synid][def.n] = struct{}{}
		r.SYNTAX[def.n] = synid
	}

	r.commitSup(def)
	r.commitMR(def)

	if def.text != nil {
		r.TEXT[def.n] = def.text
	}

	if def.flags > 0 {
		r.FLAGS[def.n] = def.flags
	}

	if len(def.ext) > 0 {
		r.EXT[def.n] = def.ext
	}
	r.IDX[def.n] = len(r.ORD)
	r.ORD = append(r.ORD, def.n)

	r.makeString(def.n)

	r.schema.mu.commit(def)

	return
}

func (r *AttributeTypeProperties) commitSup(def attributeTypeDefinition) error {
	if def.super != nil {
		_, _, _, supid := r.Resolve(def.super)
		r.SUPT[def.n] = supid
		if _, found := r.SUBT[supid]; !found {
			r.SUBT[supid] = make(map[uint]struct{})
		}
		r.SUBT[supid][def.n] = struct{}{}
		r.CHAIN[def.n] = r.buildChain(def.n)
	}

	return nil
}

func (r *AttributeTypeProperties) commitDescr(n uint, noid []byte, descrs [][]byte) error {
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

func (r *AttributeTypeProperties) commitMR(def attributeTypeDefinition) error {
	if def.equality != nil {
		if mr, _, _, mrn := r.schema.mr.Resolve(def.equality); mr != nil {
			r.EQMR[def.n] = mrn
		}
	}

	if def.substring != nil {
		if mr, _, _, mrn := r.schema.mr.Resolve(def.substring); mr != nil {
			r.SSMR[def.n] = mrn
		}
	}

	if def.ordering != nil {
		if mr, _, _, mrn := r.schema.mr.Resolve(def.ordering); mr != nil {
			r.ORMR[def.n] = mrn
		}
	}

	return nil
}

func (r *AttributeTypeProperties) unregister(id []byte) (err error) {
	noid, _, descrs, n := r.Resolve(id)
	if noid == nil {
		return // no error needed
	}

	idx := r.IDX[n]

	if err = r.checkDeps(n); err != nil {
		return
	}

	syn := r.SYNTAX[n]
	delete(r.schema.ls.AT[syn], n)
	delete(r.SYNTAX, n)

	// remove matching rule references, and
	// truncate applied ordering.
	r.unregisterTypeMR(n)

	delete(r.UB, n)
	delete(r.OC, n)
	delete(r.DC, n)
	delete(r.NF, n)
	delete(r.ID, n)
	delete(r.EXT, n)
	delete(r.IDX, n)
	delete(r.EQMR, n)
	delete(r.SSMR, n)
	delete(r.ORMR, n)
	delete(r.TEXT, n)
	delete(r.SUPT, n)
	delete(r.SUBT, n)
	delete(r.PRIV, n)
	delete(r.PRINC, n)
	delete(r.FLAGS, n)
	delete(r.USAGE, n)
	delete(r.DESCR, n)
	delete(r.CHAIN, n)
	delete(r.STRING, n)
	delete(r.MAP, string(noid))

	for i := 0; i < len(descrs); i++ {
		delete(r.MAP, string(lc(descrs[i])))
	}
	delete(r.MAP, string(noid))

	// truncate ordered collection, removing
	// the now unregistered OID.
	r.ORD = append(r.ORD[:idx], r.ORD[idx+1:]...)

	// rebuild indices list, as it will no longer
	// be accurate following this unregistration.
	r.IDX = make(map[uint]int)
	for i := 0; i < len(r.ORD); i++ {
		r.IDX[r.ORD[i]] = i
	}

	return
}

func (r *AttributeTypeProperties) checkDeps(n uint) (err error) {
	id := r.ID[n]

	if L := len(r.SUBT[n]); L > 0 {
		err = errATDep(id, " sub types", L)
	} else if L = len(r.OC[n]); L > 0 {
		err = errATDep(id, " object classes", L)
	} else if L = len(r.DC[n]); L > 0 {
		err = errATDep(id, " content rules", L)
	} else if L = len(r.NF[n]); L > 0 {
		err = errATDep(id, " name forms", L)
	}

	return
}

func (r *AttributeTypeProperties) unregisterTypeMR(n uint) {
	eqmr := r.EQMR[n]

	delete(r.schema.mu.APPLIES[eqmr], n)
	delete(r.EQMR, n)

	for i := 0; i < len(r.schema.mu.APPORD[eqmr]); i++ {
		nn := r.schema.mu.APPORD[eqmr][i]
		if n == r.schema.mu.APPORD[eqmr][nn] {
			r.schema.mu.APPORD[eqmr] = append(r.schema.mu.APPORD[eqmr][:],
				r.schema.mu.APPORD[eqmr][i+1:]...)
			break
		}
	}

	ssmr := r.SSMR[n]
	delete(r.schema.mu.APPLIES[ssmr], n)
	delete(r.SSMR, n)

	for i := 0; i < len(r.schema.mu.APPORD[ssmr]); i++ {
		nn := r.schema.mu.APPORD[ssmr][i]
		if n == r.schema.mu.APPORD[ssmr][nn] {
			r.schema.mu.APPORD[ssmr] = append(r.schema.mu.APPORD[ssmr][:i],
				r.schema.mu.APPORD[ssmr][i+1:]...)
			break
		}
	}

	ormr := r.ORMR[n]
	delete(r.schema.mu.APPLIES[ormr], n)
	delete(r.ORMR, n)

	for i := 0; i < len(r.schema.mu.APPORD[ormr]); i++ {
		nn := r.schema.mu.APPORD[ormr][i]
		if n == r.schema.mu.APPORD[ormr][nn] {
			r.schema.mu.APPORD[ormr] = append(r.schema.mu.APPORD[ormr][:i],
				r.schema.mu.APPORD[ormr][i+1:]...)
			break
		}
	}
}

func (r *AttributeTypeProperties) marshal(in []byte, checkOnly bool) (err error) {
	input := bytes.TrimSpace(trimDefinitionLabelToken(in))
	tkz := newSchemaTokenizer(input)
	tkz.startTokenParen()

	noid := tkz.this()
	if at, _, _, _ := r.Resolve(noid); at != nil && !checkOnly {
		err = errATNotUnique(noid)
		return
	}

	def := attributeTypeDefinition{
		n:    uint(len(r.ORD)),
		noid: noid,
		ext:  make(map[uint]Extension),
	}

	seen := make(map[string]struct{})

	for tkz.next() && err == nil {
		token := string(tkz.this())
		if _, found := seen[token]; found {
			err = errors.New("AttributeType clause '" + token +
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
		case "SUP":
			def.super = tkz.nextToken()
			err = r.checkSupClause(def.noid, def.super)
			seen[token] = struct{}{}
		case "SUBSTR", "SUBSTRING", "EQUALITY", "ORDERING", "SYNTAX":
			err = r.checkSyntaxMatchingRules(&def, token, tkz)
			seen[token] = struct{}{}
		case "SINGLE-VALUE", "COLLECTIVE", "OBSOLETE", "NO-USER-MODIFICATION":
			err = def.handleBoolean(token)
			seen[token] = struct{}{}
		case "USAGE":
			usage := tkz.nextToken()
			err = def.handleUsage(usage)
			seen[token] = struct{}{}
		default:
			var ext Extension
			ext, err = marshalExtension(token, "attributeType", tkz)
			def.ext[uint(len(def.ext))] = ext
			seen[token] = struct{}{}
		}
	}

	return
}

func (r *AttributeTypeProperties) check(
	def *attributeTypeDefinition,
) (err error) {

	for i := 0; i < len(def.descrs); i++ {
		item := lc(def.descrs[i])
		if reg, found := r.MAP[string(item)]; found {
			err = errATNameReg(def.descrs[i], r.PRINC[reg])
			return
		}
	}

	if sup := def.super; sup != nil {
		if def.super, _, _, _ = r.Resolve(sup); def.super == nil {
			err = errATUnregClauseMember(def.noid, sup, `SUP`)
			return
		}
	}

	if def.syntax != nil {
		if syn, _, _ := r.schema.ls.Resolve(def.syntax); syn == nil {
			err = errATUnregClauseMember(def.noid, def.syntax, `SYNTAX`)
			return
		}
	}

	if eql := def.equality; eql != nil {
		if def.equality, _, _, _ = r.schema.mr.Resolve(eql); def.equality == nil {
			err = errATUnregClauseMember(def.noid, eql, `EQUALITY`)
			return
		}
	}

	if ord := def.ordering; ord != nil {
		if def.ordering, _, _, _ = r.schema.mr.Resolve(ord); def.ordering == nil {
			err = errATUnregClauseMember(def.noid, ord, `ORDERING`)
			return
		}
	}

	if sst := def.substring; sst != nil {
		if def.substring, _, _, _ = r.schema.mr.Resolve(sst); def.substring == nil {
			err = errATUnregClauseMember(def.noid, sst, `SUBSTR`)
			return
		}
	}

	// At an absolute minimum, a type must bear a super
	// type or an LDAP syntax.
	if def.syntax == nil && def.super == nil {
		err = errATBadForm(def.noid)
	}

	return
}

func (r *AttributeTypeProperties) checkSupClause(
	noid []byte,
	clause []byte,
) (err error) {
	if !isAttribute(clause) {
		err = errATMalformedType(noid, clause)
	}

	return
}

func (r *AttributeTypeProperties) makeString(n uint) (err error) {
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

	flags, _ := r.FLAGS[n]
	bld.Write(stringBooleanClause(`OBSOLETE`, flags&flagObsolete > 0))

	if super, found := r.SUPT[n]; found {
		princ := r.PRINC[super]
		bld.Write(definitionMVDescriptors(`SUP`, princ))
	}

	bld.Write(r.syntaxMatchingRuleClauses(n))
	bld.Write(r.mutexBooleanString(n))
	bld.Write(stringBooleanClause(`NO-USER-MODIFICATION`, flags&flagNoUserMod > 0))

	if usage, found := r.USAGE[n]; found {
		bld.WriteString(` USAGE `)
		bld.WriteString(usages[usage])
	}

	if ext, found := r.EXT[n]; found {
		bld.Write(stringExtensions(ext))
	}

	bld.WriteRune(' ')
	bld.WriteRune(')')
	r.STRING[n] = bld.String()

	return
}

func (r *AttributeTypeProperties) checkSyntaxMatchingRules(
	def *attributeTypeDefinition,
	token string,
	tkz *schemaTokenizer,
) (err error) {
	switch token {
	case "EQUALITY":
		def.equality = tkz.nextToken()
	case "ORDERING":
		def.ordering = tkz.nextToken()
	case "SUBSTR", "SUBSTRING":
		def.substring = tkz.nextToken()
	case "SYNTAX":
		x := tkz.nextToken()

		// handle upper bounds, if present
		if idx := bytes.IndexRune(x, '{'); idx != -1 {
			syntax := x[:idx]
			def.syntax = syntax

			raw := bytes.Trim(x[idx+1:], `}`)
			var _mub int
			if _mub, err = strconv.Atoi(string(raw)); err == nil {
				if raw[0] == '-' {
					err = errATNegUB(def.noid, raw)
					break
				}
				def.ub = uint(_mub)
			}
		} else {
			def.syntax = x
		}
	}

	return
}

func (r *attributeTypeDefinition) handleBoolean(token string) (err error) {
	switch token {
	case "OBSOLETE":
		r.flags |= flagObsolete
	case "NO-USER-MODIFICATION":
		r.flags |= flagNoUserMod
	case "SINGLE-VALUE":
		if r.flags&flagCollective > 0 {
			err = errCollectiveSingle
			break
		}
		r.flags |= flagSingle
	case "COLLECTIVE":
		if r.flags&flagSingle > 0 {
			err = errCollectiveSingle
			break
		}
		r.flags |= flagCollective
	}

	return
}

func (r *attributeTypeDefinition) handleUsage(token []byte) (err error) {
	switch tk := string(lc(token)); tk {
	case "userapplications":
	case "dsaoperation":
		r.usage = usageDSA
	case "directoryoperation":
		r.usage = usageDir
	case "distributedoperation":
		r.usage = usageDist
	default:
		err = errATBadUsage(r.noid, tk)
	}

	return
}

func (r AttributeTypeProperties) mutexBooleanString(id uint) (clause []byte) {
	flags, found := r.FLAGS[id]
	if !found {
		return
	}
	if flags&flagSingle > 0 {
		clause = []byte(` SINGLE-VALUE`)
	} else if flags&flagCollective > 0 {
		clause = []byte(` COLLECTIVE`)
	}

	return
}

func (r AttributeTypeProperties) syntaxMatchingRuleClauses(id uint) (clause []byte) {
	bld := &strings.Builder{}
	if eq, found := r.EQMR[id]; found {
		bld.WriteString(` EQUALITY `)
		bld.Write(r.schema.mr.PRINC[eq])
	}

	if or, found := r.ORMR[id]; found {
		bld.WriteString(` ORDERING `)
		bld.Write(r.schema.mr.PRINC[or])
	}

	if ss, found := r.SSMR[id]; found {
		bld.WriteString(` SUBSTR `)
		bld.Write(r.schema.mr.PRINC[ss])
	}

	if syn, found := r.SYNTAX[id]; found {
		soid := r.schema.ls.ID[syn]
		bld.WriteString(` SYNTAX `)
		bld.Write(soid)
		if ub, _ := r.UB[id]; ub > 0 {
			bld.WriteRune('{')
			bld.WriteString(strconv.FormatUint(uint64(ub), 10))
			bld.WriteRune('}')
		}
	}

	clause = []byte(bld.String())

	return
}

/*
Resolve returns a numeric OID, a principal descriptor and slices of all descriptors
following an attempt to resolve the input term to a registered definition.

Valid input values are the numeric OID of the definition or any of its official
descriptors (names). Case is not significant in the name matching process.
*/
func (r AttributeTypeProperties) Resolve(term []byte) (noid, princ []byte, descrs [][]byte, n uint) {
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
Get returns the pre-generated string representation of the definition
bearing the input id value. A zero string is returned if not found.

The id argument may be the numeric OID or the descriptor (name) of the
desired definition.

Case is not significant in the name matching process.

Note that attribute definitions classified private WILL be returned,
if found.
*/
func (r AttributeTypeProperties) Get(id []byte) string {
	var s string
	if noid, _, _, n := r.Resolve(id); noid != nil {
		s = r.STRING[n]
	}

	return s
}

/*
RegisterAttributeType returns an error following an attempt to parse the input
def bytes. If successful, the components of def are written to the underlying
matrices.
*/
func (r *SubschemaSubentry) RegisterAttributeType(def []byte) (err error) {
	if !bytes.HasPrefix(lc(def), lc([]byte(headerTokens[5]))) {
		def = append([]byte(headerTokens[5]+": "), def...)
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	err = r.at.marshal(def, false)

	return
}

/*
UnregisterAttributeType returns an error following an attempt to remove the
specified definition from the receiver instance. Valid input values are the
numeric OID or descriptor (name) of the desired definition.

Case is not significant in the name matching process.
*/
func (r *SubschemaSubentry) UnregisterAttributeType(id []byte) (err error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	err = r.at.unregister(id)
	return
}

func errATNotUnique(noid []byte) error {
	return errors.New("AttributeType '" + string(noid) + "' already registered")
}

func errATNotFound(noid []byte) error {
	return errors.New("AttributeType '" + string(noid) + "' not found")
}

func errATNameReg(name, other []byte) error {
	return errors.New("AttributeType NAME '" + string(name) +
		"' already registered to '" + string(other) + "'")
}

func errATMalformedType(noid, sup []byte) error {
	return errors.New("AttributeType '" + string(noid) +
		"' bears malformed super type '" + string(sup) + "'")
}

func errATBadUsage(noid []byte, token string) error {
	return errors.New("AttributeType '" + string(noid) +
		"bears invalid USAGE clause member '" + token + "'")
}

func errATDep(noid []byte, dep string, count int) error {
	return errors.New("AttributeType '" + string(noid) +
		"' is depended upon by " + strconv.Itoa(count) +
		dep)
}

func errATUnregClauseMember(noid, value []byte, clause string) error {
	return errors.New("AttributeType '" + string(noid) +
		"' bears unregistered " + clause + " clause member '" +
		string(value) + "'")
}

func errATNegUB(noid, ub []byte) error {
	return errors.New("AttributeType '" + string(noid) +
		"' bears a negative UB '" + string(ub) + "'")
}

func errATBadForm(noid []byte) error {
	return errors.New("AttributeType '" + string(noid) +
		"' must bear a SUP or SYNTAX at a minimum")
}
