# PilotDeck Session API 使用文档

## 概述

PilotDeck 的 Session（会话）是 AI 代理执行任务的对话记录。以下 API 用于创建、查询、管理会话。

---

## 1. 检查 Session 是否存在并获取消息

```
GET /api/sessions/:sessionId/messages?projectPath=<项目路径>
```

### 参数

| 参数 | 位置 | 必填 | 说明 |
|------|------|------|------|
| `sessionId` | URL | 是 | 会话 ID，格式如 `web-s_xxx` 或 `web:s_xxx` |
| `projectPath` | Query | 是 | 项目绝对路径（也可以用 `projectName`） |
| `limit` | Query | 否 | 返回条数上限，默认全部 |
| `offset` | Query | 否 | 偏移量，用于翻页 |

### 返回

```json
{
  "messages": [...],
  "total": 42,
  "hasMore": false,
  "offset": 0,
  "limit": null
}
```

### 判断 Session 是否存在

- 返回 `messages` 数组不为空 → session 存在且有消息
- 返回 `messages` 为空数组（`[]`）且 `total: 0` → session 不存在或刚创建尚无消息
- **注意**：即使 session 存在但刚创建还没有消息，也返回空数组

### 注意事项

此接口 `authenticateToken` 中间件默认需要 JWT 认证。如果 `DISABLE_LOCAL_AUTH=true`（开发模式默认开启），则可直接访问。

---

## 2. 创建或延续 Session —— 提交 Agent 任务

```
POST /api/agent
```

### 关键参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `projectPath` | string | 是 | 项目绝对路径 |
| `message` | string | 是 | 任务描述（发给 AI 代理的指令） |
| `sessionId` | string | 否 | 延续已有 session 时传入；不传则创建新 session |
| `stream` | boolean | 否 | 是否 SSE 流式响应，默认 `true` |

### 认证

使用 `x-api-key` 头部传入 API key（通过 `POST /api/settings/api-keys` 创建）。

### 判断 Session 是否在运行中

- 提交后 session 进入处理状态
- 可通过 `GET /api/sessions/:sessionId/messages` 轮询查看是否有新消息产生
- 当返回的 `messages` 数组不再增加，且最后一条消息是 assistant 的完整回复时，说明处理完成

### 示例

```bash
# 创建新 session
curl -X POST http://localhost:3001/api/agent \
  -H "Content-Type: application/json" \
  -H "x-api-key: ck_xxx" \
  -d '{"projectPath": "/path/to/project", "message": "做点什么", "stream": false}'

# 延续已有 session
curl -X POST http://localhost:3001/api/agent \
  -H "Content-Type: application/json" \
  -H "x-api-key: ck_xxx" \
  -d '{"projectPath": "/path/to/project", "message": "继续", "sessionId": "web-s_xxx", "stream": false}'
```

---

## 3. 列出项目的所有 Session

```
GET /api/projects/:projectName/sessions?limit=10&offset=0
```

### 参数

| 参数 | 位置 | 必填 | 说明 |
|------|------|------|------|
| `projectName` | URL | 是 | 项目名称（URL 编码后的路径片段） |
| `limit` | Query | 否 | 返回条数，默认 5 |
| `offset` | Query | 否 | 偏移量，用于翻页 |

### 返回

```json
{
  "sessions": [
    {
      "id": "web-s_xxx",
      "title": "Fix bug in login",
      "summary": "修复用户登录时的认证问题...",
      "createdAt": "2026-07-02T10:00:00.000Z",
      "lastActivity": "2026-07-02T11:30:00.000Z",
      "messageCount": 0
    }
  ],
  "total": 42,
  "hasMore": true,
  "offset": 0,
  "limit": 10
}
```

### 判断 Session 状态

返回列表中的 session 表示项目下存在该会话记录，但 **不包含实时处理状态**。要判断 session 是否正在处理中，需：
- 轮询 `GET /api/sessions/:sessionId/messages` 观察消息增长
- 或使用 WebSocket 的 `check-session-status` 事件（见第 6 节）

---

## 4. 删除 Session

```
DELETE /api/projects/:projectName/sessions/:sessionId
```

### 参数

| 参数 | 位置 | 必填 | 说明 |
|------|------|------|------|
| `projectName` | URL | 是 | 项目名称 |
| `sessionId` | URL | 是 | 要删除的会话 ID |
| `sessionKind` | Query | 否 | 会话类型 |
| `parentSessionId` | Query | 否 | 父会话 ID |
| `relativeTranscriptPath` | Query | 否 | 相对转储路径 |

### 返回

```json
{ "success": true }
```

---

## 5. 重命名 Session

```
PUT /api/sessions/:sessionId/rename
```

### 参数

| 参数 | 位置 | 必填 | 说明 |
|------|------|------|------|
| `sessionId` | URL | 是 | 会话 ID |
| `summary` | Body | 是 | 新名称/摘要，最长 500 字符 |
| `provider` | Body | 是 | 固定为 `"pilotdeck"` |

### 返回

```json
{ "success": true }
```

---

## 6. WebSocket —— 检测 Session 实时状态

实时状态（`isProcessing`）**只能通过 WebSocket 获取**，没有等价的 REST 端点。

### 连接地址

```
ws://localhost:3001/ws
```

### 认证方式

开发模式（`DISABLE_LOCAL_AUTH=true`）下自动通过，无需 token。
生产模式下需要在 URL 后附带 JWT token：

```
ws://localhost:3001/ws?token=<jwt_token>
```

### 请求格式

连接后发送 JSON 消息：

