# 固定三仓源码候选准备

`scripts/release-candidate.py` 只准备源码，不构建镜像、不启动 Docker/服务、不执行测试、不发布 tag。产物状态固定为 `prepared-not-built`，不能作为发布或验收通过的证据；历史 `docs/release/release-candidate.json` 不读取、不覆盖。

## 默认计划与显式准备

```sh
python3 scripts/release-candidate.py
# 仅打印离线计划，不读取凭据、Git 工作树或服务。

python3 scripts/release-candidate.py --prepare \
  --output "$HOME/project/cloudops-candidates/rc-20261030-01"
```

显式准备固定读取脚本所在后端仓库的同级三仓：`AI-CloudOps`、`AI-CloudOps-web`、`AI-CloudOps-aiops`。不接受远程 URL、分支或任意源目录参数，不 fetch/pull/checkout，不 stash/reset/clean，也不处理别人的工作。

每仓必须有符合固定 GitHub 仓库身份的 origin、已提交的完整 40 位 SHA-1 HEAD 和干净工作树。暂存、未暂存与未跟踪的非忽略工作都会拒绝；稀疏/skip-worktree/assume-unchanged 等隐藏 index 状态、源码嵌套仓库、子模块和已跟踪链接也拒绝，不自动改这些状态。忽略的个人 `.env` 可以留在原工作区，但不会进入 Git archive。

三仓全部通过预检后，分别固定完整 SHA，再按这些 SHA 归档。之后 HEAD 移动不会把归档切换成新 HEAD；输出清单记录的是实际选定的三份提交，不自动推测它们已经推送、互相兼容或已经被同一轮 CI 测过。后端 `integration-sources.json` 与这份三仓选择是否一致，须在后续组合验收阶段核对，不能偷偷替换任一仓库。

## 输出与安全边界

输出必须是全新目录，不能在任意 Git 仓库内，也不能覆盖、包住或嵌套进三个源仓库；已存在文件、目录和符号链接均拒绝。必要父目录会创建。外层候选根、`archives/` 和 `sources/` 容器目录保持 `0700`，tar 与清单保持 `0600`；每仓源码根和内部目录为 `0755`，普通 Git 源文件为 `0644`，可执行文件为 `0755`，不保留 setuid 等特殊位。

两层权限有不同职责：外层 `0700` 防止宿主其他用户遍历候选；源码自身的正常读取位保证公开配置作为 bind mount 后，容器内 nobody 等非 root UID 可以读取。将全部源文件设为 `0600` 会使普通 Linux 下官方 Prometheus 等非 root 进程读不到配置，不能靠“更私有”破坏运行兼容性。个人 `.env` 不进入源码候选；后续生成的实际私有配置仍由各工具以 `0600` 独立保存，不随公开源码放宽。宿主权限不代表显式挂入容器后的隔离承诺：Docker daemon 本身拥有读取宿主文件的权限，桌面虚拟化还可能映射 UID，私有元数据/凭据不应挂给无关应用。

```text
NEW_DIRECTORY/
  candidate.json
  archives/
    AI-CloudOps.tar
    AI-CloudOps-web.tar
    AI-CloudOps-aiops.tar
  sources/
    AI-CloudOps/
    AI-CloudOps-web/
    AI-CloudOps-aiops/
```

`candidate.json` 保存固定仓库常量、实际 revision、每个原始 tar 的 SHA-256、相对源码目录及文件数，并明确 build/tests/deployment 均未运行。所有角色完成后才写清单并移除 `INCOMPLETE`；失败保留已创建目录和标记，不能将带 `INCOMPLETE` 的产物当作有效候选，也不会自动重试或清理旧产物。

归档使用无远程配置的受控临时 bare reader，只读源对象库；禁用自动/延迟网络传输、替换对象与本机执行 helper，避免源 `.git/info/attributes` 或机器配置改写归档。不会将 origin 的账号、token、query 或原始 Git 错误写入输出或清单，控制台只显示固定阶段和角色的完整提交 SHA。

