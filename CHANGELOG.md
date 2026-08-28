# 更新日志

## 2026-08-28 - 支持 z-ai/glm-5.3-flash 与 deepseek/deepseek-v4-flash

### 新增模型

- `z-ai/glm-5.3-flash` - Z.ai GLM 5.3 Flash（直接提供商模型）

### 请求头覆盖

- 默认请求头更新为 Cline 4.1.16 VSCode 客户端：`user-agent: Cline/4.1.16`、`x-client-type: cline-vscode`、`x-platform: vscode`、`x-core-version: 4.1.16`、`x-platform-version: 1.106.0`、`x-client-version: 4.1.16`、`http-referer: https://cline.bot`、`x-title: Cline`

### 更新的文件

- `proxy.go` - 更新 `/v1/models` 端点的模型列表
- `admin.go` - 更新管理后台模型列表和默认请求头
- `README.md` - 更新模型说明

## 2026-08-05 - 同步官方模型列表

### 新增模型

根据 Cline 官方支持的模型列表，新增以下 ClinePass 订阅模型：

1. **DeepSeek 系列**
   - `cline-pass/deepseek-v4-pro` - DeepSeek V4 Pro（新增）

2. **Moonshot AI (Kimi) 系列**
   - `cline-pass/kimi-k2.6` - Kimi K2.6（新增）
   - `cline-pass/kimi-k2.7-code` - Kimi K2.7 Code（新增）
   - `cline-pass/kimi-k3` - Kimi K3（新增）

3. **MiMo 系列**
   - `cline-pass/mimo-v2.5` - MiMo V2.5（新增）
   - `cline-pass/mimo-v2.5-pro` - MiMo V2.5 Pro（新增）

4. **MiniMax 系列**
   - `cline-pass/minimax-m3` - MiniMax M3（新增）

5. **Qwen 系列**
   - `cline-pass/qwen3.7-plus` - Qwen 3.7 Plus（新增）

### 更新的文件

- `proxy.go` - 更新 `/v1/models` 端点的模型列表
- `admin.go` - 更新管理后台的模型列表
- `README.md` - 更新文档中的模型说明

### 模型总数

- **免费模型**: 1 个（`cline-free/glm-5.2`）
- **ClinePass 模型**: 11 个

所有模型都已同步到 Cline 官方最新支持的列表（截至 2026-08-05）。
