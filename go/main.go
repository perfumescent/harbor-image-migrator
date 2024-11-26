package main

import (
	"bufio"
	"bytes"
	"dockerImageMigrator/harbor"
	"dockerImageMigrator/log"
	"dockerImageMigrator/ssh"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// 修改 main 函数来使用新的结构体
func deploy(localFile string) {
	log.Info(">>>>>> 开始部署", localFile)
	dest := harbor.HarborConfig{
		HarborApi:  "https://10.100.100.21:10080",
		HarborHost: "dockerhub.cestc.local",
		Username:   "admin",
		Password:   "Harbor12345",
	}

	// 读取文件内容
	yamlFile, err := os.ReadFile(localFile)
	if err != nil {
		log.Errorf("读取yaml文件失败: %v", err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(yamlFile))
	var finalDocs []map[string]interface{}

	for {
		var parsedDoc map[string]interface{}
		err := decoder.Decode(&parsedDoc)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Errorf("解码YAML文档失败: %v", err)
		}

		finalDocs = append(finalDocs, parsedDoc)

		// 只处理 Deployment 类型的文档
		kind, ok := parsedDoc["kind"].(string)
		if !ok || kind != "Deployment" {
			continue
		}

		// 获取 containers 部分
		spec, ok := parsedDoc["spec"].(map[string]interface{})
		if !ok {
			continue
		}
		template, ok := spec["template"].(map[string]interface{})
		if !ok {
			continue
		}
		podSpec, ok := template["spec"].(map[string]interface{})
		if !ok {
			continue
		}
		containers, ok := podSpec["containers"].([]interface{})
		if !ok {
			continue
		}

		// 处理每个容器的镜像
		for j, container := range containers {
			containerMap, ok := container.(map[string]interface{})
			if !ok {
				continue
			}
			imageRaw, exists := containerMap["image"].(string)
			if !exists {
				continue
			}

			// 处理镜像地址
			if !strings.HasPrefix(imageRaw, "http://") && !strings.HasPrefix(imageRaw, "https://") {
				imageRaw = "https://" + imageRaw
			}

			colonIndex := strings.LastIndex(imageRaw, ":")
			if colonIndex == -1 {
				log.Errorf("镜像地址 %s 缺少标签部分", imageRaw)
			}
			imageURLStr := imageRaw[:colonIndex]
			tag := imageRaw[colonIndex+1:]

			imageURL, err := url.ParseRequestURI(imageURLStr)
			if err != nil {
				log.Errorf("解析URL失败: %v", err)
			}
			registry := imageURL.Scheme + "://" + imageURL.Host
			path := imageURL.Path

			log.Infof("开始处理 registry: %v, path: %v, tag: %v", registry, path, tag)

			exist, err := harbor.CheckImageExists(dest.HarborApi, path, tag, dest.Username, dest.Password)
			if err != nil {
				log.Errorf("检查镜像 %s 失败: %v", imageRaw, err)
			}

			dest.ImagePath = path
			dest.ImageTag = tag

			if !exist {
				log.Infof("检测到 %v 里不存在，现在开始推送镜像", dest.HarborApi)
				source := harbor.HarborConfig{
					HarborApi:  registry,
					HarborHost: registry,
					Username:   "cmq",
					Password:   "Cmq12345",
					ImagePath:  path,
					ImageTag:   tag,
				}
				if err := harbor.MigrateImage(source, dest); err != nil {
					log.Errorf("[ERROR] 镜像 %v 迁移失败: %v", imageRaw, err)
				}
			} else {
				log.Infof("检测到镜像已存在，跳过")
			}

			// 修改镜像地址
			newImage := fmt.Sprintf("%s%s:%s", dest.HarborHost, dest.ImagePath, dest.ImageTag)
			log.Infof("将配置文件中镜像地址%s修改为: %v", registry, newImage)
			containerMap["image"] = newImage
			containers[j] = containerMap
		}

		// 更新文档中的容器信息
		podSpec["containers"] = containers
		template["spec"] = podSpec
		spec["template"] = template
		parsedDoc["spec"] = spec
	}

	// 将修改后的文档重新组合成YAML字符串
	var yamlBuilder strings.Builder
	encoder := yaml.NewEncoder(&yamlBuilder)
	defer encoder.Close()
	for _, doc := range finalDocs {
		if err := encoder.Encode(doc); err != nil {
			log.Errorf("序列化YAML文档失败: %v", err)
		}
	}

	yamlString := yamlBuilder.String()

	// SSH相关操作
	config := ssh.SSHConfig{
		Host:      "10.100.100.21",
		Port:      22,
		Username:  "root",
		Password:  "Cestc@2024",
		RemoteDir: "/opt/baseline/tmp/",
	}
	// 创建SSH客户端
	client, err := ssh.NewSSHClient(&config)
	if err != nil {
		log.Errorf("创建SSH客户端失败: %v", err)
	}
	// 连接到远程服务器
	if err := client.Connect(); err != nil {
		log.Errorf("连接到远程服务器失败: %v", err)
	}
	defer client.Close()

	// 1. 获取文件名（带后缀）
	fileName := filepath.Base(localFile) // "backend-portal-front.yaml"
	// 2. 分离文件名和后缀
	name := strings.TrimSuffix(fileName, filepath.Ext(fileName)) // "backend-portal-front"
	ext := filepath.Ext(fileName)                                // ".yaml"

	// 3. 在文件名和后缀之间添加内容
	remotePath := fmt.Sprintf("%s%s_%s%s", config.RemoteDir, name, time.Now().Format("20060102150405"), ext)

	log.Infof("正在传输文件: %s\n", fileName)
	if err := client.WriteStringToFile(yamlString, remotePath); err != nil {
		log.Infof("文件传输失败 %s: %v\n", fileName, err)
	} else {
		log.Infof("成功传输文件 %s 到 %s\n", fileName, remotePath)
	}

	output, err := client.ExecuteCommand("kubectl apply -f " + remotePath)
	if err != nil {
		log.Infof("执行命令失败: %v\n", err)
	} else {
		log.Infof("命令输出:\n%s\n", output)
	}

	fmt.Printf("👌 %s 部署结束\n\n\n", localFile)
}

func main() {
	// 初始化日志
	log.Init()

	const promptMessage = ">>> 请拖拽k8s yaml文件进来"

	fmt.Println(promptMessage)
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		input := scanner.Text()

		if strings.ToLower(input) == "exit" {
			break
		}

		// 处理所有输入的文件
		for _, path := range strings.Fields(input) {
			deploy(path)
		}

		fmt.Println(promptMessage)
	}

	if err := scanner.Err(); err != nil {
		log.Infof("读取输入错误: %v\n", err)
	}
}
