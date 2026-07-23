package indexer

import (
	"strings"

	"github.com/go-directory/schema"
)

func (r MatchingRuleProperties) Resolve(def string) (noid, ident string, _ []string) {
	if len(def) == 0 {
		return
	}

	def = strings.ToLower(def)
	f := rune(def[0])

	if 'a' <= f && f <= 'z' {
		// def is a descriptor OID
		var ok bool
		if noid, ok = r.D2O[def]; ok {
			ident, _ = r.Princ[noid]
		}
	} else if '0' <= f && f <= '2' {
		// def is a numeric OID
		var ok bool
		if ident, ok = r.Princ[def]; ok {
			noid = def
		}
	}

	return
}

/*
Index returns an integer value following an attempt to call an MatchingRule
(def) original index number within the source *[schema.MatchingRules]
instance.
*/
func (r MatchingRuleProperties) Index(def string) (idx int) {
	noid, _, _ := r.Resolve(def)
	var ok bool
	if idx, ok = r.SrcIndex[noid]; !ok {
		idx = -1
	}

	return
}

func (r *Index) seedMR(sch *schema.SubschemaSubentry) {
	r.MR = MatchingRuleProperties{}
	r.MR.Princ = make(map[string]string)
	r.MR.SrcIndex = make(map[string]int)
	r.MR.O2D = make(map[string]string)
	r.MR.D2O = make(map[string]string)
	r.temp.MR = make(map[string]*schema.MatchingRule)

	for i := 0; i < sch.MatchingRules.Len(); i++ {
		def := sch.MatchingRules.Index(i)
		r.MR.SrcIndex[def.NumericOID] = i
		ident := def.Identifier()
		r.MR.Princ[def.NumericOID] = ident
		r.MR.O2D[def.NumericOID] = ident
		r.MR.D2O[strings.ToLower(ident)] = def.NumericOID
		r.temp.MR[def.NumericOID] = def
		r.LS.MR[def.Syntax] = def.NumericOID
	}
}

func (r *Index) seedMU(sch *schema.SubschemaSubentry) {
	r.temp.MU = make(map[string]*schema.MatchingRuleUse)

	for i := 0; i < sch.MatchingRuleUses.Len(); i++ {
		def := sch.MatchingRuleUses.Index(i)
		r.temp.MU[def.NumericOID] = def
	}
}

func (r *Index) loadMR() (err error) {

	r.MR.LS = make(map[string]string)
	r.MR.Type = make(map[string]string)
	r.MR.Applies = make(map[string][]string)

	for _, v := range r.temp.MR {
		noid := v.NumericOID
		if v.Obsolete {
			r.MR.Obsolete[noid] = struct{}{}
		}
		r.MR.LS[noid] = v.Syntax

		id := strings.ToLower(v.Identifier())
		if strings.Contains(id, `substring`) {
			r.MR.Type[noid] = `S`
		} else if strings.Contains(id, `ordering`) {
			r.MR.Type[noid] = `O`
		} else {
			// must be EQUALITY
			r.MR.Type[noid] = `E`
		}
	}

	for k, v := range r.temp.MU {
		if len(v.Applies) > 0 {
			r.MR.Applies[k] = make([]string, 0)
			for a := 0; a < len(v.Applies); a++ {
				at, _ := r.AT.D2O[v.Applies[a]]
				r.MR.Applies[k] = append(r.MR.Applies[k], at)
			}
		}
	}

	return
}

type MatchingRuleProperties struct {
	O2D      map[string]string // numeric OID to descriptor
	D2O      map[string]string // descriptor to numeric OID
	Princ    map[string]string
	LS       map[string]string   // matching rule (k) implements syntax (v)
	Type     map[string]string   // matching rule (k) is what type of matching rule (v)
	Applies  map[string][]string // matching rule (k) applies to which attribute types (v)
	Obsolete map[string]struct{} // obsolete
	SrcIndex map[string]int      // integer index in schema.MatchingRules
}
