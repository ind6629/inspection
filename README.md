# 虚拟现场巡检问题标注

Unity 桌面虚拟机房 → 射线命中放置标记 → HTTP 上报 → Go JSON 持久化 → React 查看并修改状态。不需要手机，不需要模型素材。

## 环境与版本

- 本机编译验证：Unity **6000.6.7f1**，内置渲染管线，传统 Input Manager，IMGUI 表单。
- 注意：本机已有的是 Unity 6.6，未在 6.3 LTS 验证。题目指定 LTS，正式提交前需要在 LTS 编辑器中导入/验证本项目。不要将当前验证结果声称为 LTS 验证。
- Go：本机 1.27.3，模块最低声明 1.23，巡检员密码使用 golang.org/x/crypto/bcrypt（依赖锁定见 go.mod/go.sum）。
- Node.js：本机 22.14.0。React / React DOM 19、TypeScript 5、Vite 6；精确依赖见 `admin/package-lock.json`，使用 `npm ci` 重现。
- 所有地址默认用于同一台电脑本地演示。Unity 操作说明、表单、按钮、机柜编号、状态和反馈均使用中文，管理端和文档使用中文。
- Unity 从系统加载 Microsoft YaHei / SimHei / Noto Sans CJK SC 等中文字体，不分发 Windows 字体文件；本机已安装微软雅黑，其他电脑需有上述支持中文的字体之一。

## 启动

先启动 Go，再启动管理端，最后打开 Unity。根目录提供 `start-backend.cmd` 和 `start-admin.cmd`，双击可分别启动；关闭对应窗口即停止服务。

### Go

```powershell
cd backend
go run .
```

默认监听 `http://127.0.0.1:8080`。默认数据文件 `backend/data/issues.json`，路径相对于启动工作目录。重启时保持同一个数据路径。

可通过 `ADDR` 和 `DATA_FILE` 环境变量覆盖。写入失败返回 500，不更新内存；数据文件格式损坏会阻止服务启动，不静默清空数据。使用临时文件、Sync、替换正式文件；单进程互斥锁保护读写，不支持多个服务进程同时写同一文件。

### React

```powershell
cd admin
npm ci
npm run dev
```

浏览器访问 `http://localhost:5173`。支持列表、详情、标题搜索、状态筛选、修改状态、网络错误及重试。

管理端新增中文登录页：默认账号 `admin`，首次启动 Go 后随机密码写入 `backend/data/admin-password.txt`，请在本机读取。该文件和业务数据都被 Git 忽略，不打包公开。可通过 `ADMIN_USERNAME` / `ADMIN_PASSWORD` 环境变量配置，密码至少 6 个字符；不要将真实凭据写进源码。密码文件已存在时沿用，不会每次启动变化。

会话为服务端内存记录和 HttpOnly、SameSite=Strict Cookie，有效期 8 小时，刷新页面保留，退出或 Go 重启后失效。登录失败 10 次后在当前一分钟窗口内限制重试。本地 HTTP 下 Cookie 无 Secure 属性，HTTPS 下自动开启；该实现用于本地演示，不包含管理员注册、找回密码或公网部署配套。

新增接口：`POST /api/auth/login`（username/password）、`GET /api/auth/me`、`POST /api/auth/logout`、`GET /api/admin/issues`。管理端列表和状态修改需要管理员登录；Unity 的 `GET/POST /api/issues` 现在要求巡检员 Bearer 登录凭据，列表仅返回本人记录。旧版无登录 Unity 程序已不再支持提交，请使用 BuildUsers 新版本。浏览器通过 credentials=include 携带 Cookie，并校验管理写操作 Origin。

默认 API 地址自动使用网页当前主机名加 8080 端口，避免 localhost / 127.0.0.1 混用导致 Cookie 不发送。手动设置 VITE_API_BASE_URL 时也请保持主机名一致。

默认后端 `http://localhost:8080`；需要修改时，复制 `.env.example` 为 `.env` 后修改 `VITE_API_BASE_URL`，重新启动 Vite。Go CORS 允许 `http://localhost:5173` 和 `http://127.0.0.1:5173`。端口被占用时会报错，不自动换端口。

```powershell
npm run build
```

