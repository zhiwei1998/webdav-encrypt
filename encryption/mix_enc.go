package encryption

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// MixEnc MixEnc加密算法实现
type MixEnc struct {
	password      string
	passwdOutward string
	encode        []byte
	decode        []byte
	debugPrint    DebugPrint
}

// NewMixEnc 创建新的MixEnc实例
func NewMixEnc(password string, fileSize int64, debugPrint DebugPrint) *MixEnc {
	me := &MixEnc{}
	me.password = password
	me.passwdOutward = password
	me.debugPrint = debugPrint

	// 说明是输入encode的秘钥，用于找回文件加解密
	if len(password) != 32 {
		// 使用PBKDF2派生密钥
		salt := []byte("MIX")
		key := pbkdf2(password, salt, 1000, 16)
		me.passwdOutward = hex.EncodeToString(key)
	}

	me.debugPrint(fmt.Sprintf("MixEnc.passwd_outward: %s", me.passwdOutward))

	// 创建编码表
	h := sha256.New()
	h.Write([]byte(me.passwdOutward))
	encodeBytes := h.Sum(nil)
	me.encode = make([]byte, len(encodeBytes))
	copy(me.encode, encodeBytes)

	length := len(me.encode)
	me.decode = make([]byte, length)
	decodeCheck := make(map[int]bool)

	for i := 0; i < length; i++ {
		enc := me.encode[i] ^ byte(i)
		// 处理冲突
		if !decodeCheck[int(enc)%length] {
			me.decode[enc%byte(length)] = me.encode[i] & 0xff
			decodeCheck[int(enc)%length] = true
		} else {
			for j := 0; j < length; j++ {
				if !decodeCheck[j] {
					me.encode[i] = (me.encode[i] & byte(length)) | byte(j^i)
					me.decode[j] = me.encode[i] & 0xff
					decodeCheck[j] = true
					break
				}
			}
		}
	}

	return me
}

// SetPosition 设置位置（MixEnc不需要实际设置位置）
func (me *MixEnc) SetPosition(position int64) {
	me.debugPrint("in the mix")
}

// EncryptData 加密数据
func (me *MixEnc) EncryptData(data []byte) []byte {
	result := make([]byte, len(data))
	for i := len(data) - 1; i >= 0; i-- {
		result[i] = data[i] ^ me.encode[data[i]%32]
	}
	return result
}

// DecryptData 解密数据
func (me *MixEnc) DecryptData(data []byte) []byte {
	result := make([]byte, len(data))
	for i := len(data) - 1; i >= 0; i-- {
		result[i] = data[i] ^ me.decode[data[i]%32]
	}
	return result
}