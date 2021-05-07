package red

import (
	"bytes"
	"testing"
)

func TestSubstitute(t *testing.T) {
	state := State{}
	state.Buffer = createListOfLines([]string{"rjo", "rjo", "my name is rjo", "bar"})

	// to capture the output
	var buff bytes.Buffer // implements io.Writer

	nbrLinesChanged, _, err := processLines(&buff, 2, state.Buffer.Len(), &state, "rjo", "foobar", "gp")
	if err != nil {
		t.Fatalf("error %s", err)
	}
	if nbrLinesChanged != 2 {
		t.Fatalf("wrong number of lines changed, expected %d but got %d", 2, nbrLinesChanged)
	}
	if buff.String() != "foobar\nmy name is foobar\n" {
		t.Fatalf("changed lines '%s'", buff.String())
	}

}
