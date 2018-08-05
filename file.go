package main

import (
	"bufio"
	"container/list"
	//"fmt"
	"io"
	"os"
)

/*
 * Reads the entire file identified by 'filename'.
 * Each line is added to a list structure which is returned.
 * The number of bytes read is also returned.
 * Non-EOF errors are returned in the error variable.
 *
 * The file is closed when this function returns.
 */
func ReadFile(filename string) (nbrBytesRead int, listOfLines *list.List, err error) {
	file, err := os.Open(filename)

	if err != nil {
		return
	}

	defer file.Close()

	// Start reading from the file with a reader
	reader := bufio.NewReader(file)
	return ReadReader(reader)
}

/*
 * Reads the entire contents of the 'reader'.
 * Each line is added to a list structure which is returned.
 * The number of bytes read is also returned.
 * Non-EOF errors are returned in the error variable.
 */
func ReadReader(reader *bufio.Reader) (nbrBytesRead int, listOfLines *list.List, err error) {

	listOfLines = list.New()

	var lineStr string
	for lineNbr := 1; ; lineNbr++ {
		lineStr, err = reader.ReadString('\n')

		//fmt.Printf(" > Read %d characters, err=%v\n", len(lineStr), err)

		// if EOF comes directly after \n, then get length=0 and err=EOF
		// (at least using a StringReader)
		if len(lineStr) != 0 {
			// lineNbr is no longer stored in Line
			listOfLines.PushBack(Line{lineStr})
			nbrBytesRead += len(lineStr)
		}

		if err != nil {
			break
		}
	}

	if err != io.EOF {
		return nbrBytesRead, nil, err
	}

	return nbrBytesRead, listOfLines, nil
}

/*
 * Writes the list contents to a file identified by 'filename'.
 * An existing file will be truncated.
 *
 * The number of bytes written is returned.
 *
 * The file is closed when this function returns.
 */
func WriteFile(filename string, listOfLines *list.List) (nbrBytesWritten int, err error) {
	file, err := os.Create(filename)

	if err != nil {
		return
	}

	defer file.Close()

	w := bufio.NewWriter(file)
	return WriteWriter(w, listOfLines)
}

/*
 * Writes the given list to the 'writer'.
 * The number of bytes written is returned.
 */
func WriteWriter(w *bufio.Writer, listOfLines *list.List) (nbrBytesWritten int, err error) {
	for e := listOfLines.Front(); e != nil; e = e.Next() {
		line := e.Value.(Line)
		nbrBytes, err := w.WriteString(line.line)
		if err != nil {
			return nbrBytesWritten, err
		}
		nbrBytesWritten += nbrBytes
	}

	w.Flush()
	return
}
