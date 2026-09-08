package schema

import (
	"fmt"
)

func ExampleSubschemaSubentry_RegisterLDAPSyntax() {
	_ = exampleSchema.RegisterLDAPSyntax([]byte(`( 1.3.6.1.4.1.56521.999.77.88.1.6
		DESC 'Example X-PATTERN custom syntax'
		X-PATTERN '^\{([a-z](-?[A-Za-z0-9]+)*(\(\d+\))?)(\s([a-z](-?[A-Za-z0-9]+)*(\(\d+\))))*\}$' )`))

	fmt.Println(exampleSchema.LS().Get([]byte(`1.3.6.1.4.1.56521.999.77.88.1.6`)))
	// Output: ( 1.3.6.1.4.1.56521.999.77.88.1.6 DESC 'Example X-PATTERN custom syntax' X-PATTERN '^\{([a-z](-?[A-Za-z0-9]+)*(\(\d+\))?)(\s([a-z](-?[A-Za-z0-9]+)*(\(\d+\))))*\}$' )
}
