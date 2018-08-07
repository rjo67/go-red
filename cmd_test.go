package main

import (
	"bytes"
	"testing"
)

func TestPrint(t *testing.T) {
	lineList := createListOfLines([]string{"first line", "second line"})
	dotline = lineList.Front()

	// to capture the output
	var buff bytes.Buffer // implements io.Writer

	_print(&buff, lineList)
	if buff.String() != "first line\n" {
		t.Fatalf("expected 'first line' but got %s", buff.String())
	}
	dotline = dotline.Next()
	buff.Reset()
	_print(&buff, lineList)
	if buff.String() != "second line\n" {
		t.Fatalf("expected 'second line' but got %s", buff.String())
	}
}
func TestPrintRange(t *testing.T) {
	lineList := createListOfLines([]string{"1", "2", "3", "4", "5"})

	// to capture the output
	var buff bytes.Buffer // implements io.Writer

	_printRange(&buff, lineList, AddressRange{Address{addr:2}, Address{addr:3}})
	if buff.String() != "2\n3\n" {
		t.Fatalf("2,3p returned %s", buff.String())
	}

	buff.Reset()
	_printRange(&buff, lineList, AddressRange{Address{addr:1}, Address{addr:4}})
	if buff.String() != "1\n2\n3\n4\n" {
		t.Fatalf("1,4p returned %s", buff.String())
	}

	buff.Reset()
	_printRange(&buff, lineList, AddressRange{Address{addr:3}, Address{addr:3}})
	if buff.String() != "3\n" {
		t.Fatalf("3,3p returned %s", buff.String())
	}
}

func TestMoveToLine(t *testing.T) {
	data := []string{"first", "second", "3", "", "5"}
	lineList := createListOfLines(data)

	for i, str := range data {
		el, err := moveToLine(lineList, Address{addr:i+1})
		if err != nil {
			t.Fatalf("data element %d, got error: %s", i, err)
		}
		if el.Value.(Line).line != str+"\n" {
			t.Fatalf("bad data element %d, expected %s but got %s", i, str, el.Value.(Line).line)
		}
	}

}
