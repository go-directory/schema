package schema

/*
schema.go implements the subschemaSubentry type and related
methods.
*/

import (
	"bytes"
	_ "embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var now = time.Now

/*
ErrorOnDuplicateClauseMember, when true, will cause any
definition registration process to fail if a duplicate
clause member -- that is, a descriptor or numeric OID
residing within 'MUST', 'SUP' or any other multi-valued
clause -- is encountered at any point.

The default, false, is to silently discard duplicates.

This variable can be changed at any point.
*/
var ErrorOnDuplicateClauseMember = false

/*
ErrorOnUnavailableNameFormAttribute, when true, will error
on the presence of a MUST or MAY clause member attribute
type that was not extended by the object class (or possibly
its super classes, if any exist) implemented by the name
form.

The default, false, results in no error being returned if
this condition is detected during name form parsing.
*/
var ErrorOnUnavailableNameFormAttribute = false

/*
NameFormDirectAttributesOnly, when true, will only consider
attribute types valid for use in a name form's MUST or MAY
clause if they are *directly* extended by the implemented
object class's MUST and MAY clauses, and NOT by way of super
class type inheritance.

The default, false, will result in the inclusion of types
extended by any and all super class MAY and MUST clauses in
addition to those directly extended by the name form's
implemented class.

Alternatively, see the [ErrorOnUnavailableNameFormAttribute]
option to suppress errors relating to the presence of any
unavailable attribute type within the name form's MUST or MAY
clauses *for any reason*.
*/
var NameFormDirectAttributesOnly = false

//go:embed synmr/ls.schema
var lsPrimer []byte

//go:embed synmr/mr.schema
var mrPrimer []byte

/*
New returns a freshly initialized instance of *[SubschemaSubentry]. Instances of
this type serve as a platform upon which individual text definitions may be
parsed into usable instances of [Definition].

Instances of *[SubschemaSubentry] that were NOT created as a result of using
this package level function will almost certainly panic when used.

The prime variadic argument controls whether to prime, or "pre-load", standard
[LDAPSyntax] and [MatchingRule] definitions sourced from RFC 4512, RFC 4523 and
RFC 2307 into the receiver instance. The default is false, which results in no
such definitions being pre-loaded.

Generally speaking, it is RECOMMENDED that users pre-load these definitions UNLESS
they are constructing a very stringent schema structure which only contains select
syntaxes and matching rules -- a most unusual circumstance. In such a case, users
will be required to register those select syntaxes and matching rules MANUALLY.
*/
func New(prime ...bool) (sch *SubschemaSubentry, err error) {
	sch = &SubschemaSubentry{
		lock: &sync.Mutex{},
	}

	sch.ls = initLDAPSyntaxProperties(sch)
	sch.mr = initMatchingRuleProperties(sch)
	sch.at = initAttributeTypeProperties(sch)
	sch.mu = initMatchingRuleUseProperties(sch)
	sch.oc = initObjectClassProperties(sch)
	sch.dc = initDITContentRuleProperties(sch)
	sch.nf = initNameFormProperties(sch)
	sch.ds = initDITStructureRuleProperties(sch)

	if len(prime) > 0 && prime[0] {
		err = sch.primeBuiltIns()
	}

	return
}

/*
LS returns the underlying instance of [LDAPSyntaxProperties].
The return value is pointer-dereferenced and should not be
modified in any way.
*/
func (r SubschemaSubentry) LS() LDAPSyntaxProperties {
	return (*r.ls)
}

/*
MR returns the underlying instance of [MatchingRuleProperties].
The return value is pointer-dereferenced and should not be
modified in any way.
*/
func (r SubschemaSubentry) MR() MatchingRuleProperties {
	return (*r.mr)
}

/*
AT returns the underlying instance of [AttributeTypeProperties].
The return value is pointer-dereferenced and should not be
modified in any way.
*/
func (r SubschemaSubentry) AT() AttributeTypeProperties {
	return (*r.at)
}

/*
MU returns the underlying instance of [MatchingRuleUseProperties].
The return value is pointer-dereferenced and should not be
modified in any way.
*/
func (r SubschemaSubentry) MU() MatchingRuleUseProperties {
	return (*r.mu)
}

/*
OC returns the underlying instance of [ObjectClassProperties].
The return value is pointer-dereferenced and should not be
modified in any way.
*/
func (r SubschemaSubentry) OC() ObjectClassProperties {
	return (*r.oc)
}

/*
DC returns the underlying instance of [DITContentRuleProperties].
The return value is pointer-dereferenced and should not be
modified in any way.
*/
func (r SubschemaSubentry) DC() DITContentRuleProperties {
	return (*r.dc)
}

/*
NF returns the underlying instance of [NameFormProperties].
The return value is pointer-dereferenced and should not be
modified in any way.
*/
func (r SubschemaSubentry) NF() NameFormProperties {
	return (*r.nf)
}

/*
DS returns the underlying instance of [DITStructureRuleProperties].
The return value is pointer-dereferenced and should not be
modified in any way.
*/
func (r SubschemaSubentry) DS() DITStructureRuleProperties {
	return (*r.ds)
}

/*
ReadDirectory recurses all files and folders specified at 'dir',
returning parsed schema bytes (content) alongside an error.

Only files with an extension of ".schema" will be parsed, but all
subdirectories will be traversed in search of these files. Files
not bearing the ".schema" extension will be silently ignored.

File and directory naming schemes MUST guarantee the appropriate
ordering of any and all sub types, sub rules and sub classes which
would rely on the presence of dependency definitions (e.g.: 'cn'
cannot exist without 'name').
*/
func (r *SubschemaSubentry) ReadDirectory(dir string) (err error) {

	// remove any number of trailing
	// slashes from dir.
	dir = strings.TrimRight(dir, `/`)

	// avoid panicking if the directory does not exist during
	// the "walking" process.
	if _, err = os.Stat(dir); !errorIs(err, errNotExist) {
		// recurse dir path
		err = filepath.Walk(dir, func(p string, d fs.FileInfo, err error) error {
			if !d.IsDir() && strings.HasSuffix(d.Name(), ".schema") {
				err = r.ReadFile(p)
			}

			return err
		})
	}

	return
}

/*
ReadFile returns an error following an attempt to read the
specified filename into an instance of []byte, which is then
fed to the [SubschemaSubentry.ReadBytes] method automatically.

The filename MUST end in ".schema", else an error shall be raised.
*/
func (r *SubschemaSubentry) ReadFile(file string) (err error) {
	if !strings.HasSuffix(file, `.schema`) {
		err = schemaError("Filename MUST end in `.schema`")
		return
	}

	var data []byte
	if data, err = os.ReadFile(file); err == nil {
		err = r.ReadBytes(data)
	}

	return
}

/*
ReadBytes returns an error following an attempt parse data ([]byte)
into the receiver instance. This method exists as a convenient
alternative to manual parsing of individual definitions, one at a
time.

Definitions which are dependencies of other definitions should be
parsed first. For example, the following AttributeTypeDescriptions
should be parsed in the order shown:

attributeType ( 2.5.4.41 NAME 'name' EQUALITY caseIgnoreMatch SUBSTR caseIgnoreSubstringsMatch SYNTAX 1.3.6.1.4.1.1466.115.121.1.15 )

attributeType ( 2.5.4.3 NAME 'cn' SUP name )

... as "cn" depends upon "name".

Each definition MUST begin with one (1) of the following keywords
followed by any amount of whitespace:

  - "ldapSyntax" or "ldapSyntaxes"
  - "matchingRule" or "matchingRules"
  - "attributeType" or "attributeTypes"
  - "objectClass" or "objectClasses"
  - "dITContentRule" or "dITContentRules"
  - "nameForm" or "nameForms"
  - "dITStructureRule" or "dITStructureRules"

Case is not significant in the keyword matching process.
*/
func (r *SubschemaSubentry) ReadBytes(data []byte) error {
	data = removeBashComments(data)
	data = bytes.ReplaceAll(data, []byte("$\n"), []byte("$ "))
	lines := bytes.Split(data, []byte("\n"))

	var (
		result [][]byte
		cur    []byte
	)

	// Returns a boolean and the keyword IF said
	// keyword is at the start of the line.
	isKeywordLine := func(line []byte) (bool, string) {
		line = lc(line)
		for _, keyword := range headerTokens {
			if bytes.HasPrefix(line, lc([]byte(keyword))) {
				return true, keyword
			}
		}
		return false, ""
	}

	for _, line := range lines {
		line = []byte(condenseWHSP(line))
		if len(line) == 0 {
			continue
		}

		if isKeyword, _ := isKeywordLine(line); isKeyword {
			if len(cur) > 0 {
				result = append(result, cur)
			}
			cur = line
		} else {
			if len(cur) > 0 {
				cur = append(cur, ' ')
				cur = append(cur, line...)
			}
		}
	}

	// Add the final segment
	if len(cur) > 0 {
		result = append(result, cur)
	}

	return r.registerDefinitionByCase(result)
}

func (r *SubschemaSubentry) registerDefinitionByCase(defs [][]byte) (err error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	for i := 0; i < len(defs); i++ {
		def := defs[i]
		low := lc(def)
		if bytes.HasPrefix(low, lc([]byte(headerTokens[1]))) {
			err = r.ls.marshal(def, false)
		} else if bytes.HasPrefix(low, lc([]byte(headerTokens[3]))) &&
			!bytes.HasPrefix(low, lc([]byte(headerTokens[7]))) {
			err = r.mr.marshal(def, false)
		} else if bytes.HasPrefix(low, lc([]byte(headerTokens[5]))) {
			err = r.at.marshal(def, false)
		} else if bytes.HasPrefix(low, lc([]byte(headerTokens[9]))) {
			err = r.oc.marshal(def, false)
		} else if bytes.HasPrefix(low, lc([]byte(headerTokens[11]))) {
			err = r.dc.marshal(def, false)
		} else if bytes.HasPrefix(low, lc([]byte(headerTokens[13]))) {
			err = r.nf.marshal(def, false)
		} else if bytes.HasPrefix(low, lc([]byte(headerTokens[15]))) {
			err = r.ds.marshal(def, false)
		} else {
			err = schemaError("Invalid definition: " + string(def))
		}

		if err != nil {
			break
		}
		r.Timestamp = now()
	}

	return
}

/*
SubschemaSubentries implements a map of string indices to *[SubschemaSubentry]
instances.

While not derived as is from any standard, this type does facilitate some of
the components needed for advanced schema management with respect to the use
of administrativeRole-controlled subtrees.

This type also serves to facilitate administrative ease with respect to the
creation or migration of a complete schema using definitions derived from other
sources.

Each map index string label should be the string representation of the schema
schema distinguished name. Each map index value (*[SubschemaSubentry]) MUST bear
an identical distinguished name via the "DN" []byte field.

The "authoritative" *[SubschemaSubentry] instance SHOULD bear the standard naming
context of "cn=schema". For implementations managing only a single *[SubschemaSubentry]
instance, the same naming recommendation stands.

All distinguished names are normalized to lower case.

It is an error for any of the following conditions to evaluate as true:

  - Any *[SubschemaSubentry] map value bears a distinguished name that differs from its map index
  - Any *[SubschemaSubentry] map value lacks a distinguished name, or its distinguished name is malformed
  - Any *[SubschemaSubentry] map value bears a distinguished name identical to that of another instance in the same map
  - None of the *[SubschemaSubentry] map values bears an "Authoritative" field value of true
  - More than one of the *[SubschemaSubentry] map values bears an "Authoritative" field value of true

NOTE: This type is still largely conceptual and lacks most of the intended functionality
at this time.
*/
type SubschemaSubentries map[string]*SubschemaSubentry

/*
Authoritative returns the authoritative instance of *[SubschemaSubentry], or nil if not found.
*/
func (r SubschemaSubentries) Authoritative() *SubschemaSubentry {
	var ath *SubschemaSubentry
	for _, v := range r {
		if v.Authoritative {
			ath = v
			break
		}
	}

	return ath
}

/*
Get returns an instance of *[SubschemaSubentry] alongside a presence-indicative
Boolean value following an attempt to match the input DN to a registered instance.

Case is not significant in the matching process.
*/
func (r SubschemaSubentries) Get(dn []byte) (*SubschemaSubentry, bool) {
	sch, found := r[string(lc(dn))]
	return sch, found
}

/*
Add returns an error following an attempt to add the input *[SubschemaSubentry] to
the receiver instance.

The error will be non-zero if any of the following conditions evaluate as true:

  - Input *[SubschemaSubentry] is bogus or nil
  - Input *[SubschemaSubentry] lacks a distinguished name
*/
func (r *SubschemaSubentries) Add(sch *SubschemaSubentry) error {
	var err error
	if sch == nil || sch.ls == nil {
		err = schemaError("Unable to add new schema: receiver is nil")
		return err
	}

	dn := sch.DN
	name := string(dn)

	if len(dn) == 0 {
		err = schemaError("Unable to add new schema: DN is zero length or nil")
		return err
	}

	if _, found := r.Get(dn); found {
		err = schemaError("Unable to add new schema: DN '", name, "' already registered")
	}

	isAth := sch.Authoritative
	ath := r.Authoritative()

	if isAth && ath != nil {
		err = schemaError("Unable to add new schema: cannot add new authoritative schema (",
			name, ")  without demoting the existing one (", string(ath.DN), ")")
		return err
	}

	(*r)[name] = sch

	return err
}

/*
SetAuthoritative returns an error following an attempt to modify the specified
*[SubschemaSubentry]'s "Authoritative" field.

The error will be non-zero if any of the following conditions evaluate as true:

  - Input DN is zero length or nil
  - No match was found for the specified DN
  - Another preexisting *[SubschemaSubentry] is already authoritative

The optional variadic bool allows the override of the default value state of true.
In other words, the user may supply a boolean of false to *unset* authoritative
status if another *[SubschemaSubentry] instance is slated for that role.
*/
func (r *SubschemaSubentries) SetAuthoritative(dn []byte, authoritative ...bool) error {
	var err error
	var name = string(lc(dn))

	var b bool = true // assume true for the moment
	if len(authoritative) > 0 {
		b = authoritative[0]
	}

	if ath := r.Authoritative(); ath != nil {
		if eq := bytes.EqualFold(ath.DN, dn); eq {
			ath.Authoritative = b
			return err
		}

		err = schemaError("Authoritative schema '", string(ath.DN),
			"' already in force; cannot promote '", name, "'")
		return err
	}

	sch, found := (*r)[name]
	if !found || sch == nil || len(sch.DN) == 0 {
		err = schemaError("Schema '", name, "' not registered")
		return err
	}
	sch.Authoritative = b

	return err
}

/*
New returns an error following an attempt to initialize a new instance of
*[SubschemaSubentry] in association with the specified dn map index.
*/
func (r *SubschemaSubentries) New(dn []byte, prime ...bool) error {
	dn = lc(dn)

	var err error
	var name = string(dn)

	if _, found := (*r)[name]; found {
		err = schemaError("Schema '", name, "' already registered")
		return err
	}

	var sch *SubschemaSubentry
	if sch, err = New(len(prime) > 0 && prime[0]); err == nil {
		sch.DN = dn
		(*r)[name] = sch
	}

	return err
}

/*
ImportLDAPSyntaxes returns an error following an attempt to import some, or all, LDAP
syntaxes from the underlying authoritative schema and into the registered schema
associated with the input distinguished name (dest).

The variadic def input argument can be used to specify any number of desired definitions
by numeric OID. If none are provided, all are imported en masse.
*/
func (r *SubschemaSubentries) ImportLDAPSyntaxes(dest []byte, def ...[]byte) error {
	return r.importDefs(0, dest, def...)
}

/*
ImportMatchingRules returns an error following an attempt to import some, or all, matching
rules from the underlying authoritative schema and into the registered schema associated
with the input distinguished name (dest).

The variadic def input argument can be used to specify any number of desired definitions
by numeric OID or descriptor. If none are provided, all are imported en masse.
*/
func (r *SubschemaSubentries) ImportMatchingRules(dest []byte, def ...[]byte) error {
	return r.importDefs(1, dest, def...)
}

/*
ImportAttributeTypes returns an error following an attempt to import some, or all, attribute
types from the underlying authoritative schema and into the registered schema associated with
the input distinguished name (dest).

The variadic def input argument can be used to specify any number of desired definitions
by numeric OID or descriptor. If none are provided, all are imported en masse.

Successful use of this method results in the targeted schema's matchingRuleUse tables being
updated accordingly, wherever applicable.
*/
func (r *SubschemaSubentries) ImportAttributeTypes(dest []byte, def ...[]byte) error {
	return r.importDefs(2, dest, def...)
}

/*
ImportObjectClasses returns an error following an attempt to import some, or all, object
classes from the underlying authoritative schema and into the registered schema associated
with the input distinguished name (dest).

The variadic def input argument can be used to specify any number of desired definitions
by numeric OID or descriptor. If none are provided, all are imported en masse.
*/
func (r *SubschemaSubentries) ImportObjectClasses(dest []byte, def ...[]byte) error {
	return r.importDefs(4, dest, def...)
}

/*
ImportDITContentRules returns an error following an attempt to import some, or all, content
rules from the underlying authoritative schema and into the registered schema associated with
the input distinguished name (dest).

The variadic def input argument can be used to specify any number of desired definitions
by numeric OID or descriptor. If none are provided, all are imported en masse.
*/
func (r *SubschemaSubentries) ImportDITContentRules(dest []byte, def ...[]byte) error {
	return r.importDefs(5, dest, def...)
}

/*
ImportNameForms returns an error following an attempt to import some, or all, name forms
from the underlying authoritative schema and into the registered schema associated with the
input distinguished name (dest).

The variadic def input argument can be used to specify any number of desired definitions
by numeric OID or descriptor. If none are provided, all are imported en masse.
*/
func (r *SubschemaSubentries) ImportNameForms(dest []byte, def ...[]byte) error {
	return r.importDefs(6, dest, def...)
}

/*
ImportDITStructureRules returns an error following an attempt to import some, or all, structure
rules from the underlying authoritative schema and into the registered schema associated with the
input distinguished name (dest).

The variadic def input argument can be used to specify any number of desired definitions
by numeric OID or descriptor. If none are provided, all are imported en masse.
*/
func (r *SubschemaSubentries) ImportDITStructureRules(dest []byte, def ...[]byte) error {
	return r.importDefs(7, dest, def...)
}

/*
importDefs returns an error following an attempt to import the specified definition into
the destination *[SubschemaSubentry].

The return error will be non-nil if any of the following conditions evaluate as true:

  - The category is bogus
  - The destination distinguished name is not associated with a registered *[SubschemaSubentry]
  - The destination distinguished name is the same as the authoritative *[SubschemaSubentry]
  - Imported definition does not exist, or cannot be resolved
  - Imported definition depends on one or more other definitions not yet imported
  - No authoritative *[SubschemaSubentry] is registered

Valid categories are as follows:

  - (0) LDAP Syntax
  - (1) Matching Rule
  - (2) Attribute Type
  - (4) Object Class
  - (5) DIT Content Rule
  - (6) Name Form
  - (7) DIT Structure Rule

Note that (3) -- Matching Rule Use -- is unlisted, as this category is self-managing
based on the present Matching Rules and Attribute Types, and not something that should
be "imported" in the traditional sense.

If the variadic def is unused, all definitions of the specified category will be imported.
Otherwise, each def []byte slice should contain the numeric OID, rule ID or descriptor of
the desired definition.
*/
func (r *SubschemaSubentries) importDefs(cat int, dest []byte, def ...[]byte) error {
	var err error

	if cat == 3 {
		err = schemaError("cannot import 'matchingRuleUse'")
		return err
	} else if !(0 <= cat && cat <= 7) {
		err = schemaError("bad import category '", itoa(cat), "'")
		return err
	}

	ath := r.Authoritative()
	if ath == nil {
		err = schemaError("no authoritative subschemaSubentry in force")
		return err
	} else if bytes.EqualFold(dest, ath.DN) {
		err = schemaError("illogical import: authoritative subschemaSubentry cannot import from itself")
		return err
	}

	dn := lc(dest)
	name := string(dn)

	target, found := (*r)[name]
	if !found {
		err = schemaError("target subschemaSubentry '", name, "' not found")
		return err
	}

	err = doImport(cat, ath, target, def...)
	return err
}

func doImport(cat int, ath, target *SubschemaSubentry, def ...[]byte) (err error) {
	// replace def with ALL definitions
	getAll := func(ord []uint, source map[uint][]byte) [][]byte {
		var defs [][]byte
		for i := 0; i < len(ord); i++ {
			defs = append(defs, source[ord[i]])
		}
		return defs
	}

	all := len(def) == 0
	imp := func(
		typ string,
		ord []uint,
		ids map[uint][]byte,
		strs map[uint]string,
		resolver func([]byte) ([]byte, []byte, [][]byte, uint),
		marshaler func([]byte, bool) error,
	) (err error) {
		if all {
			def = getAll(ord, ids)
		}

		tk := append([]byte(typ), []byte(` `)...)
		for i := 0; i < len(def); i++ {
			noid, _, _, n := resolver(def[i])
			if noid == nil {
				err = schemaError("target definition '", string(def[i]), "' (",
					typ, ") not found in authoritative subschemaSubentry")
				break
			}

			b := append(tk, []byte(strs[n])...)
			if err = marshaler(b, false); err != nil {
				err = schemaError(string(tk), "import failed for '", string(noid),
					"': ", err.Error())
				break
			}
		}

		return
	}

	switch cat {
	case 0:
		// ldap syntax resolver is different from all the others,
		// so we wrap it specially here.
		res := func(in []byte) ([]byte, []byte, [][]byte, uint) {
			noid, _, n := ath.ls.Resolve(in)
			return noid, nil, nil, n
		}
		src := ath.LS()
		err = imp(headerTokens[0], src.ORD, src.ID, src.STRING, res, target.ls.marshal)
	case 1:
		src := ath.MR()
		err = imp(headerTokens[2], src.ORD, src.ID, src.STRING, src.Resolve, target.mr.marshal)
	case 2:
		src := ath.AT()
		err = imp(headerTokens[4], src.ORD, src.ID, src.STRING, src.Resolve, target.at.marshal)
	case 4:
		src := ath.OC()
		err = imp(headerTokens[8], src.ORD, src.ID, src.STRING, src.Resolve, target.oc.marshal)
	case 5:
		src := ath.DC()
		err = imp(headerTokens[10], src.ORD, src.ID, src.STRING, src.Resolve, target.dc.marshal)
	case 6:
		src := ath.NF()
		err = imp(headerTokens[10], src.ORD, src.ID, src.STRING, src.Resolve, target.nf.marshal)
	case 7:
		src := ath.DS()
		err = imp(headerTokens[14], src.ORD, src.ID, src.STRING, src.Resolve, target.ds.marshal)
	}

	return err
}

/*
SubschemaSubentry implements [§ 4.2 of RFC 4512] through a collection of fast lookup
tables (maps).

Instances of this type are thread safe by way of an internal instance of [sync/Mutex].
No special actions are required by users to make use of this feature, and its invocation
is automatic wherever appropriate.

[§ 4.2 of RFC 4512]: https://datatracker.ietf.org/doc/html/rfc4512#section-4.2
*/
type SubschemaSubentry struct {
	DN            []byte    // schema context DN
	Authoritative bool      // Set to true if schema is authoritative (or solitary)
	Unadvertised  bool      // indicates schema is not to be published indiscriminately
	Timestamp     time.Time // schemaTimestamp
	lock          *sync.Mutex

	ls *LDAPSyntaxProperties
	mr *MatchingRuleProperties
	at *AttributeTypeProperties
	mu *MatchingRuleUseProperties
	oc *ObjectClassProperties
	dc *DITContentRuleProperties
	nf *NameFormProperties
	ds *DITStructureRuleProperties
}

/*
primeBuiltIns is a private method used to pre-load standard LDAPSyntax
and MatchingRule instances sourced from formalized RFCs.
*/
func (r *SubschemaSubentry) primeBuiltIns() (err error) {
	if err = r.ReadBytes(lsPrimer); err == nil {
		err = r.ReadBytes(mrPrimer)
	}

	return
}

/*
String returns the string representation of the receiver instance. Note
that definitions bearing the X-DO-NOT-ADVERTISE extension with a value
of 'TRUE' will NOT be included in the output.
*/
func (r SubschemaSubentry) String() string {
	bld := strings.Builder{}

	bld.WriteString(r.ls.String())
	bld.WriteString(r.mr.String())
	bld.WriteString(r.at.String())
	bld.WriteString(r.mu.String())
	bld.WriteString(r.oc.String())
	bld.WriteString(r.dc.String())
	bld.WriteString(r.nf.String())
	bld.WriteString(r.ds.String())

	return bld.String()
}

/*
OID returns the numeric OID literal "2.5.18.10" per [§ 4.2 of RFC 4512].

[§ 4.2 of RFC 4512]: https://datatracker.ietf.org/doc/html/rfc4512#section-4.2
*/
func (r SubschemaSubentry) OID() string { return `2.5.18.10` }

// Keep plurals before singulars for optimal matching. Note that
// the respective indices correlate to the return values of the
// Type method held by collection and definition description types.
var headerTokens []string = []string{
	"ldapSyntaxes", "ldapSyntax", // 0, 1
	"matchingRules", "matchingRule", // 2, 3
	"attributeTypes", "attributeType", // 4, 5
	"matchingRuleUses", "matchingRuleUse", // 6, 7
	"objectClasses", "objectClass", // 8, 9
	"dITContentRules", "dITContentRule", // 10, 11
	"nameForms", "nameForm", // 12, 13
	"dITStructureRules", "dITStructureRule", // 14, 15
}
