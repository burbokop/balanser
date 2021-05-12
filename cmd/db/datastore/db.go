package datastore

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

var ErrNotFound = fmt.Errorf("record does not exist")

type hashIndex map[string]int64

type segment struct {
	Name string
	Size int64
	Crc  []byte
}

type Db struct {
	dir                string
	segments           []segment
	currentSegment     *os.File
	outOffset          int64
	segmentFilePreffix string
	autoMergeDuration  *time.Duration
	maxSegmentSize     int64
	index              hashIndex
	opened             *bool
	writeFileFlag      int
	writeFileMode      os.FileMode
	mutex              sync.Mutex
}

func NewDb(dir string, maxSegmentSize int64, autoMergeDuration *time.Duration) (*Db, error) {
	_ = os.MkdirAll(dir, os.ModeDir)
	db := &Db{
		dir:                dir,
		maxSegmentSize:     maxSegmentSize,
		segmentFilePreffix: "segment",
		autoMergeDuration:  autoMergeDuration,
		writeFileFlag:      os.O_APPEND | os.O_RDWR | os.O_CREATE,
		writeFileMode:      0o600,
	}
	err := db.recoverAll()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (db *Db) nextSegment() error {
	nextSegmentName := fmt.Sprintf("%s%d", db.segmentFilePreffix, db.outOffset)
	outputPath := filepath.Join(db.dir, nextSegmentName)
	f, err := os.OpenFile(outputPath, db.writeFileFlag, db.writeFileMode)
	if err != nil {
		return err
	}
	db.currentSegment = f
	db.segments = append(db.segments, segment{Name: nextSegmentName, Size: 0})
	return nil
}

func (db *Db) segmentStartPosition(segment string) (int64, error) {
	return strconv.ParseInt(segment[len(db.segmentFilePreffix):], 10, 64)
}

func (db *Db) findSegment(index int64) (path string, crc []byte, startPos int64, err error) {
	for _, s := range db.segments {
		i, err := db.segmentStartPosition(s.Name)
		if err != nil {
			return "", nil, 0, err
		}
		if i <= index && index < i+s.Size {
			return filepath.Join(db.dir, s.Name), s.Crc, i, nil
		}
	}
	return "", nil, 0, fmt.Errorf("segment for index %d not found", index)
}

func (db *Db) recoverAll() error {
	db.Close()
	err := filepath.Walk(db.dir, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			return db.recover(path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(db.segments) == 0 {
		db.segments = append(db.segments, segment{Name: fmt.Sprintf("%s%d", db.segmentFilePreffix, 0), Size: 0})
	}

	f, err := os.OpenFile(
		filepath.Join(db.dir, db.segments[len(db.segments)-1].Name),
		db.writeFileFlag,
		db.writeFileMode,
	)
	if err != nil {
		return err
	}
	db.currentSegment = f
	opnd := true
	db.opened = &opnd
	if db.autoMergeDuration != nil {
		go func() {
			for db.opened != nil {
				if !*db.opened {
					break
				}
				time.Sleep(*db.autoMergeDuration)
				db.Merge()
			}
		}()
	}

	return nil
}

func (db *Db) recover(filePath string) error {
	input, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer input.Close()

	sp, err := db.segmentStartPosition(filepath.Base(filePath))
	if err != nil {
		return err
	}

	db.outOffset = sp
	size, err := readAll(bufio.NewReaderSize(input, bufSize), func(e entry, n int64) error {
		db.index[e.key] = db.outOffset
		db.outOffset += int64(n)
		return nil
	})
	if err != nil && err != io.EOF {
		return err
	}
	crc, err := db.crcOfPath(filePath)
	if err != nil {
		return fmt.Errorf("error calculation crc: %s", err)
	}

	db.segments = append(db.segments, segment{Name: filepath.Base(input.Name()), Size: size, Crc: crc})
	return nil
}

func (db *Db) Close() error {
	if db.currentSegment != nil {
		err := db.currentSegment.Close()
		if err != nil {
			return err
		}
		db.currentSegment = nil
	}
	db.index = make(hashIndex)
	db.segments = make([]segment, 0)
	db.outOffset = 0
	if db.opened != nil {
		*db.opened = false
	}
	return nil
}

func (db *Db) Get(key string) (string, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	position, ok := db.index[key]
	if !ok {
		return "", ErrNotFound
	}

	path, segmentCrc, startPos, err := db.findSegment(position)
	if err != nil {
		return "", err
	}

	crc, err := db.crcOfPath(path)
	if err != nil {
		return "", fmt.Errorf("error getting crc in get: %s", err)
	}

	if bytes.Compare(segmentCrc, crc) != 0 {
		return "", fmt.Errorf("wrong crc: %b, %b", segmentCrc, crc)
	}

	position -= startPos
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = file.Seek(position, 0)
	if err != nil {
		return "", err
	}

	reader := bufio.NewReader(file)
	value, err := readValue(reader)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (db *Db) crcOfPath(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return db.crcOfFile(f)
}

func (db *Db) crcOfFile(f *os.File) ([]byte, error) {
	pos, err := f.Seek(0, 1)
	if err != nil {
		return nil, err
	}
	_, err = f.Seek(0, 0)
	if err != nil {
		return nil, err
	}
	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}
	_, err = f.Seek(pos, 0)
	if err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func (db *Db) Put(key, value string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	stat, err := db.currentSegment.Stat()
	if err != nil {
		return fmt.Errorf("error gettitng file stat: %s", err)
	}
	if stat.Size() > db.maxSegmentSize {
		err := db.nextSegment()
		if err != nil {
			return fmt.Errorf("error going to next segment: %s", err)
		}
	}
	e := entry{
		key:   key,
		value: value,
	}
	n, err := db.currentSegment.Write(e.Encode())
	if err != nil {
		return err
	}
	db.index[key] = db.outOffset
	db.outOffset += int64(n)
	db.segments[len(db.segments)-1].Size += int64(n)
	crc, err := db.crcOfFile(db.currentSegment)
	if err != nil {
		return fmt.Errorf("error calculation crc on put: %s", err)
	}
	db.segments[len(db.segments)-1].Crc = crc
	return nil
}

func (db *Db) readSegment(path string, values *map[string]string) (int64, error) {
	input, err := os.Open(path)
	if err != nil {
		return 0, err
	}

	defer input.Close()
	size, err := readAll(bufio.NewReaderSize(input, bufSize), func(e entry, n int64) error {
		(*values)[e.key] = e.value
		return nil
	})
	if err != nil && err != io.EOF {
		return 0, err
	}
	return size, nil
}

func (db *Db) blitSegments(dst string, src string) error {
	var values = make(map[string]string)
	_, err := db.readSegment(dst, &values)
	if err != nil {
		return fmt.Errorf("error reading dst segment (%s): %s", dst, err)
	}
	_, err = db.readSegment(src, &values)
	if err != nil {
		return fmt.Errorf("error reading src segment (%s): %s", src, err)
	}
	err = os.Remove(dst)
	if err != nil {
		return fmt.Errorf("error removing dst segment: %s", err)
	}
	err = os.Remove(src)
	if err != nil {
		return fmt.Errorf("error removing src segment: %s", err)
	}
	f, err := os.OpenFile(dst, db.writeFileFlag, db.writeFileMode)
	if err != nil {
		return fmt.Errorf("error opening new dst file: %s", err)
	}
	for k, v := range values {
		e := entry{
			key:   k,
			value: v,
		}
		_, err = f.Write(e.Encode())
		if err != nil {
			return fmt.Errorf("error priting pair: %s", err)
		}
	}
	return nil
}

func (db *Db) Merge() error {
	if len(db.segments) > 2 {
		fmt.Println("before", db.segments)

		err := db.blitSegments(
			filepath.Join(db.dir, db.segments[0].Name),
			filepath.Join(db.dir, db.segments[1].Name),
		)
		if err != nil {
			return err
		}

		err = db.recoverAll()
		fmt.Println("after", db.segments)

		return err
	}
	return nil
}
