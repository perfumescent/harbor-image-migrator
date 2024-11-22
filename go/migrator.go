package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// 全局配置
const (
	VerifySSL     = false                // 如果使用受信任的 SSL 证书，请设置为 true
	ChunkSize     = 8192                 // 文件块大小，用于上传和下载
	MaxWorkers    = 8                    // 并发线程数
	OutputDir     = "./downloaded_files" // 下载文件保存路径
	sourceProject = "digital/dev/portal-front"
	imageTag      = "0e0bd5ce7" // 源镜像标签
	NewImageTag   = "v1.0.1"    // 根据需要修改
	destProject   = "digital/dev/portal-front-test"
)

// 用户输入信息
var (
	SourceHarbor   = "https://image.cestc.cn"
	SourceUsername = "cmq"
	SourcePassword = "Cmq12345"

	DestHarbor   = "https://image.cestc.cn"
	DestUsername = "cmq"
	DestPassword = "Cmq12345"
)

// Manifest 定义 Docker 镜像的 manifest 结构，添加了顶层的 MediaType 字段
type Manifest struct {
	MediaType     string `json:"mediaType"`
	SchemaVersion int    `json:"schemaVersion"`
	Config        struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"size"`
		Digest    string `json:"digest"`
	} `json:"config"`
	Layers []struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"size"`
		Digest    string `json:"digest"`
	} `json:"layers"`
}

// 创建 HTTP 客户端，配置 TLS 验证
func createHTTPClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !VerifySSL},
	}
	return &http.Client{
		Timeout:   time.Minute * 10,
		Transport: tr,
	}
}

// 检查 Blob 是否已存在于 Harbor
func blobExists(harborURL, projectPath, digest string, client *http.Client, auth string) (bool, error) {
	url := fmt.Sprintf("%s/v2/%s/blobs/%s", harborURL, projectPath, digest)
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false, fmt.Errorf("创建 HEAD 请求失败: %v", err)
	}
	req.Header.Set("Authorization", auth)

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("发送 HEAD 请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return false, nil
	} else {
		return false, fmt.Errorf("HEAD 请求返回状态码 %d", resp.StatusCode)
	}
}

// 修改后的下载单个 blob，返回一个 io.Reader
func downloadBlobStream(harborURL, projectPath, digest string, auth string, client *http.Client) (io.ReadCloser, error) {
	uploadURL := fmt.Sprintf("%s/v2/%s/blobs/%s", harborURL, projectPath, digest)
	req, err := http.NewRequest("GET", uploadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建 GET 请求失败: %v", err)
	}
	req.Header.Set("Authorization", auth)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送 GET 请求失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("下载 blob 失败: 状态码 %d - %s", resp.StatusCode, resp.Status)
	}

	return resp.Body, nil
}

// 修改后的上传单个 blob，接受一个 io.Reader
func uploadBlobStreamToHarbor(harborURL, projectPath, digest, fileType string, auth string, client *http.Client, reader io.Reader) error {
	// 检查 Blob 是否已存在
	exists, err := blobExists(harborURL, projectPath, digest, client, auth)
	if err != nil {
		return fmt.Errorf("检查 blob 存在性失败: %v", err)
	}
	if exists {
		log.Printf("[INFO] %s 已存在，跳过上传。", fileType)
		return nil
	}

	// 创建上传会话
	uploadURL := fmt.Sprintf("%s/v2/%s/blobs/uploads/", harborURL, projectPath)
	req, err := http.NewRequest("POST", uploadURL, nil)
	if err != nil {
		return fmt.Errorf("创建上传会话请求失败: %v", err)
	}
	req.Header.Set("Authorization", auth)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送上传会话请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("创建上传会话失败: 状态码 %d - %s", resp.StatusCode, resp.Status)
	}

	location := resp.Header.Get("Location")
	if location == "" {
		return fmt.Errorf("上传会话响应缺少 Location 头")
	}
	log.Printf("[INFO] 创建上传会话成功：%s - %s", fileType, location)

	// 上传 blob 使用 PUT 方法
	uploadURLWithDigest := fmt.Sprintf("%s&digest=%s", location, digest)
	putReq, err := http.NewRequest("PUT", uploadURLWithDigest, reader)
	if err != nil {
		return fmt.Errorf("创建 PUT 请求失败: %v", err)
	}
	putReq.Header.Set("Authorization", auth)

	putResp, err := client.Do(putReq)
	if err != nil {
		return fmt.Errorf("发送 PUT 请求失败: %v", err)
	}
	defer putResp.Body.Close()

	if putResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(putResp.Body)
		return fmt.Errorf("上传失败: 状态码 %d - %s", putResp.StatusCode, string(body))
	}

	log.Printf("[INFO] %s 上传成功", fileType)
	return nil
}

