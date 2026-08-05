# 认证机制说明

本项目有两套独立的认证机制：

## 1. 管理面板密码 (Admin Password)

**用途：** 保护管理面板访问

**配置：** `.env` 文件中的 `ADMIN_PASSWORD`

**作用范围：**
- `/admin/` - 管理面板页面
- `/admin/api/*` - 所有管理 API 端点

**使用方式：**
- 访问管理面板时输入密码
- 前端自动在请求头中添加：`Authorization: Bearer <admin-password>`

**说明：**
- 如果 `ADMIN_PASSWORD` 为空或未设置，则不需要密码
- 此密码**不影响**代理 API 的调用

---

## 2. API Keys

**用途：** 保护代理 API 调用

**配置：** 在管理面板的"API Keys"标签页生成

**作用范围：**
- `/v1/chat/completions` - OpenAI 兼容接口
- `/v1/messages` - Anthropic Messages API 接口

**使用方式：**
客户端调用时需要在请求头中添加以下任一方式：

```bash
# 方式 1：使用 x-api-key 头
curl -H "x-api-key: your-api-key-here" ...

# 方式 2：使用 Authorization Bearer
curl -H "Authorization: Bearer your-api-key-here" ...
```

**说明：**
- 如果未配置任何 API Key，则允许无需认证访问
- 此 API Key **不影响**管理面板的访问

---

## 总结

```
┌─────────────────────────────────────────┐
│  管理面板密码 (ADMIN_PASSWORD)           │
│  ↓                                      │
│  /admin/          管理页面               │
│  /admin/api/*     管理 API              │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│  API Keys (在管理面板生成)                │
│  ↓                                      │
│  /v1/chat/completions   代理 API        │
│  /v1/messages           代理 API        │
└─────────────────────────────────────────┘
```

**两套认证机制完全独立，互不影响！**
