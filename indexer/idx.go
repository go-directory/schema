package indexer

import (
	"github.com/go-directory/schema"
)

/*
New returns an instance of [Index] alongside an error following
an attempt to build performance tables using the input instance
of *[schema.SubschemaSubentry].
*/
func New(sch *schema.SubschemaSubentry) (r Index, err error) {
	r.init(sch)

	for _, err = range []error{
		// syntaxes are fundamental, and
		// are already handled via the
		// r.init() method above.
		r.loadMR(),
		r.loadAT(),
		r.loadAT(),
		r.loadOC(),
		r.loadDC(),
		r.loadNF(),
		r.loadDS(),
	} {
		if err != nil {
			break
		}
	}

	// get rid of temp, regardless of
	// outcome, we're finished with it.
	r.temp = nil

	return
}

func (r *Index) init(sch *schema.SubschemaSubentry) {
	r.temp = &temporaryReferences{}
	r.seedLS(sch)
	r.seedMR(sch)
	r.seedAT(sch)
	r.seedMU(sch)
	r.seedOC(sch)
	r.seedDC(sch)
	r.seedNF(sch)
	r.seedDS(sch)
}

type Index struct {
	LS LDAPSyntaxProperties
	MR MatchingRuleProperties
	AT AttributeTypeProperties
	OC ObjectClassProperties
	DC DITContentRuleProperties
	NF NameFormProperties
	DS DITStructureRuleProperties

	temp *temporaryReferences

	version uint64
}

type temporaryReferences struct {
	LS map[string]*schema.LDAPSyntax
	MR map[string]*schema.MatchingRule
	AT map[string]*schema.AttributeType
	MU map[string]*schema.MatchingRuleUse
	OC map[string]*schema.ObjectClass
	DC map[string]*schema.DITContentRule
	NF map[string]*schema.NameForm
	DS map[string]*schema.DITStructureRule
}

func useIDForMissingDescr(k string, v []string) []string {
	if len(v) == 0 || len(v[0]) == 0 {
		// def has no descriptor/description, use numeric OID
		v = []string{k}
	}
	return v
}

func eqFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca := a[i]
		cb := b[i]

		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}

		if ca != cb {
			return false
		}
	}
	return true
}
