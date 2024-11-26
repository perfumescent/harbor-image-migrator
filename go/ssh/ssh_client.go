package ssh

import (
	"dockerImageMigrator/log"
	"fmt"
	"golang.org/x/crypto/ssh"
	"os"
	"path/filepath"
	"time"
)

// SSHConfig 存储SSH连接配置
type SSHConfig struct {
	Host      string
	Port      int
	Username  string
	Password  string
	KeyFile   string
	RemoteDir string
}

// SSHClient SSH客户端结构体
type SSHClient struct {
	Config *SSHConfig
	client *ssh.Client
}

// NewSSHClient 创建新的SSH客户端
func NewSSHClient(config *SSHConfig) (*SSHClient, error) {
	return &SSHClient{
		Config: config,
	}, nil
}

// Connect 建立SSH连接
func (s *SSHClient) Connect() error {
	var authMethods []ssh.AuthMethod

	if s.Config.Password != "" {
		authMethods = append(authMethods, ssh.Password(s.Config.Password))
	}

	if s.Config.KeyFile != "" {
		key, err := os.ReadFile(s.Config.KeyFile)
		if err != nil {
			return fmt.Errorf("unable to read private key: %v", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return fmt.Errorf("unable to parse private key: %v", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	config := &ssh.ClientConfig{
		User:            s.Config.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", s.Config.Host, s.Config.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		log.Infof("Failed to connect to %s: %v", addr, err)
		return fmt.Errorf("failed to connect: %v", err)
	}

	s.client = client
	log.Infof("Successfully connected to %s", addr)
	return nil
}

// ExecuteCommand 执行远程命令
func (s *SSHClient) ExecuteCommand(cmd string) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("client not connected")
	}

	session, err := s.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	log.Infof("Executing command: %s", cmd)
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		log.Infof("Command execution failed: %v", err)
		return string(output), fmt.Errorf("failed to execute command: %v", err)
	}

	log.Infof("Command executed successfully")
	return string(output), nil
}

// TransferFile 传输文件到远程服务器
func (s *SSHClient) TransferFile(localPath, remotePath string) error {
	if s.client == nil {
		return fmt.Errorf("client not connected")
	}

	// 创建新的SFTP客户端
	session, err := s.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %v", err)
	}
	defer localFile.Close()

	log.Infof("Transferring file from %s to %s", localPath, remotePath)

	// 确保远程目录存在
	remoteDir := filepath.Dir(remotePath)
	if _, err := s.ExecuteCommand(fmt.Sprintf("mkdir -p %s", remoteDir)); err != nil {
		return fmt.Errorf("failed to create remote directory: %v", err)
	}

	// 使用scp命令传输文件
	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()

		content, _ := os.ReadFile(localPath)
		fmt.Fprintln(w, "C0644", len(content), filepath.Base(remotePath))
		w.Write(content)
		fmt.Fprint(w, "\x00")
	}()

	if err := session.Run(fmt.Sprintf("scp -t %s", remotePath)); err != nil {
		log.Infof("File transfer failed: %v", err)
		return fmt.Errorf("failed to transfer file: %v", err)
	}

	log.Infof("File transferred successfully")
	return nil
}

// Close 关闭SSH连接
func (s *SSHClient) Close() error {
	if s.client != nil {
		log.Infof("Closing SSH connection")
		return s.client.Close()
	}
	return nil
}

// WriteStringToFile 将字符串内容写入远程文件
func (s *SSHClient) WriteStringToFile(fileContentStr, remotePath string) error {
	if s.client == nil {
		return fmt.Errorf("client not connected")
	}

	// 创建新的会话
	session, err := s.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	log.Infof("Writing content to remote file: %s", remotePath)

	// 确保远程目录存在
	remoteDir := filepath.Dir(remotePath)
	if _, err := s.ExecuteCommand(fmt.Sprintf("mkdir -p %s", remoteDir)); err != nil {
		return fmt.Errorf("failed to create remote directory: %v", err)
	}

	// 通过 stdin pipe 写入文件内容
	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()

		content := []byte(fileContentStr)
		fmt.Fprintln(w, "C0644", len(content), filepath.Base(remotePath))
		w.Write(content)
		fmt.Fprint(w, "\x00")
	}()

	if err := session.Run(fmt.Sprintf("scp -t %s", remotePath)); err != nil {
		log.Infof("Failed to write content to file: %v", err)
		return fmt.Errorf("failed to write content to file: %v", err)
	}

	log.Infof("Content written to file successfully")
	return nil
}
