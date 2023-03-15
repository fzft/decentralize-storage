package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"io"
)

func generateID() string {
	id := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, id); err != nil {
		panic(err)
	}
	return hex.EncodeToString(id)
}

func hashKey(key string) string {
	hash := md5.Sum([]byte(key))
	return hex.EncodeToString(hash[:])
}

func newEncryptionKey() []byte {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		panic(err)
	}
	return key
}

func copyStream(stream cipher.Stream, blockSize int, src io.Reader, dst io.Writer) (int, error) {
	var (
		buf = make([]byte, 32*1024)
		nw  = blockSize
	)

	for {
		n, err := src.Read(buf)
		if n > 0 {
			stream.XORKeyStream(buf, buf[:n])
			nn, err := dst.Write(buf[:n])
			if err != nil {
				return 0, err
			}
			nw += nn
		}

		if err == io.ErrUnexpectedEOF || err == io.EOF {
			break
		}

		if err != nil {
			return 0, err
		}
	}
	return nw, nil
}

func copyEncrypt(src io.Reader, dst io.Writer, key []byte) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	iv := make([]byte, block.BlockSize())
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return 0, err
	}

	// prepend iv to the encrypted data
	if _, err = dst.Write(iv); err != nil {
		return 0, err
	}

	return copyStream(cipher.NewCTR(block, iv), block.BlockSize(), src, dst)
}

// copyDecrypt decrypts the data from src and writes it to dst.
func copyDecrypt(src io.Reader, dst io.Writer, key []byte) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	// Read the iv from the beginning of the encrypted data
	iv := make([]byte, block.BlockSize())
	if _, err = io.ReadFull(src, iv); err != nil {
		return 0, err
	}

	return copyStream(cipher.NewCTR(block, iv), block.BlockSize(), src, dst)
}
