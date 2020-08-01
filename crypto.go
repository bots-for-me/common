package common

import (
	"crypto/md5"
	"encoding/hex"
	"math/rand"
)

func MD5HashBytes(obj interface{}) []byte {
	var v []byte
	switch data := obj.(type) {
	case []byte:
		v = data
	case string:
		v = []byte(data)
	case nil:
		v = make([]byte, md5.Size*2)
		if _, err := rand.Read(v); err != nil {
			Log.Error("Md5HashBytes/Read: %v", err)
			return nil
		}
	default:
		Log.Error("unsupported type: %T", obj)
		return nil
	}
	h := md5.New()
	if _, err := h.Write(v); err != nil {
		Log.Error("Md5HashBytes/Write: %v", err)
		return nil
	}
	return h.Sum(nil)
}

func MD5HashString(obj interface{}) string {
	return hex.EncodeToString(MD5HashBytes(obj))
}