// 注册 manifest 到目标 Harbor
func registerManifest(harborURL, projectPath, manifestPath, auth, imageTag string, client *http.Client) error {
	// 读取 manifest 文件
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("读取 manifest 文件失败: %v", err)
	}

	url := fmt.Sprintf("%s/v2/%s/manifests/%s", harborURL, projectPath, imageTag)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("创建 manifest 注册请求失败: %v", err)
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")

	log.Printf("[INFO] 注册 manifest: %s", url)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送 manifest 注册请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		log.Println("[INFO] Manifest 注册成功")
		return nil
	} else {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Manifest 注册失败: 状态码 %d - %s", resp.StatusCode, string(body))
	}
}

// 修改 main 函数，直接使用流式传输
func main() {
	// 创建输出目录（仅用于存储 manifest）
	if _, err := os.Stat(OutputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(OutputDir, 0755); err != nil {
			log.Fatalf("[ERROR] 创建输出目录失败: %v", err)
		}
	}

	// 创建 HTTP 客户端
	client := createHTTPClient()
	sourceAuth := "Basic " + basicAuth(SourceUsername, SourcePassword)
	destAuth := "Basic " + basicAuth(DestUsername, DestPassword)

	// Step 1: 获取源镜像的 manifest
	manifestURL := fmt.Sprintf("%s/v2/%s/manifests/%s", SourceHarbor, sourceProject, imageTag)
	log.Printf("[INFO] 获取 manifest: %s", manifestURL)
	req, err := http.NewRequest("GET", manifestURL, nil)
	if err != nil {
		log.Fatalf("[ERROR] 创建获取 manifest 请求失败: %v", err)
	}
	req.Header.Set("Authorization", sourceAuth)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("[ERROR] 发送获取 manifest 请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("[ERROR] 获取 manifest 失败: 状态码 %d - %s", resp.StatusCode, string(body))
	}

	var manifest Manifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		log.Fatalf("[ERROR] 解析 manifest 失败: %v", err)
	}

	// 保存 manifest 文件（因为后续需要修改它）
	manifestPath := filepath.Join(OutputDir, "manifest.json")
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		log.Fatalf("[ERROR] 序列化 manifest 数据失败: %v", err)
	}
	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		log.Fatalf("[ERROR] 保存 manifest 文件失败: %v", err)
	}

	// Step 2: 并发迁移所有 blobs
	var wg sync.WaitGroup
	errChan := make(chan error, len(manifest.Layers)+1) // +1 for config
	sem := make(chan struct{}, MaxWorkers)

	// 处理配置文件
	wg.Add(1)
	go func() {
		defer wg.Done()
		sem <- struct{}{}
		defer func() { <-sem }()

		reader, err := downloadBlobStream(SourceHarbor, sourceProject, manifest.Config.Digest, sourceAuth, client)
		if err != nil {
			errChan <- fmt.Errorf("下载 config.json 失败: %v", err)
			return
		}
		defer reader.Close()

		err = uploadBlobStreamToHarbor(DestHarbor, destProject, manifest.Config.Digest, "config.json", destAuth, client, reader)
		if err != nil {
			errChan <- fmt.Errorf("上传 config.json 失败: %v", err)
		}
	}()

	// 处理层文件
	for i, layer := range manifest.Layers {
		wg.Add(1)
		go func(layerIndex int, layerInfo struct {
			MediaType string `json:"mediaType"`
			Size      int    `json:"size"`
			Digest    string `json:"digest"`
		}) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			fileType := fmt.Sprintf("layer%d.tar.gz", layerIndex+1)
			reader, err := downloadBlobStream(SourceHarbor, sourceProject, layerInfo.Digest, sourceAuth, client)
			if err != nil {
				errChan <- fmt.Errorf("下载 %s 失败: %v", fileType, err)
				return
			}
			defer reader.Close()

			err = uploadBlobStreamToHarbor(DestHarbor, destProject, layerInfo.Digest, fileType, destAuth, client, reader)
			if err != nil {
				errChan <- fmt.Errorf("上传 %s 失败: %v", fileType, err)
			}
		}(i, layer)
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		for err := range errChan {
			log.Printf("[ERROR] %v", err)
		}
		log.Fatal("[ERROR] 部分 blob 迁移失败")
	}

	// Step 3: 注册 manifest
	if err := registerManifest(DestHarbor, destProject, manifestPath, destAuth, NewImageTag, client); err != nil {
		log.Fatalf("[ERROR] 注册 manifest 失败: %v", err)
	}

	// 打印新镜像地址
	newImageAddress := fmt.Sprintf("%s/v2/%s/manifests/%s", DestHarbor, destProject, NewImageTag)
	log.Printf("[INFO] 镜像迁移完成！新镜像地址：%s", newImageAddress)

	// 清理 manifest 文件
	if err := os.RemoveAll(OutputDir); err != nil {
		log.Printf("[WARN] 清理临时文件失败: %v", err)
	}
}

// 创建基本认证字符串
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
