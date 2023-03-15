package main

import (
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

	// ID is the unique ID of the store.
	ID string
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

	if len(opts.ID) == 0 {
		opts.ID = generateID()
	}

	return &Store{
		StoreOpts: opts,
	}
}

// Write writes the contents of the reader to the file.
func (s *Store) Write(key string, r io.Reader) (int64, error) {
	return s.writeStream(key, r)
}

// Read returns the contents of the file.
func (s *Store) Read(key string) (io.Reader, error) {
	return s.readStream(key)
}

// Delete ...
func (s *Store) Delete(key string) error {
	pathKey := s.PathTransformFunc(key)
	return os.RemoveAll(path.Join(s.Root, s.ID, pathKey.FirstPathName()))
}

// Clear ...
func (s *Store) Clear() error {
	return os.RemoveAll(s.Root)
}

// Exists ...
func (s *Store) Exists(key string) bool {
	pathKey := s.PathTransformFunc(key)
	info, err := os.Stat(path.Join(s.Root, s.ID, pathKey.FullPath()))
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// FileSize ...
func (s *Store) FileSize(key string) (int64, error) {
	pathKey := s.PathTransformFunc(key)
	info, err := os.Stat(path.Join(s.Root, s.ID, pathKey.FullPath()))
	if err != nil {
		return 0, err
	}
	// Get the file size in bytes
	fileSize := info.Size()
	return fileSize, nil
}

// writeDecryptStream writes the contents of the reader to the file.
func (s *Store) writeDecryptStream(encKey []byte, key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(key)
	n, err := copyDecrypt(r, f, encKey)
	if err != nil {
		return 0, err
	}

	return int64(n), nil
}

// openFileForWriting opens a file for writing.
func (s *Store) openFileForWriting(key string) (*os.File, error) {
	pathKey := s.PathTransformFunc(key)

	if err := os.MkdirAll(path.Join(s.Root, s.ID, pathKey.PathName), os.ModePerm); err != nil {
		return nil, err
	}

	pathAndFilename := path.Join(s.Root, s.ID, pathKey.FullPath())
	return os.Create(pathAndFilename)
}

func (s *Store) writeStream(key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(key)
	if err != nil {
		return 0, err
	}
	return io.Copy(f, r)
}

// readStream returns a reader that can be used to read the
// contents of the file.
func (s *Store) readStream(key string) (io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	pathAndFilename := path.Join(s.Root, s.ID, pathKey.FullPath())
	return os.Open(pathAndFilename)
}
