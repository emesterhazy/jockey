package counter_test

import (
	"io"
	"jockey/counter"
	"strings"
	"testing"
)

func TestReaderOneRead(t *testing.T) {
	testStrings := []string{"Jockey", "Go", "Fast"}
	for _, s := range testStrings {
		strReader := strings.NewReader(s)
		nExpected := strReader.Len()
		countReader := counter.NewReader(strReader)
		buf := make([]byte, strReader.Len())
		n, err := countReader.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		if n != nExpected {
			t.Fatalf("got %d expected %d", n, nExpected)
		}
		if countReader.Count() != nExpected {
			t.Fatalf("got %d expected %d", n, nExpected)
		}
	}
}

func TestReaderManyReads(t *testing.T) {
	testString := "Jockey Go Fast"
	countExpected := len(testString)
	strReader := strings.NewReader(testString)
	countReader := counter.NewReader(strReader)
	buf := make([]byte, 1)
	for {
		n, err := countReader.Read(buf)
		if err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Fatalf("got %d expected 1 byte read", n)
		}
	}
	if countReader.Count() != countExpected {
		t.Fatalf("got %d expected %d", countReader.Count(), countExpected)
	}
}
