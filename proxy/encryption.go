package proxy

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"webdav-proxy/encryption"
)

// proxyTransport 自定义传输层，处理加解密
type proxyTransport struct {
	handler       *ProxyHandler
	base          *http.Transport
	originalPaths map[*http.Request]string // 保存原始请求路径
}

// RoundTrip 执行HTTP请求
func (t *proxyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	originalPath := req.URL.Path // 保存原始请求路径
	t.handler.logger.Info("[TRANSPORT] 开始处理请求: %s %s", req.Method, originalPath)

	// 根据请求方法处理加解密
	switch req.Method {
	case http.MethodPut, http.MethodPost:
		// 上传文件 - 需要加密
		return t.handleUpload(req)
	case http.MethodGet, http.MethodHead:
		// 下载文件 - 需要解密
		return t.handleDownload(req)
	default:
		// 其他方法直接转发
		t.handler.logger.Debug("[TRANSPORT] 其他方法，直接转发: %s", req.Method)
		if t.base != nil {
			return t.base.RoundTrip(req)
		}
		return http.DefaultTransport.RoundTrip(req)
	}
}

// handleUpload 处理文件上传（加密）
func (t *proxyTransport) handleUpload(req *http.Request) (*http.Response, error) {
	t.handler.logger.Info("[UPLOAD] 开始处理文件上传: %s %s", req.Method, req.URL.Path)

	if req.Body == nil {
		t.handler.logger.Debug("[UPLOAD] 请求体为空，直接转发")
		return t.baseTransport().RoundTrip(req)
	}

	// 检查是否为文件（通过Content-Type或路径）
	contentType := req.Header.Get("Content-Type")
	if !isFileContentType(contentType) && !hasFileExtension(req.URL.Path) {
		// 不是文件类型，直接转发
		t.handler.logger.Debug("[UPLOAD] 非文件类型，跳过加密: %s, Content-Type: %s", req.URL.Path, contentType)
		return t.baseTransport().RoundTrip(req)
	}

	// 获取文件大小
	contentLength := req.ContentLength
	t.handler.logger.Debug("[UPLOAD] 文件大小: %d字节, 算法: %s, 块大小: %d", contentLength, t.handler.algorithm, t.handler.chunkSize)

	// 创建加密器
	enc, err := t.handler.getOrCreateEncryptor(contentLength)
	if err != nil {
		t.handler.logger.Error("[UPLOAD] 创建加密器失败，直接转发: %s, 错误: %v", req.URL.Path, err)
		return t.baseTransport().RoundTrip(req)
	}

	// 创建管道：读取原始数据 → 加密 → 发送到后端
	pr, pw := io.Pipe()

	// 启动goroutine处理加密
	go func() {
		defer pw.Close()
		defer req.Body.Close()

		// 创建缓冲区
		buf := make([]byte, t.handler.chunkSize)
		processedBytes := int64(0)

		t.handler.logger.Debug("[UPLOAD] 开始加密数据")
		for {
			// 从原始请求体读取
			n, err := req.Body.Read(buf)
			if n > 0 {
				processedBytes += int64(n)

				// 加密数据
				encrypted := enc.EncryptData(buf[:n])

				// 写入管道
				if _, err := pw.Write(encrypted); err != nil {
					t.handler.logger.Error("[UPLOAD] 写入管道失败: %v", err)
					return
				}

				// 每10MB记录一次进度
				if processedBytes%10*1024*1024 == 0 {
					t.handler.logger.Debug("[UPLOAD] 加密进度: %d字节已处理", processedBytes)
				}
			}

			if err != nil {
				if err != io.EOF {
					t.handler.logger.Error("[UPLOAD] 读取请求体失败: %v", err)
				} else {
					t.handler.logger.Debug("[UPLOAD] 加密完成，总处理字节数: %d", processedBytes)
				}
				return
			}
		}
	}()

	// 复制请求，替换请求体
	newReq := req.Clone(req.Context())
	newReq.Body = pr
	newReq.ContentLength = contentLength // 加密后大小不变

	// 发送请求到后端
	resp, err := t.baseTransport().RoundTrip(newReq)
	if err != nil {
		t.handler.logger.Error("[UPLOAD] 请求发送失败: %v", err)
		return nil, err
	}

	t.handler.logger.Debug("[UPLOAD] 上传完成，后端响应: %d %s", resp.StatusCode, resp.Status)
	return resp, nil
}

