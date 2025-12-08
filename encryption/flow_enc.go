package encryption

import (
	"encoding/hex"
	"sync"
)

// FlowEnc 统一的加密接口类
type FlowEnc struct {
	fileSize      int64
	encryptFlow   Encryptor
	encryptType   string
	passwdOutward string
	debugPrint    DebugPrint
}

// 缓存派生密码
var cachePasswdOutward sync.Map

// getPassWdOutward 获取派生密码
func getPassWdOutward(password string, encryptType string) string {
	// 生成缓存键
	cacheKey := password + "|" + encryptType
	
	// 检查缓存
	if cached, ok := cachePasswdOutward.Load(cacheKey); ok {
		return cached.(string)
	}
	
	// 实现与Node.js一致的派生密码逻辑
	var result string
	if len(password) != 32 {
		// 根据不同的加密类型使用不同的盐值
		var salt []byte
		switch encryptType {
		case "mix":
			salt = []byte("MIX")
		case "rc4":
			salt = []byte("RC4")
		case "aesctr":
			salt = []byte("AES-CTR")
		default:
			salt = []byte("DEFAULT")
		}

		// 使用PBKDF2派生密钥，与Node.js保持一致的参数
		dk := pbkdf2(password, salt, 1000, 16)
		result = hex.EncodeToString(dk)
	} else {
		result = password
	}
	
	// 存入缓存
	cachePasswdOutward.Store(cacheKey, result)
	
	return result
}

// SetPosition 设置加密/解密位置
func (fe *FlowEnc) SetPosition(position int64) {
	fe.encryptFlow.SetPosition(position)
}

// EncryptData 加密数据
func (fe *FlowEnc) EncryptData(data []byte) []byte {
	return fe.encryptFlow.EncryptData(data)
}

// DecryptData 解密数据
func (fe *FlowEnc) DecryptData(data []byte) []byte {
	return fe.encryptFlow.DecryptData(data)
}