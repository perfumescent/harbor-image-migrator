import requests
import os
import hashlib
import json
import warnings
from urllib3.exceptions import InsecureRequestWarning
from concurrent.futures import ThreadPoolExecutor, as_completed

# 忽略不安全的 HTTPS 请求警告
warnings.simplefilter('ignore', InsecureRequestWarning)

# 全局配置
VERIFY_SSL = False    # 如果使用受信任的 SSL 证书，请设置为 True
CHUNK_SIZE = 8192     # 文件块大小，用于上传和下载
MAX_WORKERS = 8       # 并发线程数

# 用户输入信息
SOURCE_HARBOR = "https://harbor1.cn"
SOURCE_USERNAME = "root"
SOURCE_PASSWORD = "12345"

DEST_HARBOR = "https://harbor2.cn"
DEST_USERNAME = "root"
DEST_PASSWORD = "12345"



def calculate_digest(file_path):
    """计算文件的 SHA256 摘要"""
    sha256 = hashlib.sha256()
    with open(file_path, "rb") as f:
        while True:
            chunk = f.read(CHUNK_SIZE)
            if not chunk:
                break
            sha256.update(chunk)
    return f"sha256:{sha256.hexdigest()}"

def download_blob(harbor_url, project_path, digest, output_path, auth):
    """下载单个 blob"""
    url = f"{harbor_url}/v2/{project_path}/blobs/{digest}"
    try:
        response = requests.get(url, auth=auth, verify=VERIFY_SSL, stream=True)
        if response.status_code != 200:
            raise Exception(f"下载 blob 失败：{response.status_code} - {response.text}")
        with open(output_path, "wb") as f:
            for chunk in response.iter_content(CHUNK_SIZE):
                if chunk:
                    f.write(chunk)
        print(f"[INFO] 下载 blob 成功：{os.path.basename(output_path)}")
    except Exception as e:
        print(f"[ERROR] 下载 {os.path.basename(output_path)} 失败：{str(e)}")
        raise

def blob_exists(harbor_url, project_path, digest, auth):
    """检查 Blob 是否已存在于 Harbor"""
    url = f"{harbor_url}/v2/{project_path}/blobs/{digest}"
    response = requests.head(url, auth=auth, verify=VERIFY_SSL)
    return response.status_code == 200

def upload_blob_to_harbor(harbor_url, project_path, file_path, digest, file_type, auth):
    """上传单个 blob 到目标 Harbor，带存在性检查"""
    try:
        if blob_exists(harbor_url, project_path, digest, auth):
            print(f"[INFO] {file_type} 已存在，跳过上传。")
            return
        
        # 创建上传会话
        upload_url = f"{harbor_url}/v2/{project_path}/blobs/uploads/"
        response = requests.post(upload_url, auth=auth, verify=VERIFY_SSL)
        if response.status_code != 202:
            raise Exception(f"创建上传会话失败：{response.status_code} - {response.text}")
        location = response.headers["Location"]
        print(f"[INFO] 创建上传会话成功：{file_type} - {location}")

        # 上传 blob 使用 PUT 方法
        with open(file_path, "rb") as f:
            upload_url_with_digest = f"{location}&digest={digest}"
            response = requests.put(upload_url_with_digest, auth=auth, data=f, verify=VERIFY_SSL)
        if response.status_code != 201:
            raise Exception(f"上传失败：{response.status_code} - {response.text}")

        print(f"[INFO] {file_type} 上传成功：{os.path.basename(file_path)}")
    except Exception as e:
        print(f"[ERROR] 上传 {file_type} 失败：{str(e)}")
        raise

def register_manifest(harbor_url, project_path, manifest_path, auth, image_tag):
    """注册 manifest 到目标 Harbor"""
    try:
        with open(manifest_path, "r") as f:
            manifest_data = f.read()

        url = f"{harbor_url}/v2/{project_path}/manifests/{image_tag}"
        headers = {"Content-Type": "application/vnd.docker.distribution.manifest.v2+json"}
        print(f"[INFO] 注册 manifest: {url}")
        response = requests.put(
            url,
            headers=headers,
            data=manifest_data,
            auth=auth,
            verify=VERIFY_SSL,
        )
        if response.status_code in [200, 201]:
            print("[INFO] Manifest 注册成功")
        else:
            raise Exception(f"Manifest 注册失败：{response.status_code} - {response.text}")
    except Exception as e:
        print(f"[ERROR] 注册 manifest 失败：{str(e)}")
        raise

