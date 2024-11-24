# Harbor 镜像迁移工具 / Harbor Image Migration Tool

## 1. 工作目的与亮点 / Work Purpose and Highlights

### 工作目的 / Work Purpose
开发一个高性能的 Go 语言工具，用于在 Harbor 实例之间进行 Docker 镜像的无缝迁移。该工具采用了创新的流式传输技术，显著提升了迁移效率。

**Develop a high-performance Go tool for seamless Docker image migration between Harbor instances. The tool employs innovative streaming technology for significantly improved migration efficiency.**

### 亮点 / Highlights

#### 🚀 流式传输技术 / Streaming Technology
- **零存储开销**：通过流式传输技术，实现数据的直接转发，无需在本地存储任何镜像层文件
  
  **Zero Storage Overhead**: Achieves direct data forwarding through streaming technology, eliminating the need for local storage of image layers
  
- **内存效率**：采用流式处理，避免将整个镜像层加载到内存，显著降低内存占用
  
  **Memory Efficiency**: Utilizes streaming processing to avoid loading entire image layers into memory, significantly reducing memory usage
  
- **实时传输**：数据从源 Harbor 下载的同时直接上传到目标 Harbor，最大化网络利用效率
  
  **Real-time Transfer**: Data is simultaneously downloaded from source Harbor and uploaded to target Harbor, maximizing network utilization efficiency

#### 💪 并发处理 / Concurrent Processing
- **智能并发**：采用 Go 协程实现多层并发传输，充分利用网络带宽
  
  **Smart Concurrency**: Implements multi-layer concurrent transfer using Go routines, fully utilizing network bandwidth
  
- **资源控制**：通过信号量机制智能控制并发数量，避免资源过载
  
  **Resource Control**: Intelligently controls concurrency through semaphore mechanisms to prevent resource overload

#### 🛡️ 可靠性保障 / Reliability Assurance
- **完整性校验**：自动验证每个镜像层的完整性，确保迁移质量
  
  **Integrity Verification**: Automatically verifies the integrity of each image layer to ensure migration quality
  
- **去重优化**：智能检测目标仓库中已存在的镜像层，避免重复传输
  
  **Deduplication**: Intelligently detects existing image layers in the target repository to avoid redundant transfers

#### 🔄 优化的工作流程 / Optimized Workflow
1. **智能分析**：
   - 解析源镜像 manifest
   - 规划传输策略
   
   **Smart Analysis**:
   - Parse source image manifest
   - Plan transfer strategy

2. **并发流式传输**：
   - 配置文件和镜像层的并发传输
   - 实时数据流转发
   
   **Concurrent Streaming**:
   - Concurrent transfer of config files and image layers
   - Real-time data stream forwarding

3. **自动清理**：
   - 仅保留必要的 manifest 文件
   - 传输完成后自动清理
   
   **Automatic Cleanup**:
   - Retain only necessary manifest files
   - Automatic cleanup after transfer

#### 📊 性能对比 / Performance Comparison

| 特性 / Feature | 传统方式 / Traditional | 流式传输 / Streaming |
|----------------|----------------------|-------------------|
| 本地存储 / Local Storage | 需要 / Required | 不需要 / Not Required |
| 内存占用 / Memory Usage | 高 / High | 低 / Low |
| 传输速度 / Transfer Speed | 较慢 / Slower | 更快 / Faster |
| 资源消耗 / Resource Consumption | 较高 / Higher | 更低 / Lower |

---

## 2. 遇到的困难与解决方案 / Challenges Encountered and Solutions

### 遇到的困难 / Challenges

1. **CSRF Token 错误**
   - **问题描述**：在使用 Python 脚本上传 Blob 时，Harbor 返回 `403 FORBIDDEN` 错误，提示 `CSRF token invalid`。
   - **原因分析**：Harbor 启用了 CSRF 防护机制，即使使用 HTTP Basic Authentication，某些请求（如 `POST` 和 `PUT`）仍需特定的 CSRF Token 或禁止发送 Cookie。

2. **认证与会话管理**
   - **问题描述**：初始脚本中使用了不适当的会话管理和登录步骤，导致认证失败。
   - **原因分析**：错误地处理了 Harbor 的认证机制，未正确禁用 Cookie，触发了 CSRF 防护。

3. **效率问题**
   - **问题描述**：迁移大量镜像时，单线程的下载和上传操作效率低下。
   - **原因分析**：未利用并发机制，导致整体迁移时间过长。

4. **重复上传**
   - **问题描述**：脚本在上传时未检查目标 Harbor 中是否已存在相同的 Blob，导致不必要的网络流量。
   - **原因分析**：缺乏对 Blob 存在性的预检，未利用 Harbor 的去重机制。

### 解决方案 / Solutions

1. **禁用 Cookie 和移除 CSRF Token 头**
   - **实施方法**：
     - 在所有请求中禁用 Cookie，通过 `session.cookies.clear()` 确保不发送任何 Cookie。
     - 移除所有与 CSRF 相关的请求头，如 `X-Harbor-CSRF-Token`，避免触发 CSRF 防护。
     
     **Disable Cookies and Remove CSRF Token Headers**:
     - Clear session cookies using `session.cookies.clear()` to ensure no cookies are sent with requests.
     - Remove any CSRF-related headers like `X-Harbor-CSRF-Token` to prevent triggering CSRF protection.

2. **使用 HTTP Basic Authentication**
   - **实施方法**：
     - 直接在请求中使用用户名和密码进行认证，无需额外的登录步骤或会话管理。
     - 设置 `auth = (username, password)` 并在每个请求中使用此认证方式。
     
     **Use HTTP Basic Authentication**:
     - Authenticate directly using username and password in requests without additional login steps.
     - Set `auth = (username, password)` and use it in every request.

