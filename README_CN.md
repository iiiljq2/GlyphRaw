# GlyphRaw

**GlyphRaw** 是一个自动化的命令行工具，利用 [FontDiffuser] 深度学习模型，将手写字符图像转换为个性化手写字体。只需提供你的手写样本，GlyphRaw 即可生成 TTF 或 OTF 字体文件。

## 功能特性

- **自动化字体生成**: 将手写字符图像转换为专业字体
- **Docker 集成**: 使用 Docker 容器化技术，确保运行环境的一致性
- **GPU 加速**: 全面支持 NVIDIA GPU (CUDA) 加速
- **灵活输出**: 支持生成 TTF 或 OTF 格式
- **自动配置**: 自动下载模型权重并初始化环境
- **易于配置**: 简洁的命令行交互界面

## 系统要求

- **操作系统**: Linux, macOS, 或 Windows
- **Go**: 版本 1.21 或更高
- **Docker**: 版本 20.0 或更高
- **GPU (推荐)**: 拥有 CUDA 11.7 支持的 NVIDIA GPU (至少 4GB 显存)

### 软件依赖
所有 Python 依赖均由 Docker 镜像自动处理。本程序集成了以下内容：

- PyTorch 2.0.0 (基于 CUDA 11.7)
- FontDiffuser 模型及其相关依赖
- 用于字体生成的 FontForge
- 所需的 Python 库（如 diffusers, transformers 等）

## 快速开始

```bash
./glyphraw
```

按照交互式提示操作即可生成字体。

## 使用方法

### 支持的格式

JPG, JPEG, PNG, BMP, TIFF

### 单文件处理
```bash
./glyphraw
# 提示输入时: /path/to/my_handwriting.jpg
```

### 批量处理 (目录)
```bash
./glyphraw
# 提示输入时: /path/to/my_handwriting_samples/
```

### 输出

生成的字体保存在 `article_output/` 目录中。

## 项目结构

```
glyphraw/
├── main.go              # 入口文件
├── internal/
│   ├── logger/          # 日志系统
│   ├── config/          # 配置与模型定义
│   ├── docker/          # Docker 执行逻辑
│   ├── font/            # 字体生成
│   ├── setup/           # 模型下载与初始化
│   └── cli/             # 用户交互
├── pkg/
│   ├── util/            # 工具函数
│   └── download/        # 文件下载
├── scripts/
│   └── pack_font.py     # 字体打包脚本
└── Dockerfile           # Docker 容器定义
```

## 故障排除

### Docker 未运行
请从 https://www.docker.com/ 安装并启动 Docker Desktop。

### GPU 显存不足
请关闭其他占用 GPU 的程序。模型运行至少需要 4GB+ 的显存以保证性能稳定。

### 生成的字体缺少字符
请检查 `article_output/[stylename]/` 目录下的生成的 PNG 图片，并查看日志输出。

## 开发

### 添加新功能

1. **新模型**: 在 `internal/config/models.go` 中添加配置
2. **新字体格式**: 扩展 `internal/font/packer.go`
3. **CLI 增强**: 修改 `internal/cli/interactive.go`

## 性能

**使用 GPU 加速**:
- 单个字符: 1-3 秒
- 完整字体 (100+ 字符): 5-10 分钟

## 已知限制

1. 主要支持中文字符
2. 在没有 GPU 的情况下速度显著变慢
3. 初始下载模型约 2-3GB
4. 其他脚本需要模型微调

## 相关项目

- [FontDiffuser](https://github.com/yeungchenwa/FontDiffuser) - 原始深度学习模型
- [FontForge](https://github.com/fontforge/fontforge) - 字体生成引擎

## 支持
如果你在使用过程中遇到问题，或者有任何改进建议，欢迎通过以下方式联系我获取帮助：

**Discord**: iiiljq (用户名: hunter)

---

**维护者**: iiiljq(Hunter)