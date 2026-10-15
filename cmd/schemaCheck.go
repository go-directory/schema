package main

/*
schemaChecker.go offers a fast and easy schema
file or directory parsing utility.

It is intended for verification purposes only.

Note that this utility expects a complete schema,
an will error if dependency definitions are not
found (for example, it will fail parsing 'cn' if
'name' was not already seen.
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
		fFlag = flag.String("f", "", "Path of file or directory to parse")
		//sFlag = flag.Bool("s", false, "Show schema stats")
		pFlag = flag.Bool("p", false, "Prime the schema with syntaxes and matching rules")
		eFlag = flag.Bool("e", false, "Show extended usage information and then exit")
	)

	flag.Parse()

	//showStats := *sFlag
	prime := *pFlag
	target := *fFlag
	exhelp := *eFlag

	if exhelp {
		extendedUsage()
		os.Exit(1)
	}

	if target == "" {
		err = fmt.Errorf("Must specify file or directory path using '-f'")
	}
	chkerr(err)

	sch, err = schema.New(prime)
	chkerr(err)

	x, err := os.Stat(target)
	if x.Mode().IsRegular() {
		err = sch.ReadFile(target)
	} else if x.Mode().IsDir() {
		err = sch.ReadDirectory(target)
	} else if err == nil {
		err = fmt.Errorf("Target path %q is neither a file nor a directory", target)
	}
	chkerr(err)

	//if showStats { ... }

	os.Exit(0) // Cool schema, bro
}

func chkerr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
}

func extendedUsage() {
	fmt.Printf("The Go Directory Project - schema checking utility\n\n")
	fmt.Printf("Source: https://github.com/go-directory/schema/tree/main/cmd/schemaCheck.go\n")
	fmt.Printf("Need help? Open an issue at https://github.com/go-directory/schema/issues\n")
	fmt.Printf("** Please donate to animal charities or volunteer at local shelters **\n\n")

	fmt.Printf("Usage\n\n")

	fmt.Printf("\t-f\tFull path to schema file or schema directory structure.\n\n")
	fmt.Printf("\t  \tThe path must be readable by the executing user.\n\n")

	fmt.Printf("\t  \tIf a FILE is being parsed, ensure that all definitions are ordered\n")
	fmt.Printf("\t  \tas 'dependency first, dependent second'. For example, 'cn' must be\n")
	fmt.Printf("\t  \tparsed *after* its super type 'name', not *before*.\n\n")

	fmt.Printf("\t  \tIf a DIRECTORY is being parsed, the same logic applies to FILE parsing\n")
	fmt.Printf("\t  \tand, as such, it is necessary to ensure that files (and any subdirectories)\n")
	fmt.Printf("\t  \tare named in a manner that fosters the correct ordering scheme.\n\n")

	fmt.Printf("\t-s\t<no argument>\n\n")
	fmt.Printf("\t  \tPrint out counters of parsed definitions by category.\n\n")

	fmt.Printf("\t-p\t<no argument>\n\n")
	fmt.Printf("\t  \tAutomatically 'prime' the subschemaSubentry with LDAP syntaxes and\n")
	fmt.Printf("\t  \tmatching rules. This is necessary if the parsed schema does not include\n")
	fmt.Printf("\t  \tany such definitions.\n\n")

	fmt.Printf("\t-h\t<no argument>\n\n")
	fmt.Printf("\t  \tShow available CLI flags.\n\n")

	fmt.Printf("\t-e\t<no argument>\n\n")
	fmt.Printf("\t  \tShow this help text.\n\n")
}
