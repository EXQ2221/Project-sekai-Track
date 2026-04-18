# PJSK Track

一个基于 Go + Gin + GORM + MySQL + Redis 的 Project Sekai 成绩追踪项目。  
支持账号鉴权、歌曲/难度查询、成绩上传、Best30 计算、趋势统计、随机推荐与 B30 图片导出；前端为原生 JavaScript + Vite。

## 功能概览

- 用户系统
  - 注册 / 登录 / 刷新令牌
  - 会话管理（登出、登出全部、会话查看与撤销）
  - 个人资料（头像、简介、角色）
- 歌曲与成绩
  - 歌曲列表与多条件筛选
  - 歌曲详情（各难度游玩数、FC/AP 率）
  - 成绩上传 / 删除
- B30 相关
  - Best30 列表（`official` / `const` 两种计算模式）
  - B30 趋势（3 小时桶）
  - B30 图片导出
- 随机推荐
  - 支持 `calc_mode=official|const`
  - `const` 模式候选不足时自动降级到 `official`
  - 仅推荐“目标状态高于当前状态”的谱面

## 技术栈

- Backend: Go, Gin, GORM
- DB: MySQL 8
- Cache: Redis 7
- Frontend: Vanilla JS + Vite
- Deployment: Docker / Docker Compose

## 目录结构

```text
.
├─ cmd/
│  ├─ import_music/        # 导入 musics.json + musicDifficulties.json
│  └─ fetch_covers/        # 下载歌曲封面到 static/assets
├─ internal/
│  ├─ config/              # MySQL / Redis 初始化
│  ├─ handler/             # HTTP 处理层
│  ├─ service/             # 业务层
│  ├─ repository/          # 数据访问层
│  ├─ middleware/          # 鉴权、限流、CORS
│  └─ model/               # GORM 模型
├─ frontend/               # 前端页面（Vite）
├─ static/
│  ├─ assets/              # 歌曲封面
│  ├─ uploads/avatar/      # 用户头像上传目录（运行时）
│  └─ characters/          # 角色图（包含 miku.png）
├─ migrations/             # MySQL 初始化 SQL
├─ musics.json
├─ musicDifficulties.json
├─ docker-compose.yml
├─ Dockerfile
└─ main.go
```

## 本地启动（推荐）

### 1. 准备环境

- Go `1.25+`
- Node.js `18+`
- Docker（用于 MySQL / Redis）

### 2. 配置 `.env`

项目根目录已有 `.env`，可按需修改：

```env
APP_PORT=8080

DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=pjsk
DB_USER=pjsk_user
DB_PASS=pjsk_pass_123456
DB_ROOT_PASS=root_pass_123456
DB_EXPOSE_PORT=3306

REDIS_HOST=127.0.0.1
REDIS_PORT=6379
REDIS_EXPOSE_PORT=6379

# 可选
# JWT_SECRET=replace_me
# JWT_EXPIRE_HOURS=1
# REFRESH_TOKEN_EXPIRE_HOURS=168
# CORS_ALLOWED_ORIGINS=http://localhost:5173
```

### 3. 启动 MySQL / Redis

```bash
docker compose up -d mysql redis
```

### 4. 启动后端

```bash
go run .
```

后端默认监听：`http://localhost:8080`

> 首次启动会自动执行 GORM AutoMigrate，并重建 `music_achievements` 字典数据。

### 5. 导入歌曲数据（必做）

```bash
go run ./cmd/import_music/main.go
```

如需自定义路径：

```bash
go run ./cmd/import_music/main.go -musics ./musics.json -difficulties ./musicDifficulties.json
```

### 6. 启动前端

```bash
cd frontend
npm install
npm run dev
```

前端默认地址：`http://localhost:5173`

## Docker 一键启动（含后端）

```bash
docker compose --profile app up -d --build
```

- `mysql` / `redis` 默认始终启动
- `api` 服务在 `app` profile 下启动

## 封面下载（可选）

```bash
go run ./cmd/fetch_covers/main.go --start-id 130
```

- 下载到 `static/assets/`
- 自动生成 `static/assets/_cover_map.json`

## 主要接口（摘要）

### 公共接口

- `POST /register`
- `POST /login`
- `POST /refresh`
- `GET /characters`
- `GET /musics`
- `GET /musics/:id`

### 鉴权接口（Bearer Token）

- `POST /logout`
- `POST /logout-all`
- `GET /sessions`
- `POST /sessions/revoke`
- `POST /change_pass`
- `GET /me`
- `POST /me/profile`
- `POST /me/character`
- `POST /me/avatar`
- `POST /records`
- `DELETE /records`
- `GET /records/b30?calc_mode=official|const`
- `GET /records/b30/trend?calc_mode=official|const`
- `GET /records/b30/image?calc_mode=official|const`
- `GET /records/statuses`
- `GET /records/achievement-map`
- `GET /records/statistics`
- `GET /random/music?calc_mode=official|const`

## B30 计算规则（当前实现）

- AP：`base`
- FC：`base - 1.5`（若 `base >= 33` 则 `base - 1.0`）
- 其他状态：`base - 5.0`

其中：

- `official` 模式：`base = play_level`
- `const` 模式：`base = const`（若 `const <= 0` 则回退 `play_level`）

## 提交建议

- 仓库已提供定制 `.gitignore`，默认只保留代码、必要 JSON 与 `static/characters/miku.png`
- 建议先在项目根目录初始化 Git 并提交：

```bash
git init
git add .
git commit -m "init: pjsk track project"
```



