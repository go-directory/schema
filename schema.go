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
)

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

	for i := 0; i < len(defs) && err == nil; i++ {
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
	}

	return
}

/*
SubschemaSubentry implements [§ 4.2 of RFC 4512] and contains slice types
of various [Definition] types.

Instances of this type are thread safe by way of an internal instance
of [sync/Mutex]. No special actions are required by users to make use
of this feature, and its invocation is automatic wherever appropriate.

[§ 4.2 of RFC 4512]: https://datatracker.ietf.org/doc/html/rfc4512#section-4.2
*/
type SubschemaSubentry struct {
	DN      []byte // schema context DN
	Private bool   // indicates schema is not to be published indiscriminately
	lock    *sync.Mutex
	ath     *SubschemaSubentry // if rcv.ath not nil, rcv is a VIEW schema

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
	"ldapSyntaxes", "ldapSyntax",
	"matchingRules", "matchingRule",
	"attributeTypes", "attributeType",
	"matchingRuleUses", "matchingRuleUse",
	"objectClasses", "objectClass",
	"dITContentRules", "dITContentRule",
	"nameForms", "nameForm",
	"dITStructureRules", "dITStructureRule",
}
