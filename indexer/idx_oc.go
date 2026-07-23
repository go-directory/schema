package indexer

import (
	"strings"

	"github.com/go-directory/schema"
)

func (r ObjectClassProperties) Resolve(def string) (noid, ident string, names []string) {
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
Index returns an integer value following an attempt to call an ObjectClass
(def) original index number within the source *[schema.ObjectClasses]
instance.
*/
func (r ObjectClassProperties) Index(def string) (idx int) {
	noid, _, _ := r.Resolve(def)
	var ok bool
	if idx, ok = r.SrcIndex[noid]; !ok {
		idx = -1
	}

	return
}

func (r *Index) seedOC(sch *schema.SubschemaSubentry) {
	r.OC = ObjectClassProperties{}
	r.OC.SrcIndex = make(map[string]int)
	r.OC.Princ = make(map[string]string)
	r.OC.O2D = make(map[string][]string)
	r.OC.D2O = make(map[string]string)
	r.temp.OC = make(map[string]*schema.ObjectClass)

	for i := 0; i < sch.ObjectClasses.Len(); i++ {
		def := sch.ObjectClasses.Index(i)
		r.OC.SrcIndex[def.NumericOID] = i
		r.OC.Princ[def.NumericOID] = def.Identifier()
		r.OC.O2D[def.NumericOID] = def.Name
		for _, name := range def.Name {
			name = strings.ToLower(name)
			r.OC.D2O[name] = def.NumericOID
		}
		r.temp.OC[def.NumericOID] = def
	}
}

func (r *Index) loadOC() (err error) {
	r.OC.Kind = make(map[string]uint8)
	r.OC.Must = make(map[string][]string)
	r.OC.May = make(map[string][]string)
	r.OC.AllMust = make(map[string][]string)
	r.OC.AllMay = make(map[string][]string)
	r.OC.Sup = make(map[string][]string)
	r.OC.Sub = make(map[string][]string)
	r.OC.Obsolete = make(map[string]struct{})

	for noid, class := range r.temp.OC {
		if class.Obsolete {
			r.OC.Obsolete[noid] = struct{}{}
		}
		r.OC.Kind[noid] = class.Kind

		var must, allmust []string
		for j := 0; j < len(class.Must); j++ {
			if a, _, _ := r.AT.Resolve(class.Must[j]); a != "" {
				must = append(must, a)
			}
		}
		if len(must) > 0 {
			r.OC.Must[noid] = must
		}

		alms := class.AllMust()
		for j := 0; j < alms.Len(); j++ {
			allmust = append(allmust, alms.Index(j).NumericOID)
		}
		if len(allmust) > 0 {
			r.OC.AllMust[noid] = allmust
		}

		var may, allmay []string
		for j := 0; j < len(class.May); j++ {
			if a, _, _ := r.AT.Resolve(class.May[j]); a != "" {
				may = append(may, a)
			}
		}
		if len(may) > 0 {
			r.OC.May[noid] = may
		}

		alms = class.AllMay()
		for j := 0; j < alms.Len(); j++ {
			allmay = append(allmay, alms.Index(j).NumericOID)
		}
		if len(allmay) > 0 {
			r.OC.AllMay[noid] = allmay
		}

		var supers []string
		for j := 0; j < len(class.SuperClasses); j++ {
			if o, _, _ := r.OC.Resolve(class.SuperClasses[j]); o != "" {
				supers = append(supers, o)
			}
		}
		if len(supers) > 0 {
			r.OC.Sup[noid] = supers
		}
	}

	r.subClasses()

	return
}

func (r *Index) subClasses() {
	for _, v := range r.OC.Sup {
		for _, x := range v {
			var subs []string
			for noid, class := range r.temp.OC {
				for _, s := range class.SuperClasses {
					if s == x {
						subs = append(subs, noid)
					}
				}
			}

			if len(subs) > 0 {
				r.OC.Sub[x] = subs
			}
		}
	}
}

type ObjectClassProperties struct {
	O2D      map[string][]string // numeric OID to descriptor(s)
	D2O      map[string]string   // descriptor to numeric OID
	Princ    map[string]string
	Kind     map[string]uint8
	Obsolete map[string]struct{}
	Sup      map[string][]string
	Sub      map[string][]string
	Must     map[string][]string
	AllMust  map[string][]string
	May      map[string][]string
	AllMay   map[string][]string
	SrcIndex map[string]int // integer index in schema.ObjectClasses
}
