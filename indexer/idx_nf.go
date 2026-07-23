package indexer

import (
	"strings"

	"github.com/go-directory/schema"
)

func (r NameFormProperties) Resolve(def string) (noid, ident string, names []string) {
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
Index returns an integer value following an attempt to call an NameForm
(def) original index number within the source *[schema.NameForms]
instance.
*/
func (r NameFormProperties) Index(def string) (idx int) {
	noid, _, _ := r.Resolve(def)
	var ok bool
	if idx, ok = r.SrcIndex[noid]; !ok {
		idx = -1
	}

	return
}

func (r *Index) seedNF(sch *schema.SubschemaSubentry) {
	r.NF = NameFormProperties{}
	r.NF.SrcIndex = make(map[string]int)
	r.NF.Princ = make(map[string]string)
	r.NF.O2D = make(map[string][]string)
	r.NF.D2O = make(map[string]string)
	r.NF.SOC = make(map[string]string)
	r.temp.NF = make(map[string]*schema.NameForm)

	for i := 0; i < sch.NameForms.Len(); i++ {
		def := sch.NameForms.Index(i)
		r.NF.SrcIndex[def.NumericOID] = i
		r.NF.Princ[def.NumericOID] = def.Identifier()
		r.NF.O2D[def.NumericOID] = def.Name
		for _, name := range def.Name {
			name = strings.ToLower(name)
			r.NF.D2O[name] = def.NumericOID
		}
		class, _, _ := r.OC.Resolve(def.OC)
		if class == "" {
			panic("Unknown NamedObjectClass " + def.OC)
		}
		r.NF.SOC[def.NumericOID] = class
		r.NF.D2O[def.Identifier()] = def.NumericOID
		r.temp.NF[def.NumericOID] = def
	}
}

func (r *Index) loadNF() (err error) {

	r.NF.Must = make(map[string][]string)
	r.NF.May = make(map[string][]string)
	r.NF.Obsolete = make(map[string]struct{})
	r.NF.DS = make(map[string][]string)

	for noid, form := range r.temp.NF {
		if form.Obsolete {
			r.NF.Obsolete[noid] = struct{}{}
		}

		var must []string
		for j := 0; j < len(form.Must); j++ {
			if a, _, _ := r.AT.Resolve(form.Must[j]); a != "" {
				must = append(must, a)
			}
		}
		if len(must) > 0 {
			r.NF.Must[noid] = must
		}

		var may []string
		for j := 0; j < len(form.May); j++ {
			if a, _, _ := r.AT.Resolve(form.May[j]); a != "" {
				may = append(may, a)
			}
		}
		if len(may) > 0 {
			r.NF.May[noid] = may
		}
	}

	for k := range r.NF.O2D {
		var rules []string
		for rid := range r.temp.DS {
			form := r.temp.NF[k]
			if k == form.NumericOID || k == form.Identifier() {
				rules = append(rules, rid)
			}
		}
		if len(rules) > 0 {
			r.NF.DS[k] = rules
		}
	}

	return
}

type NameFormProperties struct {
	O2D      map[string][]string // numeric OID to descriptor(s)
	D2O      map[string]string   // descriptor to numeric OID
	Princ    map[string]string
	Obsolete map[string]struct{}
	DS       map[string][]string // name form (k) is used by structure rule (v)
	SOC      map[string]string   // name form (k) uses which structural class (v)
	Must     map[string][]string
	May      map[string][]string
	SrcIndex map[string]int // integer index in schema.NameForms
}
