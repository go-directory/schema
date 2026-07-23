package indexer

import (
	"strings"

	"github.com/go-directory/schema"
)

func (r DITStructureRuleProperties) Resolve(def string) (rid, ident string, names []string) {
	if len(def) == 0 {
		return
	}

	def = strings.ToLower(def)
	f := rune(def[0])

	if 'a' <= f && f <= 'z' {
		// def is a descriptor rule ID
		var ok bool
		if rid, ok = r.D2O[def]; ok {
			if ident, ok = r.Princ[rid]; ok {
				names, _ = r.O2D[rid]
			}
		}
	} else if '0' <= f && f <= '2' {
		// def is a numeric rule ID
		var ok bool
		if ident, ok = r.Princ[def]; ok {
			rid = def
			names, _ = r.O2D[rid]
		}
	}

	return
}

/*
Index returns an integer value following an attempt to call an DITStructureRule
(def) original index number within the source *[schema.DITStructureRules]
instance.
*/
func (r DITStructureRuleProperties) Index(def string) (idx int) {
	rid, _, _ := r.Resolve(def)
	var ok bool
	if idx, ok = r.SrcIndex[rid]; !ok {
		idx = -1
	}

	return
}

func (r DITStructureRuleProperties) SubRules(def string) []string {
	var sub []string
	if d, _, _ := r.Resolve(def); d != "" {
		sub, _ = r.Sub[d]
	}
	return sub
}

func (r DITStructureRuleProperties) SuperRules(def string) []string {
	var sub []string
	if d, _, _ := r.Resolve(def); d != "" {
		sub, _ = r.Sup[d]
	}
	return sub
}

func (r *Index) seedDS(sch *schema.SubschemaSubentry) {
	r.DS = DITStructureRuleProperties{}
	r.DS.SrcIndex = make(map[string]int)
	r.DS.Princ = make(map[string]string)
	r.DS.O2D = make(map[string][]string)
	r.DS.D2O = make(map[string]string)
	r.temp.DS = make(map[string]*schema.DITStructureRule)

	for i := 0; i < sch.DITStructureRules.Len(); i++ {
		def := sch.DITStructureRules.Index(i)
		r.DS.SrcIndex[def.RuleID] = i
		r.DS.Princ[def.RuleID] = def.Identifier()
		r.DS.O2D[def.RuleID] = def.Name
		for _, name := range def.Name {
			name = strings.ToLower(name)
			r.DS.D2O[name] = def.RuleID
		}
		r.temp.DS[def.RuleID] = def
	}
}

func (r *Index) loadDS() (err error) {

	r.DS.Sup = make(map[string][]string)
	r.DS.Sub = make(map[string][]string)
	r.DS.NF = make(map[string]string)
	r.DS.NOC = make(map[string]string)
	r.DS.Obsolete = make(map[string]struct{})

	for rid, rule := range r.temp.DS {
		if rule.Obsolete {
			r.DS.Obsolete[rid] = struct{}{}
		}

		var form string
		nf := r.NF.D2O[rule.Form]
		if len(nf) == 0 {
			form = rule.Form
		} else {
			form = nf
		}
		r.DS.NF[rid] = form

		noc := rule.NamedObjectClass()
		r.DS.NOC[rid] = noc.NumericOID

		var supers []string
		for j := 0; j < len(rule.SuperRules); j++ {
			if super, _, _ := r.DS.Resolve(rule.SuperRules[j]); super != "" {
				supers = append(supers, super)
			}
		}
		if len(supers) > 0 {
			r.DS.Sup[rid] = supers
		}
	}

	for _, v := range r.DS.Sup {
		for _, x := range v {
			var subs []string
			for _, rule := range r.temp.DS {
				for _, s := range rule.SuperRules {
					if s == x {
						subs = append(subs, rule.RuleID)
					}
				}
			}

			if len(subs) > 0 {
				r.DS.Sub[x] = subs
			}
		}
	}

	return
}

type DITStructureRuleProperties struct {
	O2D      map[string][]string // numeric ID to descriptor(s)
	D2O      map[string]string   // descriptor to numeric ID
	Princ    map[string]string
	Sup      map[string][]string // structure rule (k) has super types (v)
	Sub      map[string][]string // structure rule (k) has sub types (v)
	Obsolete map[string]struct{} // obsolete
	NF       map[string]string   // structure rule (k) uses name form (v)
	NOC      map[string]string   // structure rule (k) uses objectClass (v)
	SrcIndex map[string]int      // integer index in schema.DITStructureRules
}
