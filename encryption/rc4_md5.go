package encryption

import (
	"crypto/md5"
	"encoding/hex"
	"strconv"
)

// Rc4Md5 RC4-MD5加密算法实现
type Rc4Md5 struct {
	password      string
	passwdOutward string
	fileHexKey    string
	sizeSalt      string
	position      int64
	i             int
	j             int
	sbox          []int
	debugPrint    DebugPrint
}

// 每100万字节重置一次sbox
const SEGMENT_POSITION = 100 * 10000

// NewRc4Md5 创建新的Rc4Md5实例
func NewRc4Md5(password string, fileSize int64, debugPrint DebugPrint) *Rc4Md5 {
	rc := &Rc4Md5{}
	rc.password = password
	rc.sizeSalt = strconv.FormatInt(fileSize, 10)
	rc.debugPrint = debugPrint

	// 检查密码长度
	if len(password) != 32 {
		salt := []byte("RC4")
		key := pbkdf2(password, salt, 1000, 16)
		rc.passwdOutward = hex.EncodeToString(key)
	} else {
		rc.passwdOutward = password
	}

	// 添加文件大小作为盐
	passwdSalt := rc.passwdOutward + rc.sizeSalt
	h := md5.New()
	h.Write([]byte(passwdSalt))
	rc.fileHexKey = hex.EncodeToString(h.Sum(nil))

	// 初始化KSA
	rc.ResetKSA()

	return rc
}

// ResetKSA 重置KSA状态
func (rc *Rc4Md5) ResetKSA() {
	offset := (rc.position / SEGMENT_POSITION) * SEGMENT_POSITION
	buf := uint32ToBigEndian(uint32(offset))
	rc4Key, _ := hex.DecodeString(rc.fileHexKey)

	// 混合offset到密钥中
	j := len(rc4Key) - len(buf)
	for i := range buf {
		if j+i < len(rc4Key) {
			rc4Key[j+i] ^= buf[i]
		}
	}

	// 初始化KSA
	rc.initKSA(rc4Key)
}

// SetPosition 设置位置
func (rc *Rc4Md5) SetPosition(position int64) {
	rc.position = position
	rc.ResetKSA()
	// 执行PRGA到指定位置
	rc.prgaExecPosition(position % SEGMENT_POSITION)
}

// EncryptData 加密数据
func (rc *Rc4Md5) EncryptData(data []byte) []byte {
	return rc.prgaExecute(data)
}

// DecryptData 解密数据
func (rc *Rc4Md5) DecryptData(data []byte) []byte {
	// RC4加密和解密使用相同的方法
	return rc.prgaExecute(data)
}

// prgaExecute 执行PRGA算法
func (rc *Rc4Md5) prgaExecute(buffer []byte) []byte {
	S := make([]int, len(rc.sbox))
	copy(S, rc.sbox)
	i, j := rc.i, rc.j

	result := make([]byte, len(buffer))
	copy(result, buffer)

	for k := range result {
		i = (i + 1) % 256
		j = (j + S[i]) % 256
		// 交换S[i]和S[j]
		S[i], S[j] = S[j], S[i]
		result[k] ^= byte(S[(S[i]+S[j])%256])

		// 检查是否需要重置sbox
		if (rc.position+int64(k)+1)%SEGMENT_POSITION == 0 {
			// 重置状态
			tempPos := rc.position + int64(k) + 1
			rc.position = tempPos
			rc.ResetKSA()
			copy(S, rc.sbox)
			i, j = rc.i, rc.j
		}
	}

	// 保存状态
	rc.i, rc.j = i, j
	rc.position += int64(len(buffer))
	copy(rc.sbox, S)

	return result
}

// prgaExecPosition 执行PRGA算法到指定位置
func (rc *Rc4Md5) prgaExecPosition(plainLen int64) {
	S := make([]int, len(rc.sbox))
	copy(S, rc.sbox)
	i, j := rc.i, rc.j

	for _ = range make([]struct{}, plainLen) {
		i = (i + 1) % 256
		j = (j + S[i]) % 256
		// 交换S[i]和S[j]
		S[i], S[j] = S[j], S[i]
	}

	// 保存状态
	rc.i, rc.j = i, j
	copy(rc.sbox, S)
}

// initKSA 初始化KSA（密钥调度算法）
func (rc *Rc4Md5) initKSA(key []byte) {
	// 初始化S盒
	rc.sbox = make([]int, 256)
	for i := range rc.sbox {
		rc.sbox[i] = i
	}

	// 用种子密钥填充K表
	K := make([]byte, 256)
	keyLen := len(key)
	for i := range K {
		K[i] = key[i%keyLen]
	}

	// 置换S盒
	j := 0
	for i := range rc.sbox {
		j = (j + rc.sbox[i] + int(K[i])) % 256
		rc.sbox[i], rc.sbox[j] = rc.sbox[j], rc.sbox[i]
	}

	// 重置状态
	rc.i = 0
	rc.j = 0
}