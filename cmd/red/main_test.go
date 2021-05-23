package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/rjo67/red"
)

/*
Tests which execute the given ED commands against an input file and check the result.
*/
func TestMainLoop(t *testing.T) {

	data := []struct {
		filename string
	}{
		{"test1.txt"},
	}

	for _, test := range data {
		t.Run(fmt.Sprintf("processing file %s", test.filename), func(t *testing.T) {
			expectedOutputFilename := "testfiles/compare-" + test.filename
			commandsFilename := "testfiles/commands-" + test.filename
			inputFilename := "testfiles/" + test.filename
			outputFilename := "testfiles/output/" + test.filename

			// make sure output file is not present
			err := os.Remove(outputFilename)
			if err != nil {
				t.Fatalf("could not delete output file: %s", err)
			}

			state := red.NewState()

			// read input file (as if specified on the command line)
			err = readInputFile(inputFilename, state)
			if err != nil {
				t.Fatalf("error reading input file: %s", err)
			}

			// open commands file
			f, err := os.Open(commandsFilename)
			if err != nil {
				t.Fatalf("error opening commands file: %s", err)
			}
			defer f.Close()
			reader := bufio.NewReader(f)

			// GO
			mainloop(state, reader)

			// compare
			err = fileCompare(outputFilename, expectedOutputFilename)
			if err != nil {
				t.Errorf("compare '%s' and '%s': %s ", outputFilename, expectedOutputFilename, err)
			}
		})
	}
}

func fileCompare(filename1, filename2 string) error {
	f1, err := os.Open(filename1)
	if err != nil {
		return fmt.Errorf("i/o error: %w", err)
	}
	defer f1.Close()
	f2, err := os.Open(filename2)
	if err != nil {
		return fmt.Errorf("i/o error: %w", err)
	}
	defer f2.Close()

	reader1 := bufio.NewReader(f1)
	reader2 := bufio.NewReader(f2)

	eof1, eof2 := false, false
	for lineNbr := 1; !(eof1 || eof2); lineNbr++ {
		b1, err1 := reader1.ReadBytes('\n')
		if err1 != nil {
			if err1 == io.EOF {
				eof1 = true
			} else {
				return fmt.Errorf("unexpected error file1: %w", err)
			}
		}
		b2, err2 := reader2.ReadBytes('\n')
		if err2 != nil {
			if err2 == io.EOF {
				eof2 = true
			} else {
				return fmt.Errorf("unexpected error file2: %w", err)
			}
		}
		if eof1 && !eof2 {
			return fmt.Errorf("unexpected eof file1 (line %d)", lineNbr)
		} else if eof2 && !eof1 {
			return fmt.Errorf("unexpected eof file2 (line %d)", lineNbr)
		}
		if !bytes.Equal(b1, b2) {
			return fmt.Errorf("files differ at line %d", lineNbr)
		}
	}
	return nil
}
