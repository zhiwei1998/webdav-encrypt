package encryption

import (
	"fmt"
)

func init() {
	// 注册AES-CTR加密器
	RegisterEncryptorFactoryFunc("aesctr", func(password string, fileSize int64, debugPrint DebugPrint) (Encryptor, error) {
		aesCtr, err := NewAesCTR(password, fileSize, debugPrint)
		if err != nil {
			return nil, err
		}
		debugPrint(fmt.Sprintf("@@AesCTR aesctr %d", fileSize))
		return aesCtr, nil
	})

	// 注册RC4-MD5加密器
	RegisterEncryptorFactoryFunc("rc4", func(password string, fileSize int64, debugPrint DebugPrint) (Encryptor, error) {
		rc4 := NewRc4Md5(password, fileSize, debugPrint)
		debugPrint(fmt.Sprintf("@@rc4 rc4 %d", fileSize))
		return rc4, nil
	})

	// 注册混合加密器
	RegisterEncryptorFactoryFunc("mix", func(password string, fileSize int64, debugPrint DebugPrint) (Encryptor, error) {
		mix := NewMixEnc(password, fileSize, debugPrint)
		debugPrint(fmt.Sprintf("@@mix mix %d", fileSize))
		return mix, nil
	})
}