该命令先做 TypeScript 检查，再生成生产文件到 `dist/`。

### Unity

1. Unity Hub → Add project from disk → 选择 `unity/` 目录。
2. 打开 `Assets/Scenes/Inspection.unity`。如果场景缺失，可点击菜单 `Inspection → Create or open demo scene`。
3. 点击 Play。机房、三个机柜、管线、碰撞体、玩家和表单由 `InspectionApp.cs` 自动创建，无需手动建模或拖引用。
4. 使用巡检员账号登录后，`W/A/S/D` 移动；按住鼠标右键转动视角；左键点击表面创建标记。
5. 填写问题标题、问题描述和优先级，点击“提交问题”。成功后刷新 React。
6. “取消填写”会删除未保存标记。输入框、按钮、顶部面板和模态表单不会触发新的场景标注。表单打开时暂停移动和转动。
7. 点击已保存标记查看详情；点击“刷新已保存的问题标记”从 Go 恢复标记和最新状态。已解决为绿色，高优先级红色，中优先级橙色，低优先级青色。

顶部可修改后端地址。Unity 使用 `Input` 传统输入 API；如果另建工程迁移脚本，需启用 Input Manager 或 Both。层 8 为可点击表面，层 9 为标记。世界布局固定，不要随意移动场景原点后继续使用原数据。

可使用 `ProjectSetup.BuildWindows` 批处理方法生成 `unity/BuildRegister/Inspection.exe`，或使用 Unity Build Profiles。构建目录默认不纳入 Git。`Build/` 是粉色材质旧版，`BuildFixed/` 是英文修复版，请运行 `BuildRegister/` 支持注册的新版本。

场景材质明确引用 `Resources/SiteSurface.mat` 和项目内置 `SiteSurface.shader`，不依赖运行时 Primitive 的默认材质。可传入 `-inspection-preview <绝对PNG路径>` 在运行两秒后截图，用于实际图形构建检查。

## 接口约定

### 巡检员登录与发现人分类

- 启动后端时首次生成两个演示账号 `inspector01` / `inspector02`，显示名为“巡检员1”/“巡检员2”。本机随机密码见 `backend/data/inspector-credentials.txt`，账号 ID、姓名及 bcrypt 密码哈希保存于 `backend/data/inspectors.json`；两者都在 Git 忽略目录内。重启沿用账号，不重新生成密码。管理员账号不能在巡检员登录入口使用。
- Unity 新版程序：`unity/BuildRegister/Inspection.exe`。先登录，再进入现场；顶部显示当前巡检员，可以退出/切换。切换时清除前一用户的标记和未提交内容，重新加载当前用户自己的记录。登录前不能移动或标注。启动后默认显示登录页，不在磁盘保存 Token 或巡检员密码。
- `POST /api/inspector/login`：`{"username":"inspector01","password":"<本机密码>"}`，返回 `{token,user:{id,username,displayName},expiresAt}`。会话有效期 8 小时，Go 重启后需要重新登录；退出接口为 `POST /api/inspector/logout`，身份检查为 `GET /api/inspector/me`。
- 巡检员请求头：`Authorization: Bearer <token>`。创建和读取 `/api/issues` 均需登录；读取仅返回本人数据。巡检员不能调用管理端接口或修改处理状态。管理员仍使用独立 HttpOnly Cookie 会话。
- 新 Issue 增加只读 `reporter:{id,username,displayName}`。发现人完全由后端登录身份产生；创建请求中传入 reporter 会被拒绝，不能冒充其他用户。姓名按上报时保存快照；旧问题 `reporter=null`，显示“历史记录（未记录发现人）”，不会猜测或伪造原始发现人。
- React 增加“发现人”列、详情中的账号姓名；在“问题清单”标题旁提供“发现人”下拉框，选择巡检员后筛选列表。零上报的巡检员也在选项中显示，支持全部发现人和历史记录，可与状态、标题搜索组合。`GET /api/admin/inspectors` 仅管理员可用，只返回 ID/账号/姓名，不返回密码或哈希。
- 当前预置两位演示巡检员，并支持 Unity 自助注册；尚无管理员用户管理/找回密码页面。不要公开本机生成的密码文件。

