# Harbor 镜像迁移脚本 / Harbor Image Migration

## 1. 工作目的与亮点 / Work Purpose and Highlights

### 工作目的 / Work Purpose
我们旨在开发一个高效且可靠的 Python 脚本，用于将 Docker 镜像从一个 Harbor 实例迁移到另一个。该脚本不仅需要处理基本的镜像下载和上传操作，还需应对 Harbor 的安全机制，确保迁移过程顺利完成。

**Our goal is to develop an efficient and reliable Python script to migrate Docker images from one Harbor instance to another. The script handles not only basic image download and upload operations but also addresses Harbor's security mechanisms to ensure a smooth migration process.**

### 亮点 / Highlights
- **自动化迁移**：通过脚本实现镜像的自动下载和上传，减少手动操作，提高效率。
  
  **Automated Migration**: Automate image downloading and uploading through the script, reducing manual operations and increasing efficiency.
  
- **并发处理**：利用多线程并发执行下载和上传任务，大幅缩短迁移时间。
  
  **Concurrent Processing**: Utilize multithreading to execute download and upload tasks concurrently, significantly reducing migration time.
  
- **去重机制**：在上传前检查目标 Harbor 中 Blob 是否已存在，避免重复上传，节省网络带宽和存储空间。
  
  **Deduplication Mechanism**: Check if blobs already exist in the target Harbor before uploading to avoid duplicate uploads, saving network bandwidth and storage space.
  
- **简洁性**：保持代码简洁，不依赖于额外的库，易于维护和扩展。
  
  **Simplicity**: Keep the code concise without relying on additional libraries, making it easy to maintain and extend.
  
- **用户定义标签**：允许用户为新镜像定义自定义标签，确保镜像版本的一致性和可追溯性。
  
  **User-Defined Tags**: Allow users to define custom tags for the new images, ensuring consistency and traceability of image versions.

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
