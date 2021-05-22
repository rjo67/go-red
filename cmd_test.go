package red

import (
	"bufio"
	"bytes"
	"testing"
)

func TestDelete(t *testing.T) {
	state := NewState()
	state.Buffer = createListOfLines([]string{"1", "2", "3", "4", "5"})
	cmd := Command{newValidRange("2,3"), commandDelete, ""}

	// to capture the output
	var buff bytes.Buffer               // implements io.Writer
	var writer = bufio.NewWriter(&buff) // -> bufio

	err := cmd.CmdDelete(state, true)
	checkError(t, err)
	if state.lineNbr != 2 {
		t.Fatalf("wrong state.lineNbr! got %d, expected 2", state.lineNbr)
	}
	_, err = WriteWriter(writer, state.Buffer.Front(), 1, state.Buffer.Len())
	checkError(t, err)
	if buff.String() != "1\n4\n5\n" {
		t.Fatalf("2,3d returned '%s'", buff.String())
	}

	//  delete whole file
	state.Buffer = createListOfLines([]string{"1", "2", "3", "4", "5"})
	cmd = Command{newValidRange("1, 5"), commandDelete, ""}
	buff.Reset()
	err = cmd.CmdDelete(state, true)
	checkError(t, err)
	if state.lineNbr != 0 {
		t.Fatalf("wrong state.lineNbr! got %d, expected 0", state.lineNbr)
	}
	_, err = WriteWriter(writer, state.Buffer.Front(), 1, state.Buffer.Len())
	checkError(t, err)
	if buff.String() != "" {
		t.Fatalf("1,5d returned '%s'", buff.String())
	}
}

func TestPrintRange(t *testing.T) {
	state := State{}
	state.Buffer = createListOfLines([]string{"1", "2", "3", "4", "5"})
	cmd := Command{newValidRange("2, 3"), commandPrint, ""}

	// to capture the output
	var buff bytes.Buffer // implements io.Writer

	err := _printRange(&buff, cmd, &state, false)
	checkError(t, err)
	if buff.String() != "2\n3\n" {
		t.Fatalf("2,3p returned '%s'", buff.String())
	}

	buff.Reset()
	cmd = Command{newValidRange("1, 4"), commandPrint, ""}
	err = _printRange(&buff, cmd, &state, false)
	checkError(t, err)
	if buff.String() != "1\n2\n3\n4\n" {
		t.Fatalf("1,4p returned '%s'", buff.String())
	}

	buff.Reset()
	cmd = Command{newValidRange("3, 3"), commandPrint, ""}
	err = _printRange(&buff, cmd, &state, false)
	checkError(t, err)
	if buff.String() != "3\n" {
		t.Fatalf("3,3p returned '%s'", buff.String())
	}

	buff.Reset()
	// currently at line 3
	cmd, err = ParseCommand("+1")
	checkError(t, err)
	err = _printRange(&buff, cmd, &state, false)
	checkError(t, err)
	if buff.String() != "4\n" {
		t.Fatalf("+1 returned '%s'", buff.String())
	}
}

func checkError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("error %s", err)
	}
}

/*
func TestPrintWithFile(t *testing.T) {

	//	t.Skip("currently skipping this test")

	state := NewState()
	cmd, err := ParseCommand("r test.txt")
	err = cmd.CmdRead(state)
	if err != nil {
		t.Fatalf("error %s", err)
	}

	// to capture the output
	var buff bytes.Buffer // implements io.Writer

	cmd, err = ParseCommand("p 3,5")
	checkError(t, err)
	err = _printRange(&buff, cmd, state, false)
	checkError(t, err)
	if buff.String() != "1\n" {
		t.Fatalf("empty command returned '%s'", buff.String())
	}
}
*/

func TestMoveToLine(t *testing.T) {
	state := State{}
	data := []string{"first", "second", "3", "", "5"}

	state.Buffer = createListOfLines(data)

	for i, expected := range data {
		moveToLine(i+1, &state)
		if state.lineNbr != (i + 1) {
			t.Fatalf("bad state.lineNbr, expected %d but got %d", i+1, state.lineNbr)
		}
		if state.dotline.Value.(Line).Line != expected+"\n" {
			t.Fatalf("bad data element %d, expected '%s' but got '%s'", i, expected, state.dotline.Value.(Line).Line)
		}
	}

}
