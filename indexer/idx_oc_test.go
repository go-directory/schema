package indexer

import (
	"fmt"
	"testing"
)

func ExampleObjectClassProperties_Resolve_reverseLookup() {
	numericOID, descriptor, _ := exampleIndex.OC.Resolve(`2.5.6.0`)
	fmt.Printf("Principal name for %s: %q", numericOID, descriptor)
	// Output: Principal name for 2.5.6.0: "top"
}

func ExampleObjectClassProperties_Resolve_forwardLookup() {
	numericOID, descriptor, _ := exampleIndex.OC.Resolve(`top`)
	fmt.Printf("Principal name for %s: %q", numericOID, descriptor)
	// Output: Principal name for 2.5.6.0: "top"
}

func ExampleObjectClassProperties_superChain() {
	chain := exampleIndex.OC.Sup[`2.5.6.7`]
	fmt.Println(chain)
	// Output: [2.5.6.0 2.5.6.6 2.5.6.7]
}

func BenchmarkObjectClassCalls(b *testing.B) {
	var class []string
	for k := range exampleIndex.OC.O2D {
		// list of object class OIDs
		class = append(class, k)
	}

	b.StopTimer()
	maxIdx := len(class)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, _ = exampleIndex.OC.D2O[class[i%maxIdx]]
		_, _ = exampleIndex.OC.Obsolete[class[i%maxIdx]]
		_, _ = exampleIndex.OC.Must[class[i%maxIdx]]
		_, _ = exampleIndex.OC.May[class[i%maxIdx]]
		_, _ = exampleIndex.OC.Sup[class[i%maxIdx]]
		_, _ = exampleIndex.OC.Sub[class[i%maxIdx]]
		_, _ = exampleIndex.OC.Kind[class[i%maxIdx]]
		_, _ = exampleIndex.OC.SrcIndex[class[i%maxIdx]]
	}
}
