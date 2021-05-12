package datastore

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestDb_PutGet(t *testing.T) {
	dir, err := ioutil.TempDir("", "test-db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	db, err := NewDb(dir, 256, nil)
	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()
	for i := 0; i < 20; i++ {
		value := fmt.Sprintf("gogadoda%d", i)
		err = db.Put("key1", value)
		if err != nil {
			t.Fatal(err)
		}
		str, err := db.Get("key1")
		if err != nil {
			t.Fatal(err)
		}
		if value != str {
			fmt.Printf("db.Get must return '%s' but returned '%s'", value, str)
		}
	}

	var fileCount int64 = 0
	err = filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		fmt.Println("file: ", path, "size:", f.Size())
		if !f.IsDir() {
			input, err := os.Open(path)
			if err != nil {
				return err
			}
			b, err := ioutil.ReadAll(input)
			if err != nil {
				return err
			}
			fmt.Println("content:\n", string(b), "\n-------")
			fileCount++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if fileCount != 2 {
		t.Fatal("count of segments must be 2 by recieved:", fileCount)
	}
}

func TestDb_PutGetParalel(t *testing.T) {
	dir, err := ioutil.TempDir("", "test-db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	db, err := NewDb(dir, 256, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	values := make(map[string]string)
	for i := 0; i < 1000; i++ {
		values[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
	}

	for k, v := range values {
		err = db.Put(k, v)
		if err != nil {
			t.Fatal(err)
		}
	}
	go func() {
		for k, v := range values {
			err = db.Put(k, fmt.Sprintf("%s_new", v))
		}
	}()

	for k, v := range values {
		str, err := db.Get(k)
		if err != nil {
			t.Fatal(err)
		}
		newV := fmt.Sprintf("%s_new", v)
		if v != str && newV != str {
			fmt.Printf("db.Get must return '%s' or '%s' but returned '%s'", v, newV, str)
		}
	}
}

func TestDb_Merge(t *testing.T) {
	dir, err := ioutil.TempDir("", "test-db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	db, err := NewDb(dir, 256, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	values := make(map[string]string)
	for i := 0; i < 14; i++ {
		values[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
	}

	for k, v := range values {
		for j := 0; j < 2; j++ {
			err = db.Put(k, v)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	var fileCount0 int64 = 0
	err = filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		fmt.Println("file: ", path, "size:", f.Size())
		if !f.IsDir() {
			fileCount0++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	err = db.Merge()
	if err != nil {
		t.Fatal(err)
	}

	var fileCount1 int64 = 0
	err = filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		fmt.Println("file: ", path, "size:", f.Size())
		if !f.IsDir() {
			fileCount1++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range values {
		str, err := db.Get(k)
		if err != nil {
			t.Fatal(err)
		}
		if v != str {
			fmt.Printf("db.Get must return '%s' but returned '%s'", v, str)
		}
	}

	if fileCount0 <= fileCount1 {
		t.Fatal("must contaist less segments after merge")
	}
}