### 巡检员自助注册

运行 `unity/BuildRegister/Inspection.exe`，在登录页点击“没有账号？注册巡检员”，填写姓名、账号、密码和确认密码。注册成功自动登录进入现场；如自动登录失败，可使用刚注册的账号重新登录。

账号为3～32位字母、数字、下划线，字母开头，服务端统一转小写；姓名1～40个字符；密码至少6个字符且 UTF-8 不超过72字节。重名返回409，注册参数错误返回400。管理端保留独立管理员账号，不允许注册时指定管理员角色。

`POST /api/inspector/register` 请求 `{username,displayName,password}`，成功201返回 `{id,username,displayName}`，不返回密码或哈希。用户通过 bcrypt 哈希后写入 `data/inspectors.json`，采用临时文件和替换保存，失败不修改内存；每分钟最多处理5次有效格式的新账号注册尝试。新用户在管理员点击刷新后进入发现人下拉选项，尚无上报时也可选择。

自主注册的密码不会写入演示凭据文件，请自行记住。该开放注册用于本地演示，尚无邀请码、审核、找回密码或公网防滥用配套。

### 自动集成检查（含注册）

```powershell
cd backend
$env:INSPECTION_UNITY_EXE='<绝对路径>/unity/BuildRegister/Inspection.exe'
go test -v -run 'TestUnityPlayer(Integration|Registration)' .
```

该测试启动独立临时 HTTP 服务，使用测试专用账号和临时数据，通过真实 Windows 程序完成登录→射线命中→上报→恢复自己的标记，并检查持久化的 reporter。不会读取真实巡检员密码或写入演示服务的数据。普通 `go test ./...` 不设置该变量时跳过这个可选检查。

`TestUnityPlayerRegistration` 额外验证真实 Windows 程序注册→自动登录→上报→发现人持久化，整个过程同样只写临时测试数据。

如果直接运行 Windows 程序的 `-inspection-smoke` 模式，需用环境变量 `INSPECTION_SMOKE_USER`、`INSPECTION_SMOKE_PASSWORD`、`INSPECTION_SMOKE_BACKEND` 提供测试账号和独立后端；不要把真实密码写在命令行参数或提交到源码。

所有业务字段与题目一致：`id,title,description,priority,status,position{x,y,z},createdAt,updatedAt`。时间使用 UTC RFC3339，ID 使用 128 位随机值的十六进制表示。

| 方法 | 路径 | 成功响应 |
|---|---|---|
| GET | `/health` | 200，`{"status":"ok"}` |
| POST | `/api/issues` | 201，完整 Issue |
| GET | `/api/issues` | 200，Issue 数组（空时 `[]`） |
| PATCH | `/api/issues/{id}/status` | 200，修改后的完整 Issue |

创建请求：

```json
{"title":"1号机柜温度异常","description":"右侧温度偏高","priority":"high","position":{"x":1.25,"y":0.8,"z":2.4}}
```

后端生成 ID、时间，默认 `status=open`。标题裁剪空白后必填；优先级仅 `low/medium/high`；位置的三个数值必须完整。状态请求：

```json
{"status":"in_progress"}
```

状态仅 `open/in_progress/resolved`。参数错误 400，不存在 404，方法错误 405，保存失败 500。错误结构：`{"error":{"message":"具体原因"}}`。请求体限制 1 MiB，不接受未知字段及多段 JSON。

## 验证与验收

后端运行 `go test -v ./...` 和 `go vet ./...`。测试覆盖创建、ID、默认状态、坐标、状态更新、重新加载持久化数据、非法枚举/标题/位置/JSON、CORS、保存失败及损坏文件。

Unity 可选自动集成检查：在运行中的测试后端前启动 Windows 程序并传入 `-batchmode -nographics -inspection-smoke -logFile smoke.log`。该模式会真的写入一条标题为 `Unity runtime integration check` 的问题；请使用独立 DATA_FILE。日志成功标记为 `INSPECTION_SMOKE_PASS`，失败退出码 1。该检查覆盖运行时场景生成、物理射线、UnityWebRequest、后端保存和重新加载标记，不代替鼠标交互验收。

完整手工验收：