// decryptReader 流式解密读取器
type decryptReader struct {
	source     io.ReadCloser
	encryptor  encryption.Encryptor
	position   int64
	startPos   int64 // 解密起始位置
	endPos     int64 // 解密结束位置（包含）
	debugPrint func(string)
}

// Read 实现io.Reader接口，实现流式解密
func (dr *decryptReader) Read(p []byte) (int, error) {
	// 如果已经到达结束位置，返回EOF
	if dr.endPos > 0 && dr.position >= dr.endPos {
		return 0, io.EOF
	}

	// 限制读取的数据长度不超过剩余的范围
	maxRead := len(p)
	if dr.endPos > 0 {
		remaining := dr.endPos - dr.position + 1
		if remaining < int64(maxRead) {
			maxRead = int(remaining)
		}
	}

	// 从源Reader读取数据
	n, err := dr.source.Read(p[:maxRead])
	if err != nil && err != io.EOF {
		return n, err
	}

	if n > 0 {
		// 设置当前解密位置
		dr.encryptor.SetPosition(dr.position)

		// 解密数据
		decrypted := dr.encryptor.DecryptData(p[:n])

		// 将解密后的数据复制回p
		copy(p[:n], decrypted)

		// 更新位置
		dr.position += int64(n)

		//dr.debugPrint(fmt.Sprintf("[DOWNLOAD] 流式解密进度: %d字节已处理", dr.position))
	}

	return n, err
}

// Close 实现io.ReadCloser接口
func (dr *decryptReader) Close() error {
	return dr.source.Close()
}

