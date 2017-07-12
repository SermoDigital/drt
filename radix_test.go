package drt

import (
	"bufio"
	"compress/gzip"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

const pwlist = "password_list.db"

var (
	wordList = makeWords()
	gt       *Trie
	honeynet = filepath.Join("_testdata", "honeynet.txt.gz")
)

func TestMain(m *testing.M) {
	setup()
	var err error
	gt, err = Open(pwlist)
	if err != nil {
		panic(err)
	}
	status := m.Run()
	err = gt.Close()
	if err != nil {
		panic(err)
	}
	os.Remove(pwlist)
	os.Exit(status)
}

func setup() {
	if _, err := os.Stat(pwlist); err != nil && !os.IsNotExist(err) {
		panic(err)
	}
	t, err := Create(pwlist)
	if err != nil {
		panic(err)
	}
	defer t.Close()
	file, err := os.Open(honeynet)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	r, err := gzip.NewReader(file)
	if err != nil {
		panic(err)
	}
	defer r.Close()
	s := bufio.NewScanner(r)
	for s.Scan() {
		t.Insert(s.Text())
	}
	if err := s.Err(); err != nil {
		panic(err)
	}
}

func makeWords() [][]byte {
	file, err := os.Open(honeynet)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	r, err := gzip.NewReader(file)
	if err != nil {
		panic(err)
	}
	defer r.Close()

	s := bufio.NewScanner(r)

	var words [][]byte
	for s.Scan() {
		words = append(words, []byte(s.Text()))
	}
	return words
}

func randslice(t *testing.T) []byte {
	buf := make([]byte, (rand.Intn(25-8)+1)+8)
	_, err := rand.Read(buf[:])
	if err != nil {
		if t == nil {
			panic(err)
		}
		t.Fatal(err)
	}
	return buf[:]
}

func TestHas(t *testing.T) {
	const N = 50000
	for i := 0; i < N; i++ {
		word := wordList[i%len(wordList)]
		if !gt.Has(word) {
			t.Fatalf("#%d: (%q) was not found", i, word)
		}
	}

	for i := 0; i < N; i++ {
		rs := randslice(t)
		if gt.Has(rs) {
			t.Fatalf("randslice (%#v) was found", rs)
		}
	}
}

var ghas bool

func BenchmarkHas(b *testing.B) {
	var lhas bool
	for i := 0; i < b.N; i++ {
		lhas = gt.Has(wordList[i%len(wordList)])
	}
	ghas = lhas
}
