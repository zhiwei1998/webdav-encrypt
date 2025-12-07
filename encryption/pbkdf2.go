package encryption

import (
	"crypto/hmac"
	"crypto/sha256"
)

// pbkdf2 实现PBKDF2密钥派生函数（使用SHA256，与Python版本保持一致）
func pbkdf2(password string, salt []byte, iterations int, keyLength int) []byte {
	if keyLength <= 0 {
		return []byte{}
	}

	// 创建HMAC-SHA256实例作为PRF
	prf := func(key, data []byte) []byte {
		h := hmac.New(sha256.New, key)
		h.Write(data)
		return h.Sum(nil)
	}

	// SHA256输出长度为32字节
	hashLen := 32

	// 计算需要的块数
	l := (keyLength + hashLen - 1) / hashLen
	dk := make([]byte, keyLength)

	for i := 0; i < l; i++ {
		// U_1 = PRF(password, salt || INT(i))
		uintBuf := []byte{byte((i + 1) >> 24), byte((i + 1) >> 16), byte((i + 1) >> 8), byte(i + 1)}
		ui := prf([]byte(password), append(salt, uintBuf...))

		// T_i = U_1
		temp := make([]byte, len(ui))
		copy(temp, ui)

		// U_2 = PRF(password, U_1)
		// ...
		// U_iterations = PRF(password, U_{iterations-1})
		// T_i = U_1 XOR U_2 XOR ... XOR U_iterations
		for j := 1; j < iterations; j++ {
			ui = prf([]byte(password), ui)
			for k := range temp {
				temp[k] ^= ui[k]
			}
		}

		// 将T_i复制到最终密钥
		start := i * hashLen
		end := start + hashLen
		if end > keyLength {
			end = keyLength
		}
		copy(dk[start:end], temp[:end-start])
	}

	return dk
}