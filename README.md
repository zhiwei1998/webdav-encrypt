# WebDAV加密代理（webdav-encrypt）

一个高性能WebDAV代理服务器，用于在上传时加密文件，在下载时解密文件。提供透明的加密/解密服务，并支持灵活的认证配置。

## 功能特性

- **透明加密/解密**：客户端无需特殊配置，使用标准WebDAV客户端即可
- **多种认证模式**：支持显式代理认证、同步后端认证或无认证
- **保持路径不变**：不加密文件名和目录结构，便于管理和查找
- **多种加密算法支持**：mix, rc4, aesctr（推荐）
- **流式处理**：支持大文件传输，内存占用低
- **性能优化**：连接池、超时控制和加密器缓存
- **灵活的配置管理**：支持命令行参数、环境变量和配置文件
- **配置文件生成**：自动生成默认配置文件，便于快速配置
- **调试模式**：`--debug`参数优先级高于配置文件的log_level设置
- **完整的WebDAV协议支持**：PUT, GET, DELETE, PROPFIND等所有常用方法
- **日志分级**：支持多种日志级别（trace, debug, info, warn, error, fatal），便于调试和监控

## 认证流程
客户端 → 代理认证（可选） → 加密/解密 → 后端认证（自动） → WebDAV服务器
## 快速开始

### 1. 构建

```bash
go build -o webdav-encrypt .
```

2. 运行

```bash
./webdav-encrypt \
  --listen :8080 \
  --backend "http://nextcloud.example.com/remote.php/webdav/" \
  --backend-user "your-username" \
  --backend-pass "your-password" \
  --password "your-encryption-key" \
  --algorithm aesctr
```

3. Docker运行

```bash
# 构建Docker镜像
docker build -t webdav-encrypt .

# 运行Docker容器
docker run -d \
  -p 8080:8080 \
  -e BACKEND_URL="http://nextcloud.example.com/remote.php/webdav/" \
  -e BACKEND_USER="your-username" \
  -e BACKEND_PASSWORD="your-password" \
  -e PASSWORD="your-encryption-key" \
  -e ALGORITHM="aesctr" \
  -e LOG_LEVEL="info" \
  webdav-encrypt
```

### 命令行参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--listen` | 监听地址 | `:8080` |
| `--backend` | 后端WebDAV服务器URL | 必填 |
| `--backend-user` | 后端WebDAV用户名 | 可选 |
| `--backend-pass` | 后端WebDAV密码 | 可选 |
| `--password` | 加密密码 | 必填 |
| `--algorithm` | 加密算法 (mix, rc4, aesctr) | `aesctr` |
| `--chunk-size` | 块大小(字节) | `8192` |
| `--debug` | 启用调试模式（优先级高于log_level） | `false` |
| `--log-level` | 日志级别 (trace, debug, info, warn, error, fatal) | `info` |
| `--auth-user` | 代理认证用户名 | `""` |
| `--auth-pass` | 代理认证密码 | `""` |
| `-c, --config` | 配置文件路径 (YAML格式) | 可选 |
| `--timeout` | HTTP请求超时时间(秒) | `30` |
| `--max-idle-conns` | 最大空闲连接数 | `100` |
| `--max-idle-conns-per-host` | 每个主机的最大空闲连接数 | `10` |
| `--idle-conn-timeout` | 空闲连接超时时间(秒) | `90` |
| `-h, --help` | 显示帮助信息 | - |

### 环境变量

所有命令行参数都可以通过环境变量设置：

| 环境变量 | 对应参数 |
|----------|----------|
| `LISTEN_ADDR` | `--listen` |
| `BACKEND_URL` | `--backend` |
| `BACKEND_USER` | `--backend-user` |
| `BACKEND_PASSWORD` | `--backend-pass` |
| `PASSWORD` | `--password` |
| `ALGORITHM` | `--algorithm` |
| `CHUNK_SIZE` | `--chunk-size` |
| `LOG_LEVEL` | `--log-level` |
| `DEBUG` | `--debug` |
| `AUTH_USER` | `--auth-user` |
| `AUTH_PASS` | `--auth-pass` |
| `TIMEOUT` | `--timeout` |
| `MAX_IDLE_CONNS` | `--max-idle-conns` |
| `MAX_IDLE_CONNS_PER_HOST` | `--max-idle-conns-per-host` |
| `IDLE_CONN_TIMEOUT` | `--idle-conn-timeout` |
| `CONFIG_FILE` | `--config` |

客户端连接

客户端连接到代理时，只需要代理的地址：

Windows

```powershell
# 映射网络驱动器
net use Z: http://localhost:8080

# 如果启用了代理认证
net use Z: http://localhost:8080 /user:proxy-user proxy-password
```

Linux/Mac

```bash
# 使用davfs2挂载
sudo mount -t davfs http://localhost:8080 /mnt/webdav

# 如果启用了代理认证，创建配置文件
echo "/mnt/webdav proxy-user proxy-password" > /etc/davfs2/secrets
```

使用rclone

```bash
# 如果代理启用了认证
rclone config create webdav-encrypt webdav \
  url=http://localhost:8080 \
  vendor=other \
  user=proxy-user \
  pass=proxy-password

# 如果代理未启用认证
rclone config create webdav-encrypt webdav \
  url=http://localhost:8080 \
  vendor=other

rclone mount webdav-encrypt: /mnt/webdav --vfs-cache-mode full
```

## 加密算法

