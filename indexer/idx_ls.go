package indexer

import (
	"strings"

	"github.com/go-directory/schema"
)

func (r LDAPSyntaxProperties) Resolve(def string) (noid, desc string, _ []string) {
	if len(def) == 0 {
		return
	}

	def = strings.ToLower(def)
	if f := rune(def[0]); '0' <= f && f <= '2' {
		// def is a numeric OID
		desc, _ = r.O2D[def]
		noid, _ = r.Princ[desc]
	} else if 'a' <= f && f <= 'z' {
		// def is description text
		noid, _ = r.D2O[def]
		desc, _ = r.O2D[noid]
	}

	return
}

/*
NotHumanReadable returns a Boolean value indicative of the input argument def
representing an LDAPSyntax definition whose values are not human readable.
*/
func (r LDAPSyntaxProperties) NotHumanReadable(def string) (not bool) {
	noid, _, _ := r.Resolve(def)
	_, not = r.NotHR[noid]
	return
}

/*
Index returns an integer value following an attempt to call an LDAPSyntax
(def) original index number within the source *[schema.LDAPSyntaxes]
instance.
*/
func (r LDAPSyntaxProperties) Index(def string) (idx int) {
	noid, _, _ := r.Resolve(def)
	var ok bool
	if idx, ok = r.SrcIndex[noid]; !ok {
		idx = -1
	}

	return
}

func (r *Index) seedLS(sch *schema.SubschemaSubentry) {
	r.LS = LDAPSyntaxProperties{}
	r.LS.SrcIndex = make(map[string]int)
	r.LS.Princ = make(map[string]string)
	r.LS.O2D = make(map[string]string)
	r.LS.D2O = make(map[string]string)
	r.LS.NotHR = make(map[string]struct{})
	r.LS.MR = make(map[string]string)
	r.LS.AT = make(map[string]string)
	r.temp.LS = make(map[string]*schema.LDAPSyntax)

	for i := 0; i < sch.LDAPSyntaxes.Len(); i++ {
		def := sch.LDAPSyntaxes.Index(i)
		r.LS.SrcIndex[def.NumericOID] = i
		r.LS.O2D[def.NumericOID] = def.Description
		r.LS.D2O[strings.ToLower(def.Description)] = def.NumericOID
		r.LS.Princ[def.Description] = def.NumericOID
		r.temp.LS[def.NumericOID] = def
		if !def.HR() {
			r.LS.NotHR[def.NumericOID] = struct{}{}
		}
	}
}

type LDAPSyntaxProperties struct {
	O2D      map[string]string   // syntax (k) bears descriptive text (v)
	D2O      map[string]string   // nrml. descriptive text (k) is which syntax (v)
	Princ    map[string]string   // Descriptive text (k) is which syntax (v)
	NotHR    map[string]struct{} // non human readable syntax (bool)
	MR       map[string]string   // matching rule (k) uses syntax (v)
	AT       map[string]string   // attribute type (k) uses syntax (v)
	SrcIndex map[string]int      // integer index in schema.LDAPSyntaxes
}
