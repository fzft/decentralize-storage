package storage

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"os"
	"path"
	"strings"
)

const defaultRootFolderName = "ggnetwork"

func CASPathTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	blockSize := 5
	sliceLen := len(hashStr) / blockSize
	paths := make([]string, sliceLen)
	for i := 0; i < sliceLen; i++ {
		from, to := i*blockSize, (i+1)*blockSize
		paths[i] = hashStr[from:to]
	}
	return PathKey{
		PathName: strings.Join(paths, "/"),
		Filename: hashStr,
	}
}

type PathTransformFunc func(string) PathKey

var DefaultPathTransformFunc = func(key string) PathKey {
	return PathKey{
		PathName: key,
		Filename: key,
	}
}

type PathKey struct {
	PathName string
	Filename string
}

// FullPath returns the filename of the file that will be stored
// in the path.
func (p PathKey) FullPath() string {
	return path.Join(p.PathName, string(os.PathSeparator), p.Filename)
}

// FirstPathName ...
func (p PathKey) FirstPathName() string {
	paths := strings.Split(p.PathName, string(os.PathSeparator))
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

type StoreOpts struct {
	// Root is the folder name of the root, containing all the files.
	Root string
	PathTransformFunc
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultPathTransformFunc
	}

	if opts.Root == "" {
		opts.Root = defaultRootFolderName
	}

	return &Store{
		StoreOpts: opts,
	}
}

// Write writes the contents of the reader to the file.
func (s *Store) Write(key string, r io.Reader) error {
	return s.writeStream(key, r)
}

// Read returns the contents of the file.
func (s *Store) Read(key string) (io.Reader, error) {
	f, err := s.readStream(key)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, f)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// Delete ...
func (s *Store) Delete(key string) error {
	pathKey := s.PathTransformFunc(key)
	return os.RemoveAll(path.Join(s.Root, pathKey.FirstPathName()))
}

// Clear ...
func (s *Store) Clear() error {
	return os.RemoveAll(s.Root)
}

// Exists ...
func (s *Store) Exists(key string) bool {
	pathKey := s.PathTransformFunc(key)
	info, err := os.Stat(path.Join(s.Root, pathKey.FullPath()))
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func (s *Store) writeStream(key string, r io.Reader) error {
	pathKey := s.PathTransformFunc(key)

	if err := os.MkdirAll(path.Join(s.Root, pathKey.PathName), os.ModePerm); err != nil {
		return err
	}

	pathAndFilename := path.Join(s.Root, pathKey.FullPath())
	f, err := os.Create(pathAndFilename)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, r)
	if err != nil {
		return err
	}

	return nil
}

// readStream returns a reader that can be used to read the
// contents of the file.
func (s *Store) readStream(key string) (io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	pathAndFilename := path.Join(s.Root, pathKey.FullPath())
	return os.Open(pathAndFilename)
}