```json
{
  "type": "check-session-status",
  "sessionId": "web-s_xxx",
  "includeActiveTurnMessages": false
}
```

### 响应格式

```json
{
  "type": "session-status",
  "sessionId": "web-s_xxx",
  "provider": "pilotdeck",
  "isProcessing": true,
  "activeTurnMessages": [],
  "tokenBudget": {
    "used": 500,
    "total": 100000,
    "remaining": 99500
  }
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `isProcessing` | boolean | `true`=代理正在运行, `false`=空闲(完成/未开始) |
| `activeTurnMessages` | array | `isProcessing=true` 时返回当前回合的增量消息 |
| `tokenBudget` | object | 当前 session 的 token 配额和消耗 |

### 判断不存在的情况

- `isProcessing: false` + 随后 `GET /api/sessions/:sessionId/messages` 返回空 → session 不存在或从未活动

### 调用示例

#### Node.js

```javascript
const WebSocket = require('ws');

const ws = new WebSocket('ws://localhost:3001/ws');

ws.on('open', () => {
  ws.send(JSON.stringify({
    type: 'check-session-status',
    sessionId: 'web-s_xxx',
    includeActiveTurnMessages: false,
  }));
});

ws.on('message', (data) => {
  const msg = JSON.parse(data.toString());
  if (msg.type === 'session-status') {
    console.log('处理中:', msg.isProcessing);
    console.log('token预算:', msg.tokenBudget);
    ws.close();
  }
});

ws.on('close', () => console.log('连接关闭'));
```

#### Python

```python
import asyncio
import json
import websockets

async def check_session():
    uri = "ws://localhost:3001/ws"
    async with websockets.connect(uri) as ws:
        await ws.send(json.dumps({
            "type": "check-session-status",
            "sessionId": "web-s_xxx",
            "includeActiveTurnMessages": False,
        }))
        resp = await ws.recv()
        msg = json.loads(resp)
        if msg.get("type") == "session-status":
            print(f"处理中: {msg['isProcessing']}")

asyncio.run(check_session())
```

#### curl（通过 websocat 工具）

```bash
# 需要先安装 websocat: https://github.com/vi/websocat
echo '{"type":"check-session-status","sessionId":"web-s_xxx"}' \
  | websocat ws://localhost:3001/ws
```

### 获取所有活跃 Session

```json
{
  "type": "get-active-sessions"
}
```

返回当前所有正在处理中的 session ID 列表。

---

## 7. 获取 Session Token 用量

```
GET /api/projects/:projectName/sessions/:sessionId/token-usage
```

### 参数

| 参数 | 位置 | 必填 | 说明 |
|------|------|------|------|
| `projectName` | URL | 是 | 项目名称 |
| `sessionId` | URL | 是 | 会话 ID |
| `provider` | Query | 否 | 默认 `"pilotdeck"` |

### 返回

```json
{
  "used": 1500,
  "total": 100000,
  "breakdown": {
    "input": 1000,
    "cacheCreation": 200,
    "cacheRead": 300
  }
}
```

---

## 8. 健康检查

```
GET /health
```

无需认证。返回：

```json
{
  "status": "ok",
  "timestamp": "2026-07-02T12:00:00.000Z"
}
```

---

## 9. 认证方式

### JWT Token（浏览器端）

大多数 API 需要通过 `Authorization: Bearer <token>` 头部传入 JWT token。
获取方式：`POST /api/auth/login`

### API Key（外部程序调用）

`POST /api/agent` 使用 API key 认证。通过 `x-api-key` 头部传入。

**创建 API Key：**

```
POST /api/settings/api-keys
Authorization: Bearer <jwt-token>
Content-Type: application/json

{"keyName": "my-app-key"}
```

返回 `apiKey` 字段，格式为 `ck_xxxxxxxxxxxxxxxx`。

---

## 10. 判断 Session 是否存在的完整流程

没有单个 API 能一步到位判断「session 存在 + 状态」，需要组合调用：

```
1. WebSocket → isProcessing?
   true  → 正在处理中
   false → 下一步

2. GET /api/sessions/:id/messages → 有消息?
   有消息 → session 存在且已完成
   空     → session 不存在或刚创建
```

### 推荐做法

```bash
# 第一步：通过 WebSocket 看是否正在运行（代码见第 6 节）

# 第二步：查询消息确认是否存在
curl -s "http://localhost:3001/api/sessions/web-s_xxx/messages?projectPath=/my/project" \
  -H "Authorization: Bearer <token>"
# messages 有内容 → 存在且已完成
# messages:[]     → 不存在或无消息
```

---

## 11. Session 完整生命周期

```
创建（POST /api/agent 不传 sessionId）
  │
  ▼
处理中（isProcessing=true）
  │
  ▼
轮询消息（GET /api/sessions/:sessionId/messages）
  │
  ▼
完成（消息不再增加，可以继续提交或导出）
  │
  ▼
延续（POST /api/agent 传入 sessionId）
  │
  ▼
... 重复 ...
```

---

## 12. Session 状态判断总结

没有单个 API 返回 "completed" / "failed" 状态。只能通过以下组合判断：

| 情况 | isProcessing | messages 是否有内容 | 结论 |
|------|-------------|-------------------|------|
| 未提交过 | false | 无 | session 不存在 |
| 正在处理 | true | 可能有（实时增量） | 运行中 |
| 处理完毕 | false | 有 | 已完成 |
| 刚创建未使用 | false | 无 | session 刚创建，尚未跑过 |

**判断完成的唯一可靠方法**：先 WebSocket 确认 `isProcessing=false`，再调消息接口确认有内容。
