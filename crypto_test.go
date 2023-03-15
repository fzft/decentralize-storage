package main

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCopyEncrypt(t *testing.T) {
	payload := "hello world"
	src := bytes.NewReader([]byte(payload))
	dst := new(bytes.Buffer)
	key := newEncryptionKey()

	_, err := copyEncrypt(src, dst, key)
	if err != nil {
		t.Errorf("copyEncrypt() error = %v, wantErr %v", err, false)
	}

	out := new(bytes.Buffer)
	if _, err := copyDecrypt(dst, out, key); err != nil {
		t.Errorf("copyDecrypt() error = %v, wantErr %v", err, false)
	}

	assert.Equal(t, payload, out.String())
}