3. **并发下载与上传**
   - **实施方法**：
     - 使用 `concurrent.futures.ThreadPoolExecutor` 实现多线程并发处理下载和上传任务。
     - 设置合理的并发线程数（如 `MAX_WORKERS = 8`），根据系统资源和网络带宽进行调整。
     
     **Implement Concurrent Download and Upload**:
     - Utilize `ThreadPoolExecutor` for multithreaded concurrent handling of download and upload tasks.
     - Adjust `MAX_WORKERS` based on system resources and network bandwidth.

4. **Blob 存在性检查**
   - **实施方法**：
     - 在上传每个 Blob 之前，发送 `HEAD` 请求检查目标 Harbor 中该 Blob 是否已存在。
     - 如果 Blob 已存在，跳过上传，减少不必要的网络流量和存储消耗。
     
     **Implement Blob Existence Check**:
     - Send `HEAD` requests before uploading each blob to verify its existence in the target Harbor.
     - Skip uploading if the blob already exists to save network bandwidth and storage.

5. **用户定义新镜像标签**
   - **实施方法**：
     - 在脚本中定义新的镜像标签（如 `NEW_IMAGE_TAG = "v1.0.0"`），并在注册 manifest 时使用该标签。
     - 确保迁移后的镜像在目标 Harbor 中具有明确且可追溯的版本标识。
     
     **Allow User-Defined Image Tags**:
     - Define new image tags (e.g., `NEW_IMAGE_TAG = "v1.0.0"`) and use them when registering the manifest.
     - Ensure migrated images have clear and traceable version identifiers in the target Harbor.

6. **简化日志与错误处理**
   - **实施方法**：
     - 使用基础的 `print` 语句记录日志，保持代码简洁。
     - 在关键步骤添加详细的错误处理和日志输出，确保任何失败都能被及时捕捉和报告。
     
     **Simplify Logging and Error Handling**:
     - Use basic `print` statements for logging to keep the code simple.
     - Add detailed error handling and logging at key steps to ensure failures are promptly captured and reported.

---

## 3. 后续优化展望 / Future Optimization Prospects

1. **动态配置管理 / Dynamic Configuration Management**
   - **建议 / Recommendation**：
     - 将 Harbor 配置信息（如 URL、用户名、密码、新标签）通过环境变量或外部配置文件传递，避免在脚本中硬编码敏感信息。
     
     **Dynamic Configuration**:
     - Pass Harbor configuration details (URL, username, password, new tags) via environment variables or external configuration files to avoid hardcoding sensitive information.

2. **高级错误处理与重试机制 / Advanced Error Handling and Retry Mechanism**
   - **建议 / Recommendation**：
     - 为上传和下载操作添加更智能的错误处理和重试机制，处理临时的网络故障或服务器问题。
     
     **Advanced Error Handling and Retries**:
     - Implement more intelligent error handling and retry mechanisms for upload and download operations to handle temporary network failures or server issues.

3. **日志记录与监控 / Logging and Monitoring**
   - **建议 / Recommendation**：
     - 将日志输出到文件中，便于后续分析和审计。
     
     **Logging and Monitoring**:
     - Output logs to files for easier post-operation analysis and auditing.

4. **进度显示 / Progress Display**
   - **建议 / Recommendation**：
     - 为下载和上传操作添加进度条，提升用户体验。
     
     **Progress Indicators**:
     - Add progress indicators for download and upload operations to enhance user experience.

5. **清理临时文件 / Cleanup Temporary Files**
   - **建议 / Recommendation**：
     - 在迁移完成后，自动清理下载的临时文件，节省磁盘空间。
     
     **Automatic Cleanup**:
     - Automatically clean up temporary downloaded files after migration to save disk space.

6. **多平台支持与优化 / Cross-Platform Support and Optimization**
   - **建议 / Recommendation**：
     - 优化脚本以支持不同操作系统和环境，确保跨平台的兼容性。
     
     **Cross-Platform Compatibility**:
     - Optimize the script to support various operating systems and environments, ensuring cross-platform compatibility.

7. **用户交互与界面改进 / User Interaction and Interface Enhancements**
   - **建议 / Recommendation**：
     - 提供命令行参数或交互式输入，允许用户动态指定源和目标 Harbor 信息、项目路径、标签等。
     
     **Enhanced User Interaction**:
     - Provide command-line arguments or interactive inputs to allow users to dynamically specify source and target Harbor details, project paths, tags, etc.

---

## 结语 / Conclusion

通过此次项目，我们成功开发了一个高效、可靠且简洁的 Python 脚本，实现了 Docker 镜像在 Harbor 实例之间的自动迁移。尽管在过程中遇到了 CSRF 防护和认证机制的挑战，但通过合理的解决方案，我们确保了迁移过程的顺利进行。

未来，我们计划进一步优化脚本，提升其灵活性、鲁棒性和用户体验，使其在更广泛的应用场景中发挥更大的作用。

**Through this project, we successfully developed an efficient, reliable, and concise Python script to automate the migration of Docker images between Harbor instances. Despite encountering challenges related to CSRF protection and authentication mechanisms, we ensured a smooth migration process through appropriate solutions.**

未来，我们计划进一步优化脚本，提升其灵活性、鲁棒性和用户体验，使其在更广泛的应用场景中发挥更大的作用。

**In the future, we plan to further optimize the script to enhance its flexibility, robustness, and user experience, enabling it to perform better in a wider range of application scenarios.**

感谢你的积极参与和反馈，期待在未来的项目中继续协作！

**Thank you for your active participation and feedback. We look forward to collaborating on future projects!**

祝工作顺利，生活愉快！😊

**Wishing you success in your work and happiness in life! 😊**