1. **mix** - 自定义的简单加密算法
2. **rc4** - 基于RC4和MD5的流加密算法
3. **aesctr** - AES-CTR模式（推荐）

## 认证逻辑

代理支持三种认证模式：

1. **显式代理认证**：提供了`--auth-user`和`--auth-pass`（命令行或配置文件），使用这些凭据进行代理认证
2. **同步后端认证**：没有提供显式代理认证，但提供了`--backend-user`和`--backend-pass`（命令行或配置文件），同步使用后端凭据进行代理认证
3. **无认证**：其他情况，禁用代理认证

## 配置文件

### 生成默认配置文件

```bash
./webdav-encrypt -c config.yaml
```

生成默认配置文件后，程序会自动退出，需要修改配置文件后重新启动。

### 配置文件示例

```yaml
# WebDAV Proxy 配置文件

## 服务端设置
# 后端WebDAV服务器URL (必填项)
backend_url: "http://example.com/remote.php/webdav/"
# 后端WebDAV用户名 (可选)
backend_user: "your-backend-user"
# 后端WebDAV密码 (可选)
backend_pass: "your-backend-pass"

## 加密设置
# 加密算法 (可选，默认: aesctr)
algorithm: aesctr
# 加密密码 (必填)
password: "your-encryption-key"

## 代理端设置
# 监听地址 (默认: :8080)
listen_addr: ":8080"

# 代理认证用户名 (可选)
auth_user: ""
# 代理认证密码 (可选)
auth_pass: ""

## 日志设置
# 日志级别 (可选，默认: info)
log_level: "info"

## 性能设置
# 块大小(字节) (可选，默认: 8192)
chunk_size: 8192
# 请求超时时间 (可选，默认: 30s)
timeout: 30s
# 最大空闲连接数 (可选，默认: 100)
max_idle_conns: 100
# 每个主机的最大空闲连接数 (可选，默认: 10)
max_idle_conns_per_host: 10
# 空闲连接超时时间 (可选，默认: 90s)
idle_conn_timeout: 90s
```

测试

测试连接

```bash
# 测试代理是否正常运行
curl -I http://localhost:8080/

# 测试文件上传
curl -X PUT http://localhost:8080/test.txt \
  -H "Content-Type: text/plain" \
  --data "Hello World"

# 测试文件下载
curl http://localhost:8080/test.txt
```

自动化测试脚本

```bash
#!/bin/bash
echo "测试WebDAV代理..."

# 生成测试文件
echo "This is a test file for WebDAV proxy" > test.txt

# 上传文件
echo "1. 上传文件..."
curl -X PUT http://localhost:8080/test-encrypt.txt \
  --data-binary @test.txt \
  -H "Content-Type: text/plain"

# 下载文件
echo "2. 下载文件..."
curl http://localhost:8080/test-encrypt.txt -o downloaded.txt

# 比较文件
if cmp -s test.txt downloaded.txt; then
    echo "✅ 测试通过：文件加解密正常"
else
    echo "❌ 测试失败：文件内容不一致"
    diff test.txt downloaded.txt
fi

# 清理
rm -f test.txt downloaded.txt
```

## 安全建议

1. **后端密码安全**：确保后端WebDAV服务器的密码安全，避免泄露
2. **加密密码强度**：使用强密码（至少32字符），包含大小写字母、数字和特殊字符
3. **代理认证**：在生产环境中启用代理端认证，增加额外安全层
4. **HTTPS**：在代理和客户端之间、代理和后端之间都使用HTTPS
5. **防火墙**：使用防火墙限制代理服务器的访问，只允许必要的IP地址
6. **定期更换密码**：定期更换加密密码，建议每3-6个月更换一次
7. **日志管理**：根据需要设置合适的日志级别，避免泄露敏感信息
8. **更新软件**：定期更新代理软件，获取最新的安全修复和功能改进

故障排除

常见问题

1. 认证失败
   · 检查后端用户名和密码是否正确
   · 检查后端URL是否正确
   · 检查代理端认证配置（如果启用）
2. 文件无法加解密
   · 检查加密密码是否正确
   · 检查加密算法是否匹配
   · 检查文件是否有Content-Length头
3. 性能问题
   · 调整chunk-size参数
   · 检查网络连接
   · 启用调试模式查看详细信息

## 日志管理

### 日志级别

代理支持以下日志级别（从低到高）：

- `trace`：最详细的日志，包含所有调试信息
- `debug`：调试信息，用于开发和测试
- `info`：普通信息日志，默认级别
- `warn`：警告信息
- `error`：错误信息
- `fatal`：致命错误信息，记录后会退出程序

### 配置日志级别

1. **通过命令行参数**：

```bash
./webdav-encrypt --log-level debug ...
```

2. **通过环境变量**：

```bash
export LOG_LEVEL=info
./webdav-encrypt ...
```

3. **通过配置文件**：

```yaml
## 日志设置
# 日志级别 (可选，默认: info)
log_level: "warn"
# 启用调试模式 (优先级高于log_level，设置为true时强制使用debug级别)
debug: false
```

### 调试模式优先级

当同时使用 `--debug` 参数和配置文件中的 `log_level` 设置时，`--debug` 参数的优先级更高，会强制使用 DEBUG 级别日志。

```bash
# 即使配置文件中log_level设置为info，也会输出DEBUG级别的日志
./webdav-encrypt -c config.yaml --debug
```

许可证

MIT

```