// handleDownload 处理文件下载（解密）
func (t *proxyTransport) handleDownload(req *http.Request) (*http.Response, error) {
	t.handler.logger.Debug("[DOWNLOAD] 开始处理文件下载: %s %s", req.Method, req.URL.Path)

	// 先发送请求到后端
	resp, err := t.baseTransport().RoundTrip(req)
	if err != nil {
		t.handler.logger.Error("[DOWNLOAD] 请求发送失败: %v", err)
		return nil, err
	}

	// 拦截302重定向响应进行特殊处理
	if resp.StatusCode == http.StatusFound {
		location := resp.Header.Get("Location")
		t.handler.logger.Info("[DOWNLOAD] 拦截到302重定向响应: %s", location)

		// 创建新的请求来跟随重定向
		redirectReq, err := http.NewRequest(http.MethodGet, location, nil)
		if err != nil {
			t.handler.logger.Error("[DOWNLOAD] 创建重定向请求失败: %v", err)
			return resp, nil // 创建失败时返回原始响应
		}

		// 设置请求上下文
		redirectReq = redirectReq.WithContext(req.Context())

		// 复制原始请求的所有头信息，包括认证信息
		redirectReq.Header = make(http.Header)
		for k, vv := range req.Header {
			for _, v := range vv {
				redirectReq.Header.Add(k, v)
			}
		}

		// 如果重定向URL包含查询参数（可能是预签名URL），则移除Authorization头以避免冲突
		if redirectReq.URL.RawQuery != "" {
			redirectReq.Header.Del("Authorization")
			t.handler.logger.Debug("[DOWNLOAD] 重定向URL包含查询参数，已移除Authorization头")
		}

		// 确保重定向请求有正确的Host头
		redirectReq.Host = redirectReq.URL.Host

		// 使用独立的HTTP客户端发送请求，处理可能的重定向
		client := &http.Client{
			Transport: t.baseTransport(), // 复用现有的传输层配置
			// 自动跟随重定向，但限制最大重定向次数
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				// 确保每次重定向都携带认证信息
				for k, vv := range redirectReq.Header {
					for _, v := range vv {
						req.Header.Add(k, v)
					}
				}
				return nil
			},
		}

		// 保存原始请求信息，用于响应日志
		originalRequest := redirectReq
		redirectReq = redirectReq.Clone(req.Context())

		resp, err = client.Do(redirectReq)
		if resp != nil {
			// 设置响应的Request字段为原始请求，确保响应日志显示正确的路径
			resp.Request = originalRequest
		}
		if err != nil {
			t.handler.logger.Error("[DOWNLOAD] 重定向请求发送失败: %v", err)
			return nil, err
		}

		// 重定向后的响应需要重新检查是否需要解密
		t.handler.logger.Debug("[DOWNLOAD] 重定向响应状态码: %d, 头信息: %v", resp.StatusCode, resp.Header)
	}

	// 确保响应使用原始请求路径
	resp.Request.URL.Path = req.URL.Path

	// 检查响应状态码，支持200和206（部分内容）
	if (resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent) || resp.Body == nil {
		t.handler.logger.Debug("[DOWNLOAD] 非200/206响应或响应体为空，跳过解密: %d %s", resp.StatusCode, resp.Status)
		return resp, nil
	}

	// 检查是否为文件（通过Content-Type或Content-Disposition）
	contentType := resp.Header.Get("Content-Type")
	contentDisposition := resp.Header.Get("Content-Disposition")

	if !isFileContentType(contentType) && !hasFileExtension(req.URL.Path) &&
		!strings.Contains(contentDisposition, "attachment") {
		// 不是文件类型，直接返回
		t.handler.logger.Debug("[DOWNLOAD] 非文件类型，跳过解密: %s, Content-Type: %s", req.URL.Path, contentType)
		return resp, nil
	}

	// 获取文件大小
	contentLength := resp.ContentLength
	var fullFileSize int64 = contentLength

	// 解析原始请求的Range头
	var requestRange string
	if rangeHeader := req.Header.Get("Range"); rangeHeader != "" {
		requestRange = rangeHeader
	}

	// 检查是否为部分内容响应
	isPartial := resp.StatusCode == http.StatusPartialContent

	// 解析Content-Range头获取文件大小和范围信息
	startPos := int64(0)
	endPos := int64(0)

	if rangeHeader := resp.Header.Get("Content-Range"); rangeHeader != "" {
		// 解析Content-Range: bytes 0-999/1000
		parts := strings.Split(rangeHeader, "/")
		if len(parts) == 2 {
			if parts[1] != "*" {
				if size, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					fullFileSize = size
				}
			}

			// 解析范围部分：bytes start-end
			rangePart := strings.TrimPrefix(parts[0], "bytes ")
			rangeParts := strings.Split(rangePart, "-")
			if len(rangeParts) == 2 {
				if start, err := strconv.ParseInt(rangeParts[0], 10, 64); err == nil {
					startPos = start
				}
				if end, err := strconv.ParseInt(rangeParts[1], 10, 64); err == nil {
					endPos = end
				} else {
					// 如果end部分解析失败，使用文件末尾
					endPos = fullFileSize - 1
				}
			}
		}
	} else if requestRange != "" {
		// 如果没有Content-Range，但原始请求有Range头，解析Range头
		// 解析Range: bytes=0-999
		if strings.HasPrefix(requestRange, "bytes=") {
			rangeSpec := strings.TrimPrefix(requestRange, "bytes=")
			rangeParts := strings.Split(rangeSpec, "-")
			if len(rangeParts) == 2 {
				if start, err := strconv.ParseInt(rangeParts[0], 10, 64); err == nil {
					startPos = start
					isPartial = true // 标记为部分内容响应
				}
				if rangeParts[1] != "" {
					if end, err := strconv.ParseInt(rangeParts[1], 10, 64); err == nil {
						endPos = end
					} else {
						endPos = fullFileSize - 1
					}
				} else {
					// 如果end部分为空，使用文件末尾
					endPos = fullFileSize - 1
				}
			}
		}
	}

	// 设置默认endPos
	if endPos == 0 && fullFileSize > 0 {
		endPos = fullFileSize - 1
	}

	t.handler.logger.Info("[DOWNLOAD] 文件大小: %d字节, 范围: %d-%d, 算法: %s, 块大小: %d",
		fullFileSize, startPos, endPos, t.handler.algorithm, t.handler.chunkSize)

	// 创建解密器
	enc, err := t.handler.getOrCreateEncryptor(fullFileSize)
	if err != nil {
		t.handler.logger.Error("[DOWNLOAD] 创建解密器失败，直接返回原始响应: %s, 错误: %v", req.URL.Path, err)
		return resp, nil
	}

	// 设置解密起始位置
	enc.SetPosition(startPos)

	t.handler.logger.Debug("[DOWNLOAD] 开始流式解密，起始位置: %d", startPos)

	// 替换响应体为流式解密Reader
	resp.Body = &decryptReader{
		source:     resp.Body,
		encryptor:  enc,
		position:   startPos,
		startPos:   startPos,
		endPos:     endPos,
		debugPrint: func(msg string) { t.handler.logger.Debug(msg) },
	}

	// 设置Accept-Ranges头，表明支持字节范围请求
	resp.Header.Set("Accept-Ranges", "bytes")

	// 如果是部分内容响应或有起始位置，确保Content-Range头正确
	if isPartial || startPos > 0 {
		// 计算当前部分的长度
		currentLength := endPos - startPos + 1

		// 设置Content-Length为当前部分的长度
		resp.Header.Set("Content-Length", strconv.FormatInt(currentLength, 10))

		// 设置Content-Range头
		resp.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", startPos, endPos, fullFileSize))

		// 如果响应状态码不是206，设置为206
		if resp.StatusCode != http.StatusPartialContent {
			resp.StatusCode = http.StatusPartialContent
			resp.Status = "206 Partial Content"
		}
	} else {
		// 对于完整响应，设置正确的Content-Length
		resp.Header.Set("Content-Length", strconv.FormatInt(fullFileSize, 10))
	}

	// 设置缓存控制头
	resp.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	resp.Header.Set("Pragma", "no-cache")
	resp.Header.Set("Expires", "0")

	t.handler.logger.Debug("[DOWNLOAD] 下载设置完成，准备返回给客户端")
	return resp, nil
}

