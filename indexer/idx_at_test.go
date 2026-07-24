package indexer

import (
	"fmt"
	"testing"
)

func ExampleAttributeTypeProperties_Resolve_reverseLookup() {
	numericOID, descriptor, altNames := exampleIndex.AT.Resolve(`2.5.4.3`)
	fmt.Printf("Principal name for %s: %q, alt names: %v", numericOID, descriptor, altNames)
	// Output: Principal name for 2.5.4.3: "cn", alt names: [cn commonName]
}

func ExampleAttributeTypeProperties_Resolve_forwardLookup() {
	numericOID, descriptor, altNames := exampleIndex.AT.Resolve(`cn`)
	fmt.Printf("Principal name for %s: %q, alt names: %v", numericOID, descriptor, altNames)
	// Output: Principal name for 2.5.4.3: "cn", alt names: [cn commonName]
}

func BenchmarkAttributeTypeResolve(b *testing.B) {
	b.StopTimer()
	attrs := []string{
		`cn`, `sn`, `l`, `objectClass`, `entryDN`,
		`entryUUID`, `gecos`, `mail`, `userPassword`,
	}
	maxIdx := len(attrs)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = exampleIndex.AT.Resolve(attrs[i%maxIdx])
	}
}

func BenchmarkAttributeTypeCalls(b *testing.B) {
	var attr []string
	for k := range exampleIndex.AT.O2D {
		// list of attribute type OIDs
		attr = append(attr, k)
	}

	b.StopTimer()
	maxIdx := len(attr)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, _ = exampleIndex.AT.D2O[attr[i%maxIdx]]
		_, _ = exampleIndex.AT.UB[attr[i%maxIdx]]
		_, _ = exampleIndex.AT.LS[attr[i%maxIdx]]
		_, _ = exampleIndex.AT.EQ[attr[i%maxIdx]]
		_, _ = exampleIndex.AT.Sup[attr[i%maxIdx]]
		_, _ = exampleIndex.AT.Sub[attr[i%maxIdx]]
		_, _ = exampleIndex.AT.Usage[attr[i%maxIdx]]
		_, _ = exampleIndex.AT.SrcIndex[attr[i%maxIdx]]
	}
}
