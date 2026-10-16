package schema

import (
	_ "embed"
	"fmt"
	"os"
	"testing"
)

var exampleSchema *SubschemaSubentry

func TestMain(m *testing.M) {
	if err := setupExampleSchema(); err != nil {
		panic(err)
	}

	/*
		fmt.Printf("MUST: %#v\n", exampleSchema.OC().MUST[`0.9.2342.19200300.100.4.13`])
		fmt.Printf("ALLMUST: %#v\n", exampleSchema.OC().ALLMUST[`0.9.2342.19200300.100.4.13`])

		fmt.Printf("MAY: %#v\n", exampleSchema.OC().MAY[`0.9.2342.19200300.100.4.13`])
		fmt.Printf("ALLMAY: %#v\n", exampleSchema.OC().ALLMAY[`0.9.2342.19200300.100.4.13`])
	*/

	code := m.Run()
	os.Exit(code)
}

func ExampleSubschemaSubentries() {
	// allocate a new instance of SubschemaSubentries
	schs := make(SubschemaSubentries)

	// add your pre-built schema, e.g.: "exampleSchema",
	// to the new SubschemaSubentries slice type. The
	// schema should already bear a DN, e.g.: "cn=schema".
	schs.Add(exampleSchema)

	// declare "cn=schema" as authoritative
	schemaDN := []byte("cn=schema")
	err := schs.SetAuthoritative(schemaDN)
	if err != nil {
		fmt.Println(err)
		return
	}

	// ensure authoritative (source) schema is properly
	// registered as such via a call to the Authoritative
	// method.
	ath := schs.Authoritative()
	if ath == nil {
		fmt.Println("Authoritative schema not found")
		return
	}

	fmt.Printf("Authoritative schema: %q\n", ath.DN)

	// Now we can create our new schema, e.g.: "cn=alt-schema"
	altDN := []byte("cn=alt-schema")

	// initialize new alt schema instance, but do not prime
	// with built-in syntaxes and matching rules, as we'll
	// just import those en masse from the authoritative
	// instance.
	if err = schs.New(altDN, false); err != nil {
		fmt.Println(err)
		return
	}

	// ensure new schema was created by calling it using its DN
	target, found := schs.Get(altDN)
	if !found {
		fmt.Printf("failed to register %q schema", altDN)
		return
	}

	// import ALL ldapSyntaxes from authoritative schema to new schema
	if err = schs.ImportLDAPSyntaxes(altDN); err != nil {
		fmt.Println(err)
		return
	}

	// import ALL matching rules from authoritative schema to new schema
	if err = schs.ImportMatchingRules(altDN); err != nil {
		fmt.Println(err)
		return
	}

	// types to import. One can use descriptors or
	// numeric OIDs safely.
	types := [][]byte{
		[]byte(`distinguishedName`),
		[]byte(`description`),
		[]byte(`name`),
		[]byte(`2.5.4.3`), // 'cn'
		[]byte(`objectClass`),
		[]byte(`2.5.21.9`), // 'structuralObjectClass'
		[]byte(`sn`),
		[]byte(`userPassword`),
		[]byte(`subtreeSpecification`),
		[]byte(`telephoneNumber`),
		[]byte(`seeAlso`),
	}

	// import desired types
	if err = schs.ImportAttributeTypes(altDN, types...); err != nil {
		fmt.Println(err)
		return
	}

	// ensure 'cn' made it over
	noid, _, _, _ := target.AT().Resolve([]byte(`cn`))
	if noid == nil {
		fmt.Println("failed to resolve imported type 'cn' from authoritative schema")
		return
	}

	// classes to import
	classes := [][]byte{
		[]byte(`2.5.6.0`), // 'top'
		[]byte(`person`),
		[]byte(`extensibleObject`),
		[]byte(`subentry`),
	}

	// import desired classes
	if err = schs.ImportObjectClasses(altDN, classes...); err != nil {
		fmt.Println(err)
		return
	}

	// ensure 'top' made it over
	if noid, _, _, _ = target.OC().Resolve([]byte(`top`)); noid == nil {
		fmt.Println("failed to resolve imported class 'top' from authoritative schema")
		return
	}

	// Print some attribute types just for fun
	fmt.Printf("attribute types imported into new schema %q:\n\n", target.DN)
	fmt.Printf("%s\n", target.AT().String())

	// Output:
	// Authoritative schema: "cn=schema"
	// attribute types imported into new schema "cn=alt-schema":
	//
	// attributeTypes ( 2.5.4.49 NAME 'distinguishedName' EQUALITY distinguishedNameMatch SYNTAX 1.3.6.1.4.1.1466.115.121.1.12 )
	// attributeTypes ( 2.5.4.13 NAME 'description' EQUALITY caseIgnoreMatch SUBSTR caseIgnoreSubstringsMatch SYNTAX 1.3.6.1.4.1.1466.115.121.1.15 )
	// attributeTypes ( 2.5.4.41 NAME 'name' EQUALITY caseIgnoreMatch SUBSTR caseIgnoreSubstringsMatch SYNTAX 1.3.6.1.4.1.1466.115.121.1.15 )
	// attributeTypes ( 2.5.4.3 NAME ( 'cn' 'commonName' ) DESC 'RFC4519: common name(s) for which the entity is known by' SUP name )
	// attributeTypes ( 2.5.4.0 NAME 'objectClass' EQUALITY objectIdentifierMatch SYNTAX 1.3.6.1.4.1.1466.115.121.1.38 )
	// attributeTypes ( 2.5.21.9 NAME 'structuralObjectClass' EQUALITY objectIdentifierMatch SYNTAX 1.3.6.1.4.1.1466.115.121.1.38 SINGLE-VALUE NO-USER-MODIFICATION )
	// attributeTypes ( 2.5.4.4 NAME 'sn' SUP name )
	// attributeTypes ( 2.5.4.35 NAME 'userPassword' EQUALITY octetStringMatch SYNTAX 1.3.6.1.4.1.1466.115.121.1.40 )
	// attributeTypes ( 2.5.18.6 NAME 'subtreeSpecification' SYNTAX 1.3.6.1.4.1.1466.115.121.1.45 SINGLE-VALUE X-ORIGIN 'RFC3672' )
	// attributeTypes ( 2.5.4.20 NAME 'telephoneNumber' EQUALITY telephoneNumberMatch SUBSTR telephoneNumberSubstringsMatch SYNTAX 1.3.6.1.4.1.1466.115.121.1.50 )
	// attributeTypes ( 2.5.4.34 NAME 'seeAlso' SUP distinguishedName )
}

/*
func exampleDump(t *testing.T) {
	t.Logf("LS: %#v", exampleSchema.LS())
	t.Logf("MR: %#v", exampleSchema.MR())
	t.Logf("AT: %#v", exampleSchema.AT())
	t.Logf("MU: %#v", exampleSchema.MU())
	t.Logf("DC: %#v\n\n", exampleSchema.DC())
	t.Logf("OC: %#v\n\n", exampleSchema.OC())
	t.Logf("NF: %#v\n\n", exampleSchema.NF())
	t.Logf("DS: %#v", exampleSchema.DS())
}
*/

//go:embed testdata/example.schema
var exampleSchemaFile []byte

func setupExampleSchema() (err error) {
	if exampleSchema, err = New(true); err == nil {
		exampleSchema.DN = []byte("cn=schema")
		err = exampleSchema.ReadBytes(exampleSchemaFile)
	}
	return
}
