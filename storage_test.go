package main

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestSore(t *testing.T) {
	s := newStore(t)
	defer teardown(t, s)

	key := "hello"
	data := []byte("world")

	if _, err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}

	r, err := s.readStream(key)
	if err != nil {
		t.Fatal(err)
	}

	buf, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, data, buf)
}

func TestDelete(t *testing.T) {
	s := newStore(t)
	defer teardown(t, s)
	key := "hello"
	s.Delete(key)

	assert.True(t, !s.Exists(key))
}

func newStore(t *testing.T) *main.Store {
	opts := main.StoreOpts{
		PathTransformFunc: main.CASPathTransformFunc,
	}
	s := main.NewStore(opts)
	return s
}

func teardown(t *testing.T, s *main.Store) {
	s.Clear()
}
