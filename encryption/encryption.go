package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/binary"
	"strconv"
)

// Decrypt a segment using AES-128 CBC
func DecryptSegment(segment []byte, key string, segmentId string) ([]byte, error) {
	//convert the key to byte array from base64
	bytes, err := base64.URLEncoding.DecodeString(key)

	if err != nil {
		return nil, err
	}

	iv, err := strconv.ParseInt(segmentId, 10, 64)
	defaultIV := defaultIV(uint64(iv))
	if err != nil {
		return nil, err
	}
	return decryptAES128(segment, bytes, defaultIV)
}

func decryptAES128(crypted, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, iv[:blockSize])
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = pkcs5UnPadding(origData)
	return origData, nil
}

func pkcs5UnPadding(origData []byte) []byte {
	length := len(origData)
	unPadding := int(origData[length-1])
	return origData[:(length - unPadding)]
}
func defaultIV(seqID uint64) []byte {
	buf := make([]byte, 16)
	binary.BigEndian.PutUint64(buf[8:], seqID)
	return buf
}
