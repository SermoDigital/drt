package drt

import (
	"bufio"
	"compress/gzip"
	"log"
	"math/rand"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	setup()
	var err error
	gt, err = Open("password_list.db")
	if err != nil {
		log.Fatalln(err)
	}
	status := m.Run()
	err = gt.Close()
	if err != nil {
		log.Fatalln(err)
	}
	os.Remove("password_list.db")
	os.Exit(status)
}

func setup() {
	_, err := os.Stat("password_list.db")
	if !os.IsNotExist(err) {
		return
	}
	t, err := Create("password_list.db")
	if err != nil {
		log.Fatalln(err)
	}
	defer t.Close()
	file, err := os.Open("honeynet.txt.gz")
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()
	r, err := gzip.NewReader(file)
	if err != nil {
		log.Fatalln(err)
	}
	defer r.Close()
	s := bufio.NewScanner(r)
	for s.Scan() {
		t.Insert(s.Text())
	}
	err = s.Err()
	if err != nil {
		log.Fatalln(err)
	}
}

func randslice(t *testing.T) []byte {
	buf := make([]byte, (rand.Intn(25-8)+1)+8)
	_, err := rand.Read(buf[:])
	if err != nil {
		if t == nil {
			log.Fatalln(err)
		}
		t.Fatal(err)
	}
	return buf[:]
}

var gt *Trie

func TestHas(t *testing.T) {
	file, err := os.Open("honeynet.txt.gz")
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()
	r, err := gzip.NewReader(file)
	if err != nil {
		log.Fatalln(err)
	}
	defer r.Close()
	s := bufio.NewScanner(r)
	for i := 1; s.Scan(); i++ {
		if !gt.Has(s.Bytes()) {
			t.Fatalf("#%d: (%q) was not found", i, s.Bytes())
		}
	}

	for i := 0; i < 1000; i++ {
		rs := randslice(t)
		if gt.Has(rs) {
			t.Fatalf("randslice (%#v) was found", rs)
		}
	}
}
