# 虚拟现场巡检问题标注

Unity 桌面虚拟机房 → 射线命中放置标记 → HTTP 上报 → Go JSON 持久化 → React 查看并修改状态。不需要手机，不需要模型素材。

## 环境与版本

- 本机编译验证：Unity **6000.6.7f1**，内置渲染管线，传统 Input Manager，IMGUI 表单。
- 注意：本机已有的是 Unity 6.6，未在 6.3 LTS 验证。题目指定 LTS，正式提交前需要在 LTS 编辑器中导入/验证本项目。不要将当前验证结果声称为 LTS 验证。
- Go：本机 1.27.3，模块最低声明 1.22，无第三方依赖。
- Node.js：本机 22.14.0。React / React DOM 19、TypeScript 5、Vite 6；精确依赖见 `admin/package-lock.json`，使用 `npm ci` 重现。
- 所有地址默认用于同一台电脑本地演示。界面中的 Unity UI 使用英文，管理端和文档使用中文。

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

默认后端 `http://localhost:8080`；需要修改时，复制 `.env.example` 为 `.env` 后修改 `VITE_API_BASE_URL`，重新启动 Vite。Go CORS 允许 `http://localhost:5173` 和 `http://127.0.0.1:5173`。端口被占用时会报错，不自动换端口。

```powershell
npm run build
```

该命令先做 TypeScript 检查，再生成生产文件到 `dist/`。

### Unity

1. Unity Hub → Add project from disk → 选择 `unity/` 目录。
2. 打开 `Assets/Scenes/Inspection.unity`。如果场景缺失，可点击菜单 `Inspection → Create or open demo scene`。
3. 点击 Play。机房、三个机柜、管线、碰撞体、玩家和表单由 `InspectionApp.cs` 自动创建，无需手动建模或拖引用。
4. `W/A/S/D` 移动；按住鼠标右键转动视角；左键点击表面创建标记。
5. 填写 Title、Description、Priority，点击 Submit issue。成功后刷新 React。
6. Cancel 会删除未保存标记。输入框、按钮、顶部面板和模态表单不会触发新的场景标注。表单打开时暂停移动和转动。
7. 点击已保存标记查看详情；点击 Reload saved markers 从 Go 恢复标记和最新状态。已解决为绿色，高优先级红色，中优先级橙色，低优先级青色。

顶部可修改 Backend 地址。Unity 使用 `Input` 传统输入 API；如果另建工程迁移脚本，需启用 Input Manager 或 Both。层 8 为可点击表面，层 9 为标记。世界布局固定，不要随意移动场景原点后继续使用原数据。

可使用 `ProjectSetup.BuildWindows` 批处理方法生成 `unity/Build/Inspection.exe`，或使用 Unity Build Profiles。构建目录默认不纳入 Git。

## 接口约定

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
- 使用轻量 IMGUI，无中文字体美化。标记用球体，以命中点为球心，部分球体会嵌入表面；上传坐标为原始 `hit.point`。
- 无登录、分页、删除接口或实时推送。修改状态后 Unity 需主动 Reload。
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
