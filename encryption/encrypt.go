package encryption

import "fmt"

// Encryptor 加密接口
type Encryptor interface {
	SetPosition(position int64)
	EncryptData(data []byte) []byte
	DecryptData(data []byte) []byte
}

// EncryptorFactory 加密器工厂接口
type EncryptorFactory interface {
	Create(password string, fileSize int64, debugPrint DebugPrint) (Encryptor, error)
}

// EncryptorFactoryFunc 加密器工厂函数类型
type EncryptorFactoryFunc func(password string, fileSize int64, debugPrint DebugPrint) (Encryptor, error)

// Create 实现EncryptorFactory接口
func (f EncryptorFactoryFunc) Create(password string, fileSize int64, debugPrint DebugPrint) (Encryptor, error) {
	return f(password, fileSize, debugPrint)
}

// DebugPrint 调试打印函数类型
type DebugPrint func(string)

// 加密器工厂注册表
var encryptorFactories = make(map[string]EncryptorFactory)

// RegisterEncryptorFactory 注册加密器工厂
func RegisterEncryptorFactory(encryptType string, factory EncryptorFactory) {
	encryptorFactories[encryptType] = factory
}

// RegisterEncryptorFactoryFunc 通过函数注册加密器工厂
func RegisterEncryptorFactoryFunc(encryptType string, factoryFunc func(password string, fileSize int64, debugPrint DebugPrint) (Encryptor, error)) {
	encryptorFactories[encryptType] = EncryptorFactoryFunc(factoryFunc)
}

// NewEncryptor 创建加密器
func NewEncryptor(password, encryptType string, fileSize int64, debugPrint DebugPrint) (Encryptor, error) {
	factory, ok := encryptorFactories[encryptType]
	if !ok {
		return nil, fmt.Errorf("unknown encrypt type: %s", encryptType)
	}
	return factory.Create(password, fileSize, debugPrint)
}

// NewFlowEnc 创建FlowEnc加密器（兼容旧接口）
func NewFlowEnc(password, encryptType string, fileSize int64, debugPrint DebugPrint) (Encryptor, error) {
	return NewEncryptor(password, encryptType, fileSize, debugPrint)
}
