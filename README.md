# 虚拟现场巡检问题标注

Unity 虚拟机房 → 点击实际命中位置放置问题标记 → Go 保存 → React 管理端查看与更新状态。支持电脑运行和 Android 虚拟场景，不要求 AR 设备或外部模型。

## 项目结构

```text
admin/                  React + TypeScript 管理端
backend/                Go API 与 JSON 持久化
unity/                  Unity 工程（Assets、Packages、ProjectSettings）
android/INSTALL.md      安卓安装与连接说明
AR基础理解.md            真实 AR 迁移基础说明
AI_LOG.md               AI 使用记录
build-android.ps1       Android 构建脚本
start-backend.cmd       本机后端启动脚本
start-backend-lan.cmd   局域网后端启动脚本
start-admin.cmd         管理端启动脚本
```

本地最新 Windows 程序在 `windows/Inspection.exe`，已删除登录框下方的说明文字；请保留整个 windows 目录。已有安卓包为 `android/Inspection-Android-1.0.1.apk`。这些构建产物被 Git 忽略，克隆源码不会包含它们；可自行构建或从下方 Releases 下载。

## 下载程序

发布页：[v1.0.1](https://github.com/ind6629/inspection/releases/tag/v1.0.1)。

- [Windows 完整程序 ZIP](https://github.com/ind6629/inspection/releases/download/v1.0.1/Inspection-Windows-1.0.1.zip)：完整解压后运行 Inspection-Windows/Inspection.exe，勿单独移动 exe。
- [Android APK](https://github.com/ind6629/inspection/releases/download/v1.0.1/Inspection-Android-1.0.1.apk)：适用于 Android 8.0+、ARM64，演示调试签名。
- [SHA256 校验文件](https://github.com/ind6629/inspection/releases/download/v1.0.1/SHA256SUMS.txt)。

程序需要连接独立运行的 Go 后端，安装包不会自动启动后端或管理端。手机使用电脑可达的局域网地址。两端账户、问题数据均由同一后端保存。

## 环境与版本

| 部分 | 使用版本 / 要求 |
| --- | --- |
| Unity | 本机工程为 6000.6.7f1，内置渲染管线、传统 Input Manager、C# |
| 管理端 | React 19、TypeScript 5、Vite 6；Node.js 本机 22.14.0 |
| 后端 | Go 模块最低 1.23.0，本机曾使用 1.27.3；标准库 HTTP、JSON 文件、bcrypt |
| Android | Android 8.0 / API 26 起、ARM64、横屏；构建需 Android Build Support、SDK、NDK、OpenJDK |

题目要求 Unity LTS；当前工程尚未在 LTS 编辑器验证，应在正式验收前补充，不将当前编辑器版本声称为已验证的 LTS。依赖精确版本见 package-lock.json、go.mod/go.sum 和 Unity Packages。

中文字体采用随工程提供的 Noto Sans CJK SC，字体与许可位于 `unity/Assets/Resources/Fonts/`。

## 启动方式（Windows / VS Code）

在 VS Code 中打开本项目根目录。先启动后端，再启动管理端，两个终端都保持运行。

### 1. Go 后端

终端一从项目根目录执行：

```powershell
cd backend
go run .
```

注意 `run` 与 `.` 之间有空格。默认地址为 `http://127.0.0.1:8080`，打开 `http://localhost:8080/health` 应返回 `{"status":"ok"}`。

如果需要手机访问，在启动前设置监听地址：

```powershell
$env:ADDR = "0.0.0.0:8080"
go run .
```

也可从项目根目录运行 `.\start-backend.cmd` 或 `.\start-backend-lan.cmd`。

### 2. React 管理端

新建终端二，从项目根目录执行：

```powershell
cd admin
npm.cmd ci
npm.cmd run dev
```

打开 `http://localhost:5173/`。首次运行或锁文件更新后执行 `npm.cmd ci`；后续可直接启动。也可从根目录运行 `.\start-admin.cmd`。

生产构建：

```powershell
npm.cmd run build
```

前端默认请求网页当前主机的 8080 端口。自定义地址时复制 `admin/.env.example` 为 `.env`，设置 `VITE_API_BASE_URL` 后重启 Vite。前后端主机名保持一致，避免 localhost 与 127.0.0.1 混用影响 Cookie。Go 默认允许这两个本机地址的 5173 端口跨域访问。

### 3. Unity 客户端

1. Unity Hub 添加已有工程，选择 `unity/` 文件夹，使用对应编辑器打开。
2. 打开 `Assets/Scenes/Inspection.unity`；场景缺失时选择菜单 `Inspection → Create or open demo scene`。
3. 点击 Play，使用巡检员账户登录；也可在客户端注册账户。
4. `W/A/S/D` 移动，按住鼠标右键转动，鼠标左键点击设备或表面标注。
5. 填写标题、描述和优先级并提交，在管理端刷新查看。

场景由基础几何体和脚本生成，不需要手动建模。UI 输入不应放置标记；表单打开时暂停场景操作；取消移除未保存标记。点击已保存标记可查看详情，主动刷新可恢复本人标记及最新状态。

直接运行已有 Windows 程序：在根目录执行 `.\windows\Inspection.exe`。源码仓库不包含此目录时，使用 Unity 的 Build Profiles 构建 Windows。工程也提供 `ProjectSetup.BuildWindows`，其默认输出为 `unity/BuildLocationPassword/`，与本地整理后的 windows 分发目录不同。

### 4. Android 手机

手机和电脑处于可互通的局域网，启动局域网后端，在手机浏览器验证 `http://<电脑IPv4地址>:8080/health`，再在客户端填写相同后端地址。手机不能使用 localhost 连接电脑。APK 不包含 Go 后端，电脑服务必须独立运行。

手机使用左下方向键移动、拖动场景观察、轻点表面标注。登录及表单使用系统键盘。详见 [安卓安装说明](android/INSTALL.md)。

在项目根目录构建：

```powershell
.\build-android.ps1 -BackendUrl 'http://<电脑IPv4地址>:8080'
```

构建脚本生成 `android/Inspection-Android.apk`。已有 1.0.1 为本地调试签名版本，版本代码 2；不是应用商店发行配置。构建脚本支持系统代理及可选本地 Maven 缓存，不修改系统代理。构建日志可能含本机环境信息，不应直接上传。

## 账户与权限

- 管理员通过 React 登录，查看所有问题、更新状态和管理巡检员。
- 首次启动后端默认生成管理员初始密码文件 `backend/data/admin-password.txt`；可使用 ADMIN_USERNAME、ADMIN_PASSWORD 环境变量配置。已有本地密码不随源码分发，请勿把实际密码写进 README。
- 巡检员可在 Unity 注册、登录、退出和修改自己的密码。预置演示巡检员初始凭据仅保存在本机 `backend/data/inspector-credentials.txt`。修改密码后初始凭据文件不会同步更新。
- 管理员可创建巡检员、修改姓名、启停账户和重置密码。只管理巡检员，不开放管理员注册。
- 管理员使用 HttpOnly Cookie，巡检员使用 Bearer Token。会话有效期 8 小时，Go 重启需要重新登录；巡检员改密后撤销其全部会话。
- 发现人由后端登录身份确定，客户端不能指定其他人；历史问题保留上报时的姓名快照。

## 已实现功能

- 固定机房场景、三座机柜与管线、移动和视角控制、实际射线命中标注。
- 标题必填、优先级校验、单个待提交标记、取消清理、成功失败提示和 UI 点击隔离。
- JSON 持久化、恢复本人标记、标记详情及按优先级和处理状态着色。
- 管理端问题清单、详情、搜索、发现人下拉筛选、区域与状态筛选、状态更新和网络错误提示。
- 管理员与巡检员权限、注册、账户管理、自助改密及管理员重置密码。
- 问题记录设备编号、所属区域和发现人。CAB-001/002 为机房A区，CAB-003 为机房B区，PIPE-001 为机房公共区；旧数据缺字段时显示未记录。

## 数据与接口

默认数据目录为启动工作目录下的 `data/`；从 backend 目录启动时是 `backend/data/`。问题保存在 issues.json，巡检员账户保存为 bcrypt 哈希。DATA_FILE 可覆盖问题文件路径；重启时保持相同工作目录和配置。不要公开本机 data 目录。

文件通过临时文件与替换写入；保存失败返回错误，损坏文件不会静默重置。不支持多进程同时写入同一份 JSON。

| 方法 | 路径 | 说明 / 权限 |
| --- | --- | --- |
| GET | `/health` | 服务健康检查 |
| POST | `/api/auth/login` | 管理员登录，username/password |
| GET / POST | `/api/auth/me` / `/api/auth/logout` | 管理员身份 / 退出 |
| POST | `/api/inspector/register` | 注册，username/displayName/password |
| POST | `/api/inspector/login` | 巡检员登录，返回 token、user、expiresAt |
| GET / POST | `/api/inspector/me` / `/api/inspector/logout` | 巡检员身份 / 退出 |
| POST | `/api/inspector/password` | 本人改密，currentPassword/newPassword |
| GET / POST | `/api/issues` | 巡检员读取本人问题 / 创建问题 |
| GET | `/api/admin/issues` | 管理员读取全部问题 |
| PATCH | `/api/issues/{id}/status` | 管理员更新，参数 status |
| GET | `/api/admin/inspectors` | 管理员读取发现人筛选选项 |
| GET / POST | `/api/admin/users` | 管理员查看 / 创建巡检员 |
| PATCH | `/api/admin/users/{id}` | 修改 displayName 或 disabled |
| POST | `/api/admin/users/{id}/password` | 管理员重置密码，参数 password |

创建问题示例（需巡检员 Bearer Token）：

```json
{
  "title": "1号机柜温度异常",
  "description": "机柜右侧温度明显偏高",
  "priority": "high",
  "position": {"x": -4, "y": 0.8, "z": 2.4},
  "deviceId": "CAB-001",
  "area": "机房A区"
}
```

成功返回完整 Issue：id、title、description、priority、status、position、createdAt、updatedAt，以及 reporter、deviceId、area 扩展字段。后端生成唯一 ID 和 UTC 时间，默认状态 open。priority 仅允许 low/medium/high；status 仅允许 open/in_progress/resolved。设备与区域匹配由后端校验。

创建成功 201，读取和更新成功 200；参数错误 400，未认证 401，无权限 403，不存在 404，冲突 409，请求过多 429，保存失败 500。错误响应格式为 `{"error":{"message":"具体原因"}}`。请求体限制 1 MiB，不接受未知字段。管理写操作还校验允许的 Origin。

## 验证与已知限制

- 此前已完成 Go、React 构建和 Windows 程序集成验证，以及 Android IL2CPP ARM64 构建、签名和清单检查；这些历史结果不等同于最新状态下重新完整验收。
- 按用户要求，后端独立测试文件已删除，当前源码不包含原先的 *_test.go 测试；不能再用 go test 的无测试输出作为功能验证证据。
- Android 系统键盘、实际触控手感及用户修复后的视角问题，仍需在实际交付设备上复测；本 README 不宣称已完成独立真机验收。
- 尚需补齐 Unity LTS 验证、两段各 3～5 分钟演示视频及最终交付链接。
- 尚无分页、删除接口、实时推送、跨真实 AR 会话定位和公网运维配套。
- 标记以实际 hit.point 为球心，部分球体会嵌入表面。世界布局固定，更改原点或设备布局前需考虑已存坐标。
- 请求超时可能发生在服务端保存之后，目前无幂等去重；重试前先在管理端核实。

手工验收：登录并移动观察 → 点击设备标注 → 检查 UI 不误触及取消清理 → 上报 → React 刷新查看和修改状态 → 重启 Go 后重新登录，核实问题和状态保留 → 停止后端，检查两端错误提示。

## AR 基础理解

1. **平面检测与放置：**使用 ARSession、XR Origin 和对应平台插件，通过 ARPlaneManager 检测真实平面，ARRaycastManager 获取屏幕点击命中位姿，在命中处创建标记；需要稳定跟踪时使用 ARAnchorManager。普通 Physics.Raycast 不会自行检测真实平面。
2. **坐标跨启动恢复：**x/y/z 依赖本次会话原点与朝向，下次会话可能变化。需要平台支持的持久化锚点、空间地图与重新定位，或通过可识别参照物建立一致坐标系；临时锚点不保证重启后恢复。
3. **异常提示：**区分不支持、权限不足、初始化失败和跟踪不稳定，提供明确提示、重试或返回入口；跟踪异常时暂停标注，引导改善光照并缓慢扫描，恢复后再允许放置。

详见 [AR 基础理解](AR基础理解.md) 及其中的 Unity 官方参考资料。当前项目未实现真实 AR。

## 提交与演示

项目介绍视频：[下载 MP4](https://github.com/ind6629/inspection/releases/download/v1.0.1/project-introduction.mp4)。视频作为 Release 附件保存，不进入源码 Git 历史。该链接用于分享用户提供的视频，本次未审核其是否完整覆盖题目要求的两段演示。

源码应保留完整 Git 历史。Unity 仅提交 Assets（含 .meta）、Packages、ProjectSettings；排除 Library、Temp、Logs、构建目录以及 node_modules、.env 和 backend/data。Windows、APK、演示视频可作为 Release 附件另行发布。

第一段视频展示移动观察、命中标注、表单、取消与上报以及 React 查看；第二段展示状态修改、重启持久化和网络错误处理。每段 3～5 分钟，备选题可使用 Editor 或桌面虚拟场景，无需真实 AR。

AI 使用记录见 [AI_LOG.md](AI_LOG.md)。优化版提示词与最初发送的原文存在区别，正式提交前应结合真实会话核实顺序、追问纠错与验证结果。
