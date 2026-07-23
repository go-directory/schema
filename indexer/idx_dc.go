package indexer

import (
	"strings"

	"github.com/go-directory/schema"
)

func (r DITContentRuleProperties) Resolve(def string) (noid, ident string, names []string) {
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
Index returns an integer value following an attempt to call an DITContentRule
(def) original index number within the source *[schema.DITContentRules]
instance.
*/
func (r DITContentRuleProperties) Index(def string) (idx int) {
	noid, _, _ := r.Resolve(def)
	var ok bool
	if idx, ok = r.SrcIndex[noid]; !ok {
		idx = -1
	}

	return
}

func (r *Index) seedDC(sch *schema.SubschemaSubentry) {
	r.DC = DITContentRuleProperties{}
	r.DC.SrcIndex = make(map[string]int)
	r.DC.Princ = make(map[string]string)
	r.DC.O2D = make(map[string][]string)
	r.DC.D2O = make(map[string]string)
	r.temp.DC = make(map[string]*schema.DITContentRule)

	for i := 0; i < sch.DITContentRules.Len(); i++ {
		def := sch.DITContentRules.Index(i)
		r.DC.SrcIndex[def.NumericOID] = i
		r.DC.Princ[def.NumericOID] = def.Identifier()
		r.DC.O2D[def.NumericOID] = def.Name
		for _, name := range def.Name {
			name = strings.ToLower(name)
			r.DC.D2O[name] = def.NumericOID
		}
		r.temp.DC[def.NumericOID] = def
	}
}

func (r *Index) loadDC() (err error) {

	r.DC.Aux = make(map[string][]string)
	r.DC.Must = make(map[string][]string)
	r.DC.May = make(map[string][]string)
	r.DC.Not = make(map[string][]string)
	r.DC.Obsolete = make(map[string]struct{})

	for noid, rule := range r.temp.DC {
		if rule.Obsolete {
			r.DC.Obsolete[noid] = struct{}{}
		}
		var aux []string
		for j := 0; j < len(rule.Aux); j++ {
			if a, _, _ := r.OC.Resolve(rule.Aux[j]); a != "" {
				aux = append(aux, a)
			}
		}
		if len(aux) > 0 {
			r.DC.Aux[noid] = aux
		}
		var must []string
		for j := 0; j < len(rule.Must); j++ {
			if a, _, _ := r.AT.Resolve(rule.Must[j]); a != "" {
				must = append(must, a)
			}
		}
		if len(must) > 0 {
			r.DC.Must[noid] = must
		}
		var may []string
		for j := 0; j < len(rule.May); j++ {
			if a, _, _ := r.AT.Resolve(rule.May[j]); a != "" {
				may = append(may, a)
			}
		}
		if len(may) > 0 {
			r.DC.May[noid] = may
		}
		var not []string
		for j := 0; j < len(rule.Not); j++ {
			if a, _, _ := r.AT.Resolve(rule.Not[j]); a != "" {
				not = append(not, a)
			}
		}
		if len(not) > 0 {
			r.DC.Not[noid] = not
		}
	}

	return
}

type DITContentRuleProperties struct {
	O2D      map[string][]string // numeric ID to descriptor(s)
	D2O      map[string]string   // descriptor to numeric ID
	Princ    map[string]string
	Obsolete map[string]struct{} // obsolete
	Aux      map[string][]string // auxiliary object classes
	Must     map[string][]string // mandatory attribute types
	May      map[string][]string // optional attribute types
	Not      map[string][]string // prohibited attribute types
	SrcIndex map[string]int      // integer index in schema.DITContentRules
}
