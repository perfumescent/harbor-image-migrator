package main

import (
	"bufio"
	"dockerImageMigrator/ssh"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// 创建SSH配置
	config := ssh.SSHConfig{
		Host:      "43.155.183.120",
		Port:      22,
		Username:  "root",
		Password:  "YBW522866ybw!",
		RemoteDir: "/root/tmp/",
	}

	// 创建SSH客户端
	client, err := ssh.NewSSHClient(&config)
	if err != nil {
		log.Fatalf("Failed to create SSH client: %v", err)
	}

	// 连接到远程服务器
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	fmt.Println("SSH连接已建立。您可以：")
	fmt.Println("1. 拖拽文件到窗口来传输文件")
	fmt.Println("2. 输入 'cmd:' 开头的命令来执行shell命令")
	fmt.Println("3. 输入 'exit' 退出程序")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := scanner.Text()

		// 检查是否退出
		if strings.ToLower(input) == "exit" {
			break
		}

		// 检查是否是shell命令
		if strings.HasPrefix(input, "cmd:") {
			command := strings.TrimPrefix(input, "cmd:")
			command = strings.TrimSpace(command)
			
			output, err := client.ExecuteCommand(command)
			if err != nil {
				fmt.Printf("执行命令失败: %v\n", err)
			} else {
				fmt.Printf("命令输出:\n%s\n", output)
			}
		} else {
			// 处理文件传输
			filePaths := strings.Fields(input)
			
			for _, path := range filePaths {
				localFile := strings.Trim(path, "\"'")
				fileName := filepath.Base(localFile)
				remotePath := config.RemoteDir + fileName

				fmt.Printf("正在传输文件: %s\n", fileName)
				if err := client.TransferFile(localFile, remotePath); err != nil {
					log.Printf("文件传输失败 %s: %v\n", fileName, err)
				} else {
					fmt.Printf("成功传输文件 %s 到 %s\n", fileName, remotePath)
				}
			}
		}

		fmt.Println("\n请继续操作（拖拽文件/输入命令/exit）：")
	}

	if err := scanner.Err(); err != nil {
		log.Printf("读取输入错误: %v\n", err)
	}
}
