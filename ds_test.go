package schema

import (
	"fmt"
)

/*
This example demonstrates the means to determine the
'namedObjectClass' of a dITStructureRule.

	dITStructureRule.FORM -> nameForm.OC -> objectClass.NAME
*/
func ExampleDITStructureRuleProperties_NamedObjectClass() {

	rule := []byte(`1`) // integer identifier of indicated rule
	id, princ, _, _ := exampleSchema.DS().NamedObjectClass(rule)
	fmt.Printf("namedObjectClass %q (%s)", id, princ)
	// Output: namedObjectClass "0.9.2342.19200300.100.4.9" (documentSeries)
}

func ExampleSubschemaSubentry_UnregisterDITStructureRule_dependencyViolation() {
	if err := exampleSchema.UnregisterDITStructureRule([]byte(`1`)); err != nil {
		fmt.Println(err)
	}
	// Output: DITStructureRule '1' has 1 dependent structure rules
}
