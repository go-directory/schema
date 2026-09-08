package schema

import (
	"strings"
)

type MatchingRuleUseProperties struct {
	APPLIES map[uint]map[uint]struct{}
	APPORD  map[uint][]uint
	STRING  map[uint]string

	schema *SubschemaSubentry
}

func initMatchingRuleUseProperties(sch *SubschemaSubentry) (mup *MatchingRuleUseProperties) {
	mup = &MatchingRuleUseProperties{
		APPLIES: make(map[uint]map[uint]struct{}),
		APPORD:  make(map[uint][]uint),
		STRING:  make(map[uint]string),
		schema:  sch,
	}

	return
}

func (r *MatchingRuleUseProperties) commit(def attributeTypeDefinition) {
	for _, mr := range [][]byte{
		def.equality,
		def.substring,
		def.ordering,
	} {
		if mr != nil {
			if moid, _, _, n := r.schema.mr.Resolve(mr); moid != nil {
				if _, found := r.APPLIES[n]; !found {
					r.APPLIES[n] = make(map[uint]struct{})
				}
				r.APPLIES[n][def.n] = struct{}{}
				r.APPORD[n] = append(r.APPORD[n], def.n)
				r.makeString(n)
			}
		}
	}
}

/*
makeString returns an error following an attempt to create the string
representation of the specified definition.

If successful, the string value is written to the underlying receiver
STRING map.

If the input id (numeric OID) is not registered, an error is returned.
*/
func (r *MatchingRuleUseProperties) makeString(id uint) (err error) {
	applies, _ := r.APPORD[id]
	if len(applies) == 0 {
		return
	}

	noid := r.schema.mr.ID[id]
	descrs := r.schema.mr.DESCR[id]

	bld := &strings.Builder{}
	bld.WriteRune('(')
	bld.WriteRune(' ')
	bld.Write(noid)
	bld.Write(definitionName(descrs))

	if text, found := r.schema.mr.TEXT[id]; found {
		bld.Write(definitionDescription(text))
	}

	if _, found := r.schema.mr.OBS[id]; found {
		bld.WriteString(` OBSOLETE`)
	}

	var appl [][]byte
	for i := 0; i < len(applies); i++ {
		princ := r.schema.at.PRINC[applies[i]]
		appl = append(appl, princ)
	}
	bld.Write(definitionMVDescriptors(`APPLIES`, appl))

	if ext, found := r.schema.mr.EXT[id]; found {
		bld.Write(stringExtensions(ext))
	}

	bld.WriteRune(' ')
	bld.WriteRune(')')
	r.STRING[id] = bld.String()
	return
}
