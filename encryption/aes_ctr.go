package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/hex"
	"strconv"
)

// bigEndianToUint32 将大端字节序转换为uint32
func bigEndianToUint32(b []byte) uint32 {
	if len(b) < 4 {
		return 0
	}
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

// uint32ToBigEndian 将uint32转换为大端字节序
func uint32ToBigEndian(n uint32) []byte {
	return []byte{
		byte((n >> 24) & 0xFF),
		byte((n >> 16) & 0xFF),
		byte((n >> 8) & 0xFF),
		byte(n & 0xFF),
	}
}

// AesCTR AES-CTR加密算法实现
type AesCTR struct {
	password      string
	passwdOutward string
	key           []byte
	iv            []byte
	sourceIV      []byte
	block         cipher.Block
	stream        cipher.Stream
	debugPrint    DebugPrint
}

// NewAesCTR 创建新的AesCTR实例
func NewAesCTR(password string, fileSize int64, debugPrint DebugPrint) (*AesCTR, error) {
	ac := &AesCTR{}
	ac.password = password
	ac.debugPrint = debugPrint

	// 检查密码长度，如果不是32位，进行派生
	if len(password) != 32 {
		// 确保使用与Node.js版本完全相同的盐值和迭代次数
		salt := []byte("AES-CTR")
		key := pbkdf2(password, salt, 1000, 16)
		ac.passwdOutward = hex.EncodeToString(key)
	} else {
		ac.passwdOutward = password
	}

	// 创建文件AES-CTR密钥
	passwdSalt := ac.passwdOutward + strconv.FormatInt(fileSize, 10)
	h := md5.New()
	h.Write([]byte(passwdSalt))
	ac.key = h.Sum(nil)

	ivHash := md5.New()
	ivHash.Write([]byte(strconv.FormatInt(fileSize, 10)))
	ac.iv = ivHash.Sum(nil)
	ac.sourceIV = make([]byte, len(ac.iv))
	copy(ac.sourceIV, ac.iv)

	// 创建加密器
	block, err := aes.NewCipher(ac.key)
	if err != nil {
		return nil, err
	}

	ac.block = block
	ac.stream = cipher.NewCTR(block, ac.iv)

	return ac, nil
}

// SetPosition 设置加密/解密位置
func (ac *AesCTR) SetPosition(position int64) {
	// 重置IV
	ac.iv = make([]byte, len(ac.sourceIV))
	copy(ac.iv, ac.sourceIV)

	increment := position / 16
	ac.incrementIV(increment)

	// 创建新的流
	ac.stream = cipher.NewCTR(ac.block, ac.iv)

	// 跳过偏移量
	offset := position % 16
	if offset > 0 {
		dummy := make([]byte, offset)
		ac.stream.XORKeyStream(dummy, dummy)
	}
}

// incrementIV 增加IV值
func (ac *AesCTR) incrementIV(increment int64) {
	const MAX_UINT32 = 0xffffffff
	incrementBig := increment / MAX_UINT32
	incrementLittle := (increment % MAX_UINT32) - incrementBig

	// 将128位IV分成4个32位数字
	overflow := int64(0)
	for idx := 0; idx < 4; idx++ {
		// 从高位到低位读取
		pos := 12 - idx*4
		num := int64(bigEndianToUint32(ac.iv[pos : pos+4]))

		inc := overflow
		if idx == 0 {
			inc += incrementLittle
		}
		if idx == 1 {
			inc += incrementBig
		}

		num += inc
		numBig := num / MAX_UINT32
		numLittle := (num % MAX_UINT32) - numBig
		overflow = numBig

		// 写回IV
		copy(ac.iv[pos:pos+4], uint32ToBigEndian(uint32(numLittle)))
	}
}

// EncryptData 加密数据
func (ac *AesCTR) EncryptData(data []byte) []byte {
	result := make([]byte, len(data))
	ac.stream.XORKeyStream(result, data)
	return result
}

// DecryptData 解密数据
func (ac *AesCTR) DecryptData(data []byte) []byte {
	// AES-CTR模式下，加密和解密使用相同的操作
	return ac.EncryptData(data)
}