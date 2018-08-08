package main

import (
	"bytes"
	"testing"
)

func TestPrintRange(t *testing.T) {
	state := State{}
	state.buffer = createListOfLines([]string{"1", "2", "3", "4", "5"})
	state.lastLineNbr = 5
	cmd := Command{AddressRange{Address{addr: 2}, Address{addr: 3}}, "p", ""}

	// to capture the output
	var buff bytes.Buffer // implements io.Writer

	err := _printRange(&buff, cmd, &state)
	if err != nil {
		t.Fatalf("error %s", err)
	}
	if buff.String() != "2\n3\n" {
		t.Fatalf("2,3p returned '%s'", buff.String())
	}

	buff.Reset()
	cmd = Command{AddressRange{Address{addr: 1}, Address{addr: 4}}, "p", ""}
	err = _printRange(&buff, cmd, &state)
	if err != nil {
		t.Fatalf("error %s", err)
	}
	if buff.String() != "1\n2\n3\n4\n" {
		t.Fatalf("1,4p returned '%s'", buff.String())
	}

	buff.Reset()
	cmd = Command{AddressRange{Address{addr: 3}, Address{addr: 3}}, "p", ""}
	err = _printRange(&buff, cmd, &state)
	if err != nil {
		t.Fatalf("error %s", err)
	}
	if buff.String() != "3\n" {
		t.Fatalf("3,3p returned '%s'", buff.String())
	}
}

func TestMoveToLine(t *testing.T) {
	state := State{}
	data := []string{"first", "second", "3", "", "5"}

	state.buffer = createListOfLines(data)

	for i, expected := range data {
		moveToLine(i+1, &state)
		if state.lineNbr != (i + 1) {
			t.Fatalf("bad state.lineNbr, expected %d but got %d", i+1, state.lineNbr)
		}
		if state.dotline.Value.(Line).line != expected+"\n" {
			t.Fatalf("bad data element %d, expected '%s' but got '%s'", i, expected, state.dotline.Value.(Line).line)
		}
	}

}