1. 移动并查看不同设备；点击机柜正面、侧面和地面，核实标记位置。
2. 点击 UI 不落标；按住鼠标不会连续落标；表单打开时不能新建第二个待提交标记；取消后标记消失。
3. 空标题被拒绝；正常提交后 React 刷新可见，标题/优先级/坐标一致。
4. React 修改状态；重启 Go 后问题和状态保留；Unity Reload 后标记颜色更新。
5. 停止 Go，Unity 提交和 React 刷新都显示错误；Unity 输入内容保留，可取消或重试。

## 完成情况和已知限制

- 已实现：三端源码、固定场景、移动观察、命中位置、单个待提交标记、取消、上报反馈、持久化、管理端状态更新和错误提示。
- 扩展：恢复已保存标记、点击标记详情、按优先级/状态着色、管理端搜索/状态筛选、后端测试。
- Unity 编译、场景生成和 Windows 构建已通过；React 类型检查和生产构建已通过；Go 普通测试与 vet 已通过。
- Unity Windows 程序自动集成检查已通过，日志确认射线命中 → UnityWebRequest → Go 保存 → 恢复标记；React 浏览器实际读取该问题并改为 resolved，停止并重新启动 Go 后刷新仍保留该状态。
- 当前机器 Go 的 race 模式出现系统退出码 `0xc0000139`，未通过 race 环境检查；普通测试正常，不宣称 race 检查通过。
- 窗口截图自动化连续超时，尚未完成逐项鼠标交互验收；不能把自动集成检查当作人工操作录像。
- 未完成：LTS 编辑器验证、两段 3～5 分钟演示视频、上传 GitHub/Gitee/网盘获得评审访问链接。当前仅本地交付。
- 使用轻量 IMGUI 和系统中文字体。标记用球体，以命中点为球心，部分球体会嵌入表面；上传坐标为原始 `hit.point`。
- 已提供管理员/巡检员两类权限和发现人分类；尚无分页、删除接口、用户管理页面或实时推送。修改状态后 Unity 需主动刷新标记。
- 网络超时可能发生在服务端已保存之后：重试前到管理端核实，当前未实现幂等去重；不要连续重试。
- 不提供跨真实 AR 会话定位，也不支持多进程同时写 JSON。当前用于本地考核演示。

## AR 基础理解

1. 迁移到真实 AR：通常需要 ARSession、XR Origin、ARPlaneManager 检测平面，ARRaycastManager 将屏幕点击转换成真实可跟踪表面的命中位姿，必要时使用 ARAnchorManager 创建锚点。普通 Physics.Raycast 主要检测已有 Collider，并不能自行检测真实平面。
2. 只保存 x/y/z 不足：AR 坐标相对于本次会话的原点和方向，下次启动可能改变。需要平台支持的持久化锚点/空间地图恢复、重新定位，或通过可识别参照物建立一致坐标系；普通临时锚点本身不保证跨启动恢复。
3. 初始化/Tracking 异常：区分设备不支持、权限未授予、初始化中、跟踪受限等状态，给出明确文字及重试入口。跟踪不可靠时暂停放置，提示改善光照、移动设备扫描环境，避免把不可靠位置保存成确定结果。

参考：[UnityWebRequest 官方文档](https://docs.unity.com/en-us/engine/6000.3/script-reference/unityengine/networking/unitywebrequest)、[AR Foundation managers](https://docs.unity.cn/Packages/com.unity.xr.arfoundation%404.1/manual/trackable-managers.html)、[本机 Editor 发布信息](https://unity.com/releases/editor/whats-new/6000.6.7f1)。

## 视频与提交

第一段录制现场移动、命中标注、UI 拦截、取消和 Unity 提交，在 React 刷新查看。第二段录制修改状态、Go 重启数据保留、停止后端的错误提示和恢复。每段 3～5 分钟，按备选题用 Editor 或 Windows 程序演示，无需手机。

上传完整源码、README、AI_LOG 和视频。Unity 保留 Assets（含 .meta）、Packages、ProjectSettings，排除 Library/Temp/Logs 等。保留 Git 历史，发布前填写真实访问链接及权限/有效期；不得填写虚构链接或提交凭据。
