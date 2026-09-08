package schema

import (
	_ "embed"
	//"fmt"
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
		err = exampleSchema.ReadBytes(exampleSchemaFile)
	}
	return
}