def download_from_harbor(harbor_url, username, password, project_path, image_tag, output_dir):
    """从源 Harbor 下载镜像"""
    auth = (username, password)

    # 获取 manifest
    manifest_url = f"{harbor_url}/v2/{project_path}/manifests/{image_tag}"
    print(f"[INFO] 获取 manifest: {manifest_url}")
    try:
        manifest_response = requests.get(manifest_url, auth=auth, verify=VERIFY_SSL)
        if manifest_response.status_code != 200:
            raise Exception(f"获取 manifest 失败：{manifest_response.status_code} - {manifest_response.text}")
        manifest = manifest_response.json()
    except Exception as e:
        print(f"[ERROR] 获取 manifest 失败：{str(e)}")
        raise

    # 保存 manifest 文件
    manifest_path = os.path.join(output_dir, "manifest.json")
    try:
        with open(manifest_path, "w") as f:
            json.dump(manifest, f, indent=4)
        print(f"[INFO] 保存 manifest 文件: {manifest_path}")
    except Exception as e:
        print(f"[ERROR] 保存 manifest 文件失败：{str(e)}")
        raise

    # 下载配置文件和层文件
    blobs = []

    # 配置文件
    config_digest = manifest["config"]["digest"]
    config_path = os.path.join(output_dir, "config.json")
    blobs.append({
        "digest": config_digest,
        "output_path": config_path,
        "file_type": "config.json"
    })

    # 层文件
    for i, layer in enumerate(manifest["layers"]):
        layer_digest = layer["digest"]
        layer_path = os.path.join(output_dir, f"layer{i + 1}.tar.gz")
        blobs.append({
            "digest": layer_digest,
            "output_path": layer_path,
            "file_type": f"layer{i + 1}.tar.gz"
        })

    # 并发下载
    print(f"[INFO] 开始并发下载 {len(blobs)} 个 blob...")
    with ThreadPoolExecutor(max_workers=MAX_WORKERS) as executor:
        future_to_blob = {
            executor.submit(download_blob, harbor_url, project_path, blob["digest"], blob["output_path"], auth): blob
            for blob in blobs
        }
        for future in as_completed(future_to_blob):
            blob = future_to_blob[future]
            try:
                future.result()
            except Exception as e:
                print(f"[ERROR] 下载 {blob['file_type']} 失败：{str(e)}")
                raise

    print(f"[INFO] 镜像下载完成，保存路径：{output_dir}")


def upload_to_harbor_concurrent(harbor_url, project_path, output_dir, image_tag, auth):
    """上传镜像到目标 Harbor，使用并发"""
    blobs = []

    # 配置文件
    config_path = os.path.join(output_dir, "config.json")
    config_digest = calculate_digest(config_path)
    blobs.append({
        "file_path": config_path,
        "digest": config_digest,
        "file_type": "config.json"
    })

    # 层文件
    layer_files = sorted(
        [f for f in os.listdir(output_dir) if f.startswith("layer") and f.endswith(".tar.gz")],
        key=lambda x: int(''.join(filter(str.isdigit, x)) or 0)
    )
    for layer_file in layer_files:
        layer_path = os.path.join(output_dir, layer_file)
        layer_digest = calculate_digest(layer_path)
        blobs.append({
            "file_path": layer_path,
            "digest": layer_digest,
            "file_type": layer_file
        })

    # 并发上传
    print(f"[INFO] 开始并发上传 {len(blobs)} 个 blob...")
    with ThreadPoolExecutor(max_workers=MAX_WORKERS) as executor:
        future_to_blob = {
            executor.submit(
                upload_blob_to_harbor,
                harbor_url,
                project_path,
                blob["file_path"],
                blob["digest"],
                blob["file_type"],
                auth
            ): blob
            for blob in blobs
        }
        for future in as_completed(future_to_blob):
            blob = future_to_blob[future]
            try:
                future.result()
            except Exception as e:
                print(f"[ERROR] 上传 {blob['file_type']} 失败：{str(e)}")
                raise

    # 注册 manifest
    print(f"[INFO] 开始注册 manifest...")
    try:
        register_manifest(harbor_url, project_path, os.path.join(output_dir, "manifest.json"), auth, image_tag)
    except Exception as e:
        print(f"[ERROR] 注册 manifest 失败：{str(e)}")
        raise

def upload_to_harbor(harbor_url, username, password, project_path, output_dir, image_tag):
    """上传镜像到目标 Harbor"""
    auth = (username, password)

    # 创建 session 并禁用 Cookie
    session = requests.Session()
    session.auth = auth
    session.cookies.clear()  # 确保不发送任何 Cookie

    try:
        upload_to_harbor_concurrent(harbor_url, project_path, output_dir, image_tag, auth)
        print("[INFO] 镜像上传完成")
    except Exception as e:
        print(f"[ERROR] 上传过程出现错误：{str(e)}")
        raise

def main():
    source_project = "project/original-image"
    source_tag = "v0"  # 源镜像标签
    dest_project = "project/new-image"
    dest_tag = "v1" # 定义新镜像标签
    output_dir = "./downloaded_files"

    # 创建输出目录
    if not os.path.exists(output_dir):
        os.makedirs(output_dir)

    try:
        # Step 1: 从源 Harbor 下载镜像
        print("[INFO] 开始从源 Harbor 拉取镜像...")
        download_from_harbor(SOURCE_HARBOR, SOURCE_USERNAME, SOURCE_PASSWORD, source_project, source_tag, output_dir)

        # Step 2: 将镜像推送到目标 Harbor
        print("[INFO] 开始将镜像推送到目标 Harbor...")
        upload_to_harbor(DEST_HARBOR, DEST_USERNAME, DEST_PASSWORD, dest_project, output_dir, dest_tag)

        # 打印新镜像地址
        new_image_address = f"{DEST_HARBOR}/v2/{dest_project}/manifests/{dest_tag}"
        print(f"[INFO] 镜像迁移完成！新镜像地址：{new_image_address}")
    except Exception as e:
        print(f"[ERROR] 发生错误：{str(e)}")

if __name__ == "__main__":
    main()