提取前核对整个路径集合，拒绝绝对路径、`..`、反斜杠、控制目录、大小写/Unicode 规范化碰撞、外链/硬链接/设备等特殊项。普通文件按选定 tree 的大小、执行位和 Git blob ID 逐项匹配；缺文件、额外文件或内容变化都会失败。单文件最多 64 MiB、源码总量最多 512 MiB、最多 100000 个文件，原始 tar 上限 1 GiB。

本工具要求归档与完整选定 tree 一致。因此已提交的 `export-ignore`/`export-subst` 若省略或变换文件会明确拒绝，不以省略源码换取成功。Git LFS 不自动下载，大对象指针仍只是提交中的普通内容；本阶段没有声称其构建依赖已就绪。

## 个人环境与公开构建配置

已跟踪的个人 `.env`、`.env.*` 及私有备份路径会在归档前被拒绝，不能只从提取结果删掉却留在原始 tar 中。当前 Web 仓库以下四份已跟踪公开构建配置明确保留：

- `apps/web-antd/.env`
- `apps/web-antd/.env.analyze`
- `apps/web-antd/.env.development`
- `apps/web-antd/.env.production`

这不是通用密钥扫描器。其他源码内容仍须经过正常 gitleaks/审查；允许名单不表示可以向这些公开构建配置写入真实秘密。工具不读取原工作区忽略的 `.env` 内容。

## 后续本地隔离候选流程

以下是下一阶段的操作顺序，准备器不会自动执行：

1. 核对 `candidate.json` 没有 `INCOMPLETE`，复核三仓 SHA 与原始 tar SHA-256，以及源码选择和 CI 锁的一致性。确认选定后端包含空/半机器人凭据零 HTTP 修复，不能借旧 CI 功能结果证明隔离边界。
2. 只从 `sources/AI-CloudOps` 调用 `release-env.py` 生成新的私有配置，不拷贝原工作区凭据。模型、AIOps 通知及 Go webhook 的测试配置必须为空凭据/回环端点；禁止带入线上配置。
3. 在该固定源码布局中调用默认离线 `release-fresh.py` 计划，确认后只对全新本地专用项目显式初始化。构建上下文和只读 bind 来自持久候选目录，fresh 的私有快照另在其同父 `.cloudops-fresh/` 保留实际配置与选定 image ID。不要回到正在变化的原工作区构建。
4. 新库迁移、clock、运行角色及实际镜像 ID 核验通过后，才由独立验收步骤执行 API 和[精确五例浏览器门禁](business-browser-ci.md)。不调用真实模型、云、通知或集群。
5. 把后续构建和验收结果写入新的制品/验收记录，保留这份 `prepared-not-built` 源码记录和历史候选；不得先把准备状态改成 validated。生产部署仍由用户安排，不在上述本地流程内。

## 本阶段验证

```sh
python3 -m unittest scripts.tests.test_release_candidate -v
```

上述纯单测仅创建临时 Git 仓库与恶意 tar，不访问 GitHub、Docker 或既有服务。覆盖三仓 SHA/tar 哈希/内容、HEAD 移动后仍固定旧提交、公开 Web env 保留与个人配置排除、dirty/index 隐藏状态拒绝、覆盖与嵌套目录拒绝、token-origin 和异常不回显，以及路径穿越/外链/特殊文件/重复项/错误 blob 拒绝。它证明源码准备合同，不代表新三仓候选已构建或五例浏览器已通过。

另有一次自有唯一标签、network=none、UID 65534 的小容器演练：仅运行读取命令，以实际 file bind 和源码目录 bind 两种方式读取公开 Prometheus 配置，确认修正后的权限可用；不启动 Prometheus 服务，不访问既有容器。该演练不把主动挂载的 private metadata 可读性当作隔离保证；本机 OrbStack 的 UID 映射会影响这一额外假设，host mode 仍由纯测精确核对。夹具仅在核验唯一标签后清理。
