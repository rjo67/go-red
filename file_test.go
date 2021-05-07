package red

import (
	"bufio"
	"bytes"
	//   "fmt"
	"container/list"
	"os"
	"strings"
	"testing"
)

type testdata []struct {
	lineLength int
}

func TestLongLines(t *testing.T) {
	const filename string = "longline.txt"
	createFileWithLongLine(filename)
	defer os.Remove(filename)

	data := testdata{
		{4194304 + 1}, // first line is 4MB + 1
		{12},          // second line has no \n
	}

	doReadTestWithFile(t, data, filename)
}

func TestLinesThatDoNotFinishWithALinebreak(t *testing.T) {
	const filename string = "nolinebreak.txt"
	createFileThatDoesNotEndWithALineBreak(filename)
	defer os.Remove(filename)

	data := testdata{
		{28},
	}
	doReadTestWithFile(t, data, filename)
}

func TestStringReader(t *testing.T) {
	str := "line1\n\neol\n"

	data := testdata{
		{6}, {1}, {4},
	}
	reader := strings.NewReader(str)
	doReadTestWithReader(t, data, bufio.NewReader(reader))
}

func TestStringWriter(t *testing.T) {
	listOfLines := createListOfLines([]string{"first line", "second line"})

	createWriterAndDoTest(t, listOfLines)
}

func createListOfLines(lines []string) *list.List {
	listOfLines := list.New()
	for _, line := range lines {
		listOfLines.PushBack(Line{line + "\n"})
	}
	return listOfLines
}

/* --------------------  helper routines ---------------- */

func doReadTestWithFile(t *testing.T, data testdata, filename string) {
	nbrBytes, myList, err := ReadFile(filename)
	if err != nil {
		t.Fatalf("got error message %v", err)
	}
	doReadTest(t, data, nbrBytes, myList)
}
func doReadTestWithReader(t *testing.T, data testdata, reader *bufio.Reader) {
	nbrBytes, myList, err := ReadReader(reader)
	if err != nil {
		t.Fatalf("got error message %v", err)
	}
	doReadTest(t, data, nbrBytes, myList)
}
func doReadTest(t *testing.T, data testdata, nbrBytes int, myList *list.List) {
	// length of list (file lines) must equal length of testdata
	if myList.Len() != len(data) {
		t.Fatalf("Expected %d lines but got %d", len(data), myList.Len())
	}
	expectedNbrBytes := 0
	var e *list.Element
	for currentLine, want := range data {
		if e == nil {
			e = myList.Front()
		} else {
			e = e.Next()
		}
		line := e.Value.(Line)
		if len(line.Line) != want.lineLength {
			t.Fatalf("Bad line length at line %d, expected %d but got %d", currentLine+1, want.lineLength, len(line.Line))
		}
		expectedNbrBytes += len(line.Line)
	}
	if expectedNbrBytes != nbrBytes {
		t.Fatalf("Expected %d bytes but read %d", expectedNbrBytes, nbrBytes)
	}
}

func createWriterAndDoTest(t *testing.T, listOfLines *list.List) {
	var buff bytes.Buffer               // implements io.Writer
	var writer = bufio.NewWriter(&buff) // -> bufio

	// tot up total number of bytes in list and build the string
	expectedNbrBytes := 0
	var sb strings.Builder
	for e := listOfLines.Front(); e != nil; e = e.Next() {
		line := e.Value.(Line)
		sb.WriteString(line.Line)
		expectedNbrBytes += len(line.Line)
	}
	// sanity check -- should never fail
	if expectedNbrBytes != len(sb.String()) {
		t.Fatalf("Bad string length, expected %d but got %d", expectedNbrBytes, len(sb.String()))
	}
	nbrBytesWritten := doWriteTest(t, listOfLines, writer)
	// make sure the correct number of bytes were written
	if nbrBytesWritten != expectedNbrBytes {
		t.Fatalf("Bad nbrBytesWritten, expected %d but got %d", expectedNbrBytes, nbrBytesWritten)
	}
	// make sure the contents of the file are as expected
	if sb.String() != buff.String() {
		t.Fatalf("Bad content, expected %s but got %s", sb.String(), buff.String())
	}
	t.Logf("got %s", buff.String())
}

func doWriteTest(t *testing.T, myList *list.List, writer *bufio.Writer) (nbrBytesWritten int) {

	nbrBytesWritten, err := WriteWriter(writer, myList.Front(), 1, myList.Len())
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	return nbrBytesWritten
}

func createFileThatDoesNotEndWithALineBreak(fn string) (err error) {
	file, err := os.Create(fn)

	if err != nil {
		return err
	}

	defer file.Close()

	w := bufio.NewWriter(file)
	w.WriteString("Does not end with linebreak.")
	w.Flush()

	return
}

func createFileWithLongLine(fn string) (err error) {
	file, err := os.Create(fn)
	defer file.Close()

	if err != nil {
		return err
	}

	w := bufio.NewWriter(file)

	fs := 1024 * 1024 * 4 // 4MB

	// Create a 4MB long line consisting of the letter a.
	for i := 0; i < fs; i++ {
		w.WriteRune('a')
	}

	// Terminate the line with a break.
	w.WriteRune('\n')

	// Put in a second line, which doesn't have a linebreak.
	w.WriteString("Second line.")

	w.Flush()

	return
}
