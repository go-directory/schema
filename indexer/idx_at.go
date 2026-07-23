package indexer

import (
	"strings"

	"github.com/go-directory/schema"
)

const (
	flagSingle     uint8 = 1 << iota // 1
	flagCollective                   // 2
	flagNoUserMod                    // 4
	flagObsolete                     // 8
)

func (r AttributeTypeProperties) Resolve(def string) (noid, ident string, names []string) {
	if len(def) == 0 {
		return
	}

	def = strings.ToLower(def)
	f := rune(def[0])

	if 'a' <= f && f <= 'z' {
		// def is a descriptor OID
		var ok bool
		if noid, ok = r.D2O[def]; ok {
			if ident, ok = r.Princ[noid]; ok {
				names, _ = r.O2D[noid]
			}
		}
	} else if '0' <= f && f <= '2' {
		// def is a numeric OID
		var ok bool
		if ident, ok = r.Princ[def]; ok {
			noid = def
			names, _ = r.O2D[noid]
		}
	}

	return
}

/*
Index returns an integer value following an attempt to call an AttributeType
(def) original index number within the source *[schema.AttributeTypes]
instance.
*/
func (r AttributeTypeProperties) Index(def string) (idx int) {
	noid, _, _ := r.Resolve(def)
	var ok bool
	if idx, ok = r.SrcIndex[noid]; !ok {
		idx = -1
	}

	return
}

func (r *Index) seedAT(sch *schema.SubschemaSubentry) {
	r.AT = AttributeTypeProperties{}
	r.AT.Princ = make(map[string]string)
	r.AT.SrcIndex = make(map[string]int)
	r.AT.Flags = make(map[string]uint8)
	r.AT.MR = make(map[string]string)
	r.AT.LS = make(map[string]string)
	r.AT.O2D = make(map[string][]string)
	r.AT.D2O = make(map[string]string)
	r.temp.AT = make(map[string]*schema.AttributeType)

	for i := 0; i < sch.AttributeTypes.Len(); i++ {
		def := sch.AttributeTypes.Index(i)
		for _, rule := range []func() *schema.MatchingRule{
			def.EffectiveEquality,
			def.EffectiveSubstring,
			def.EffectiveOrdering,
		} {
			if mr := rule(); mr != nil {
				r.AT.MR[def.NumericOID] = mr.NumericOID
				break
			}
		}

		efs := def.EffectiveSyntax()
		r.AT.Princ[def.NumericOID] = def.Identifier()
		r.AT.LS[def.NumericOID] = efs.NumericOID
		r.AT.SrcIndex[def.NumericOID] = i
		r.AT.O2D[def.NumericOID] = def.Name
		r.LS.AT[efs.NumericOID] = def.NumericOID
		r.AT.attributeBools(def)
		for _, name := range def.Name {
			name = strings.ToLower(name)
			r.AT.D2O[name] = def.NumericOID
		}
		r.temp.AT[def.NumericOID] = def
	}
}

func (r *Index) loadAT() (err error) {

	r.AT.UB = make(map[string]uint)
	r.AT.Usage = make(map[string]string)
	r.AT.Sup = make(map[string]string)
	r.AT.Sub = make(map[string][]string)

	for noid, attr := range r.temp.AT {
		if ub := attr.MinUpperBounds; ub > 0 {
			r.AT.UB[noid] = ub
		}

		if attr.Usage != "" {
			r.AT.Usage[noid] = attr.Usage
		}

		syn := r.AT.LS[noid]
		r.AT.LS[noid] = syn

		for _, rule := range []string{
			attr.Equality,
			attr.Substring,
			attr.Ordering,
		} {
			if rule != "" {
				r.AT.MR[noid], _, _ = r.MR.Resolve(rule)
			}
		}

		if sup := attr.SuperType; sup != "" {
			r.AT.Sup[noid], _, _ = r.AT.Resolve(sup)
		}
	}

	for _, v := range r.AT.Sup {
		var subs []string
		for noid, attr := range r.temp.AT {
			if attr.SuperType == v {
				subs = append(subs, noid)
			}
		}

		if len(subs) > 0 {
			r.AT.Sub[v] = subs
		}
	}

	return
}

func (r *AttributeTypeProperties) attributeBools(attr *schema.AttributeType) {
	noid := attr.NumericOID

	var f uint8
	if attr.Single {
		f |= flagSingle
	}
	if attr.Collective {
		f |= flagCollective
	}
	if attr.NoUserModification {
		f |= flagNoUserMod
	}
	if attr.Obsolete {
		f |= flagObsolete
	}
	r.Flags[noid] = f
}

func (r AttributeTypeProperties) IsSingleValued(def string) bool {
	var is bool
	if f, ok := r.Flags[def]; ok {
		is = f&flagSingle != 0
	}
	return is
}

func (r AttributeTypeProperties) IsCollective(def string) bool {
	var is bool
	if f, ok := r.Flags[def]; ok {
		is = f&flagCollective != 0
	}
	return is
}

func (r AttributeTypeProperties) IsObsolete(def string) bool {
	var is bool
	if f, ok := r.Flags[def]; ok {
		is = f&flagObsolete != 0
	}
	return is
}

func (r AttributeTypeProperties) IsNotUserModifiable(def string) bool {
	var is bool
	if f, ok := r.Flags[def]; ok {
		is = f&flagNoUserMod != 0
	}
	return is
}

type AttributeTypeProperties struct {
	O2D      map[string][]string // numeric OID to descriptor(s)
	D2O      map[string]string   // descriptor to numeric OID
	Princ    map[string]string   // attribute (k) has principal identifier (v)
	LS       map[string]string   // attribute (k) uses syntax (v)
	MR       map[string]string   // attribute (k) uses matching rule (v)
	Flags    map[string]uint8    // attribute (k) has bool flags (v)
	Usage    map[string]string   // attribute (k) is <usage>
	Sup      map[string]string   // attribute (k) has super type (v)
	Sub      map[string][]string // attribute (k) has sub types (v)
	SrcIndex map[string]int      // integer index in schema.AttributeTypes
	UB       map[string]uint     // upper bounds (max value size)
}
