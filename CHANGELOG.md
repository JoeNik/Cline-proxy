# 更新日志

## 2026-09-15 - 官方模型列表查询与支持模型动态增删

### 新功能

- 新增官方推荐模型接口查询：`https://api.cline.bot/api/v1/ai/cline/recommended-models`
- 管理后台可一键拉取官方列表，按官方推荐/免费/Cline Pass/Cline Cloud 分组展示
- 支持通过官方列表或手动输入模型 ID 添加/删除代理支持的模型，列表持久化到 `.cline-models.json`
- 支持模型列表支持勾选批量删除和一键清空
- `/v1/models` 与 `/admin/api/models` 改为从可持久化的支持模型列表动态生成

### 模型调整

- 默认模型改为 `cline-free/deepseek-v4.1-flash`
- 默认支持列表同步官方模型分组，新增 `cline-free/muse-spark-1.3-contributor`、
  `cline-free/solar-pro4`、`poolside/laguna-s-2.1:free`、`cline-pass/deepseek-v4.1-flash`、
  `cline-pass/glm-5.3`、`cline-pass/glm-5.3-flash`、`cline-pass/qwen3.8-max`、
  `cline-cloud/kimi-k3`、`cline-cloud/deepseek-v4-flash`、`cline-cloud/glm-5.2`

### 更新的文件

- `models_store.go` - 支持模型列表存储、官方模型列表拉取与分组
- `admin.go` / `admin_html.go` - 管理后台模型管理 API 与 UI
- `proxy.go` - `/v1/models` 改用支持模型列表
- `README.md` / `.gitignore` - 文档与数据文件忽略规则

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