// baseTransport 获取基础传输层
func (t *proxyTransport) baseTransport() http.RoundTripper {
	if t.base != nil {
		return t.base
	}
	return http.DefaultTransport
}

// isHopByHopHeader 检查是否为Hop-by-hop头
func isHopByHopHeader(header string) bool {
	hopByHopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"TE",
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
	}

	// 将头名称转换为小写进行比较
	headerLower := strings.ToLower(header)
	for _, h := range hopByHopHeaders {
		if strings.ToLower(h) == headerLower {
			return true
		}
	}
	return false
}

// isFileContentType 检查是否为文件类型
func isFileContentType(contentType string) bool {
	// 常见的非文件类型
	nonFileTypes := []string{
		"text/html",
		"text/xml",
		"application/xml",
		"application/json",
		"text/css",
		"application/javascript",
		"application/x-www-form-urlencoded",
		"multipart/form-data",
	}

	for _, t := range nonFileTypes {
		if strings.HasPrefix(contentType, t) {
			return false
		}
	}

	// 常见的文件类型
	fileTypes := []string{
		"application/octet-stream",
		"application/pdf",
		"image/",
		"video/",
		"audio/",
		"text/plain",
		"application/msword",
		"application/vnd.",
		"application/zip",
		"application/x-rar-compressed",
		"application/x-tar",
		"application/x-gzip",
	}

	for _, t := range fileTypes {
		if strings.HasPrefix(contentType, t) {
			return true
		}
	}

	// 默认认为是文件
	return true
}

// hasFileExtension 检查路径是否有文件扩展名
func hasFileExtension(path string) bool {
	// 检查常见的文件扩展名
	extensions := []string{
		".pdf", ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".svg", ".webp",
		".mp4", ".avi", ".mov", ".wmv", ".flv", ".mkv", ".webm",
		".mp3", ".wav", ".flac", ".aac", ".ogg", ".m4a",
		".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".zip", ".rar", ".7z", ".tar", ".gz", ".bz2",
		".txt", ".log", ".csv", ".json", ".xml", ".yaml", ".yml",
		".exe", ".dmg", ".pkg", ".deb", ".rpm",
	}

	lowerPath := strings.ToLower(path)
	for _, ext := range extensions {
		if strings.HasSuffix(lowerPath, ext) {
			return true
		}
	}

	return false
}
