package main

/*
schema-check.go offers a fast and easy schema
file or directory parsing utility.

See extended help (-e) for details.

It is intended for verification purposes only.

Note that this utility expects a complete schema,
an will error if dependency definitions are not
found.
*/

import (
	"flag"
	"fmt"
	"os"

	"github.com/go-directory/schema"
)

func main() {
	var (
		err   error
		sch   *schema.SubschemaSubentry
		fFlag = flag.String("f", "", "Path of schema file or directory")
		dFlag = flag.Bool("d", false, "Print schema to stdout")
		sFlag = flag.Bool("s", false, "Show number of parsed definitions per category")
		pFlag = flag.Bool("p", false, "Prime the schema with syntaxes and matching rules")
		eFlag = flag.Bool("e", false, "Show extended usage information and then exit")
	)

	flag.Parse()

	showStats := *sFlag
	dump := *dFlag
	prime := *pFlag
	target := *fFlag
	exhelp := *eFlag

	if exhelp {
		extendedUsage()
		os.Exit(1)
	}

	// No path, no parse.
	if target == "" {
		err = fmt.Errorf("Must specify file or directory path using '-f'")
	}
	chkerr(err)

	// Initialize a new subschemaSubentry instance.
	sch, err = schema.New(prime)
	chkerr(err)

	// TODO: support special flags that map to variable settings
	// which control the "strictness" of the parsing routine.

	// File, directory or bail.
	x, err := os.Stat(target)
	if x.Mode().IsRegular() {
		err = sch.ReadFile(target)
	} else if x.Mode().IsDir() {
		err = sch.ReadDirectory(target)
	} else if err == nil {
		err = fmt.Errorf("Target path %q is neither a file nor a directory", target)
	}
	chkerr(err)

	if showStats {
		schemaStats(sch)
	} else if dump {
		dumpSchema(sch)
	}

	// Cool schema, bro 8)
}

func dumpSchema(sch *schema.SubschemaSubentry) {
	fmt.Printf("%s\n", sch.String())
}

func schemaStats(sch *schema.SubschemaSubentry) {
	ls := len(sch.LS().ID)
	mr := len(sch.MR().ID)
	at := len(sch.AT().ID)
	oc := len(sch.OC().ID)
	dc := len(sch.DC().ID)
	nf := len(sch.NF().ID)
	ds := len(sch.DS().ID)

	fmt.Printf("Definitions parsed:\n")
	fmt.Printf("  - LDAP syntaxes:   %d\n", ls)
	fmt.Printf("  - matching rules:  %d\n", mr)
	fmt.Printf("  - attribute types: %d\n", at)
	fmt.Printf("  - object classes:  %d\n", oc)
	fmt.Printf("  - content rules:   %d\n", dc)
	fmt.Printf("  - name forms:      %d\n", nf)
	fmt.Printf("  - structure rules: %d\n", ds)
}

func chkerr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
}

func extendedUsage() {
	fmt.Printf("The Go Directory Project - schema checking utility\n\n")
	fmt.Printf("Source: https://github.com/go-directory/schema/tree/main/cmd/schema-check.go\n")
	fmt.Printf("Need help? Open an issue at https://github.com/go-directory/schema/issues\n\n")

	fmt.Printf("Examples:\n\n")

	fmt.Printf("\t# Parse and then print schema stats\n")
	fmt.Printf("\t$ go run schema-check.go -f /path/to/your/schema/dirOrFile -s\n\n")
	fmt.Printf("\t# Parse and then dump schema to stdout\n")
	fmt.Printf("\t$ go run schema-check.go -f /path/to/your/schema/dirOrFile -d\n\n")
	fmt.Printf("\t# Parse only, no output if successful\n")
	fmt.Printf("\t$ go run schema-check.go -f /path/to/your/schema/dirOrFile\n\n")

	fmt.Printf("Extended usage information:\n\n")

	fmt.Printf("\t-f\tFull path to schema file or schema directory structure.\n\n")
	fmt.Printf("\t  \tThe path must be readable by the executing user.\n\n")

	fmt.Printf("\t  \tIf a FILE is being parsed, ensure that all definitions are ordered\n")
	fmt.Printf("\t  \tas 'dependency first, dependent second'. For example, 'cn' must be\n")
	fmt.Printf("\t  \tparsed *after* its super type 'name', not *before*.\n\n")

	fmt.Printf("\t  \tIf a DIRECTORY is being parsed, the same logic as FILE parsing applies\n")
	fmt.Printf("\t  \tand, as such, it is necessary to ensure that files (as well as any\n")
	fmt.Printf("\t  \tsubdirectories) are named in a manner that fosters the correct order.\n\n")

	fmt.Printf("\t-d\t<no argument>\n\n")
	fmt.Printf("\t  \tPrint the contents of the schema after successful parsing; mutually\n")
	fmt.Printf("\t  \texclusie to the -s (stats) flag.\n\n")

	fmt.Printf("\t-s\t<no argument>\n\n")
	fmt.Printf("\t  \tPrint a counter for all definitions parsed by category after successful\n")
	fmt.Printf("\t  \tparsing; mutually exclusive to the -d (dump) flag.\n\n")

	fmt.Printf("\t-p\t<no argument>\n\n")
	fmt.Printf("\t  \tAutomatically 'prime' the subschemaSubentry with LDAP syntaxes and\n")
	fmt.Printf("\t  \tmatching rules. This is necessary if the parsed schema does not include\n")
	fmt.Printf("\t  \tany such definitions, but contains definitions that depend upon them.\n\n")

	fmt.Printf("\t-h\t<no argument>\n\n")
	fmt.Printf("\t  \tShow available CLI flags and exit. Mutually exclusive to all other flags.\n\n")

	fmt.Printf("\t-e\t<no argument>\n\n")
	fmt.Printf("\t  \tShow this help text and exit. Mutually exclusive to all other flags.\n\n")

	fmt.Printf("** Please DONATE to animal charities and/or VOLUNTEER at animal shelters in your area **\n\n")
}
