package schema

import (
	"fmt"
)

/*
This example demonstrates use of the [AttributeTypeProperties.EffectiveUsage]
method. This is used for cases where a particular attribute is subtyped and
does not explicitly state its USAGE clause value, e.g.:

( 1.3.6.1.4.1.56521.389.2.60.3.4 NAME 'configTLSClientCert' SUP configTLSListenerCert )

This method will climb the attribute's super type chain until it arrives at a
definitive answer. If the end of the chain is reached without a determination,
the method defers to the established default USAGE of "userApplications" (0),
as per RFC 4512.
*/
func ExampleAttributeTypeProperties_EffectiveUsage() {
	usageName, usageNumber:= exampleSchema.AT().EffectiveUsage([]byte(`configTLSClientCert`))
	fmt.Printf("name: %q (%d)", usageName, usageNumber)
	// Output:
	// name: "dSAOperation" (3)
}

/*
This example demonstrates use of the [AttributeTypeProperties.EffectiveFlags]
method. This is used for cases where a particular attribute is subtyped and
does not explicitly declare its OBSOLETE, NO-USER-MODIFICATIONS, COLLECTIVE
and/or SINGLE-VALUE Boolean clause values, e.g.:

( 1.3.6.1.4.1.56521.389.2.60.3.4 NAME 'configTLSClientCert' SUP configTLSListenerCert )

This method will climb the attribute's super type chain until it arrives at a
definitive answer. If the end of the chain is reached without a determination,
the method simply returns all false Boolean values, as it is not out of the
realm of possibility that none of these flags were actually set for the type.
*/
func ExampleAttributeTypeProperties_EffectiveFlags() {
	single, collect, nousermod, obsolete :=
		exampleSchema.AT().EffectiveFlags([]byte(`configTLSClientCert`))

        fmt.Printf("single:%t, collect:%t, nousermod:%t, obsolete:%t",
		single, collect, nousermod, obsolete)
        // Output:
        // single:true, collect:false, nousermod:false, obsolete:false
}

