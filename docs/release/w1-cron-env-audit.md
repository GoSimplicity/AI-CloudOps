# W1：定时任务环境变量存量核查（只读 SQL 草稿）

> **需要用户批准后才能执行。** 本文件只是草稿，编写时没有连接任何数据库，也没有执行过。
> 语句全部是只读的 `SELECT`，不会修改数据。执行前请在只读账号或从库上先确认。

## 背景

新规则（`internal/cron/task/task_env.go`）只对命令行任务（`job_type = 2`）校验 `environment`：

- key（去首尾空白后）和 value 都为空的行：保存前丢弃，执行时跳过（不拒绝）；
- 拒绝：key 为空但 value 非空；
- **变量名白名单格式**：key 去首尾空白后必须匹配 `^[A-Za-z_][A-Za-z0-9_]*$`，否则拒绝。
  这同时挡住 `=`、NUL、`%`（如 `BASH_FUNC_echo%%`）、`-`、`.`、空格、非 ASCII 字母，以及夹带零宽字符的变体（如 `L\u200bD_PRELOAD`）；
- 格式合法后再过拒绝名单（转大写比较）：前缀 `LD_`、`DYLD_`、`BASH_FUNC_`、`MALLOC_`；
  精确名 `BASH_ENV`、`ENV`、`PROMPT_COMMAND`、`IFS`、`SHELLOPTS`、`BASHOPTS`、`PS4`、`GCONV_PATH`、`GLIBC_TUNABLES`、`LOCPATH`。

存量中命中拒绝规则的命令行任务，升级后**执行时会被拒绝**，需要编辑后重新保存。

**这是纵深防御。** 类型 2 任务本身就能在后端主机上执行任意命令（受 `CRON_ALLOW_LOCAL_EXEC` 开关控制），
环境变量校验只是防止“只改环境变量、不改命令”就让既有命令加载任意库或执行额外代码，不能替代对任务创建/编辑权限的控制。

**暂不禁止、但有风险的解释器变量**：`PYTHONPATH`、`NODE_OPTIONS`、`PERL5OPT` 有正当用途（指定依赖目录、内存上限、警告选项），
因此目前允许设置。但它们同样能让 Python/Node/Perl 进程在启动时加载任意模块或执行代码
（例如 `NODE_OPTIONS=--require ...`、`PERL5OPT=-M...`、`PYTHONPATH` 指向可写目录放置 `sitecustomize.py`）。
任务命令调用这些解释器时，请把这些变量与命令本身同等审查。查询 5 可列出存量中设置了这些变量的任务。

## 表结构（已在代码中确认）

| 项 | 值 | 依据 |
| --- | --- | --- |
| 数据库 | MySQL 8.4 | `deploy/release/compose.yaml` 的 `mysql:8.4.11`；`gorm.io/driver/mysql` |
| 表名 | `cl_cron_jobs` | `internal/model/cron_job.go` 中 `(*CronJob).TableName()` |
| 列 | `environment`，`TEXT` | `CronJob.Environment KeyValueList gorm:"type:text"` |
| 存储格式 | JSON 数组 `[{"key":"…","value":"…"}]`；空列表写为 `[]`，也可能是 `NULL` 或空串 | `model.KeyValueList.Value()` 用 `json.Marshal` |
| 类型列 | `job_type`（1 系统 2 命令行 3 HTTP 4 脚本 5 SSH） | `CronJobType` 常量 |
| 软删除 | `deleted_at`，整数，`0` 表示未删除 | `gorm.io/plugin/soft_delete` |

## 查询 1：会被新规则拒绝的命令行任务（只读）

结果只列出任务 ID、名称、状态、违规原因和变量名，**不输出变量值**（值里可能有凭据）。
`env_key_hex` 用于看清零宽字符等不可见字符。

```sql
-- 只读审计。需要用户批准后才能执行。
SELECT j.id,
       j.name,
       j.status,
       e.idx                      AS env_row,
       CASE
         WHEN e.k_trim = '' AND COALESCE(e.v, '') <> ''                       THEN 'key为空但value非空'
         WHEN NOT REGEXP_LIKE(e.k_trim, '^[A-Za-z_][A-Za-z0-9_]*$', 'c')       THEN '变量名格式非法'
         ELSE '拒绝名单'
       END                        AS reason,
       LEFT(REPLACE(e.k_trim, CHAR(0 USING utf8mb4), '\\0'), 64) AS env_key,
       HEX(LEFT(e.k_trim, 64))    AS env_key_hex
FROM cl_cron_jobs AS j
JOIN JSON_TABLE(
       IF(JSON_VALID(j.environment), j.environment, '[]'), '$[*]'
       COLUMNS (
         idx FOR ORDINALITY,
         k   VARCHAR(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin PATH '$.key'   NULL ON EMPTY,
         v   TEXT                                                   PATH '$.value' NULL ON EMPTY
       )
     ) AS e0
JOIN LATERAL (
       SELECT e0.idx,
              e0.v,
              REGEXP_REPLACE(COALESCE(e0.k, ''), '^[[:space:]]+|[[:space:]]+$', '') AS k_trim
     ) AS e
WHERE j.deleted_at = 0
  AND j.job_type = 2
  AND j.environment IS NOT NULL
  AND j.environment NOT IN ('', '[]')
  AND NOT (e.k_trim = '' AND COALESCE(e.v, '') = '')            -- 空行：会被跳过，不算违规
  AND (
        (e.k_trim = '' AND COALESCE(e.v, '') <> '')
     OR (e.k_trim <> '' AND NOT REGEXP_LIKE(e.k_trim, '^[A-Za-z_][A-Za-z0-9_]*$', 'c'))
     OR UPPER(e.k_trim) IN ('BASH_ENV','ENV','PROMPT_COMMAND','IFS','SHELLOPTS','BASHOPTS','PS4',
                            'GCONV_PATH','GLIBC_TUNABLES','LOCPATH')
     OR UPPER(e.k_trim) LIKE 'LD\_%'
     OR UPPER(e.k_trim) LIKE 'DYLD\_%'
     OR UPPER(e.k_trim) LIKE 'BASH\_FUNC\_%'
     OR UPPER(e.k_trim) LIKE 'MALLOC\_%'
  )
ORDER BY j.id, e.idx;
```

## 查询 2：environment 不是合法 JSON 的命令行任务（只读）

这类行在读取时就会出错，需要人工确认。

```sql
-- 只读审计。需要用户批准后才能执行。
SELECT id, name, status, CHAR_LENGTH(environment) AS env_len
FROM cl_cron_jobs
WHERE deleted_at = 0
  AND job_type = 2
  AND environment IS NOT NULL
  AND environment NOT IN ('', '[]')
  AND NOT JSON_VALID(environment);
```

## 查询 3（参考）：含空行的命令行任务数量（只读）

这些空行不会被拒绝，只会在下次保存时被丢弃、执行时跳过。这条查询仅用于了解影响面。

```sql
-- 只读审计。需要用户批准后才能执行。
SELECT COUNT(DISTINCT j.id) AS jobs_with_blank_rows
FROM cl_cron_jobs AS j
JOIN JSON_TABLE(IF(JSON_VALID(j.environment), j.environment, '[]'), '$[*]'
       COLUMNS (k VARCHAR(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin PATH '$.key'   NULL ON EMPTY,
                v TEXT                                                   PATH '$.value' NULL ON EMPTY)) AS e
WHERE j.deleted_at = 0
  AND j.job_type = 2
  AND REGEXP_REPLACE(COALESCE(e.k, ''), '^[[:space:]]+|[[:space:]]+$', '') = ''
  AND COALESCE(e.v, '') = '';
```

## 查询 4：变量名不符合白名单格式的命令行任务（只读）

新增的白名单格式可能拒掉存量中“看起来合法”的变量名（如 `MY-VAR`、`app.mode`、带中文或零宽字符的 key）。
这条查询只看格式，不看拒绝名单，用于评估白名单对存量的影响。

```sql
-- 只读审计。需要用户批准后才能执行。
SELECT j.id,
       j.name,
       j.status,
       e.idx                   AS env_row,
       LEFT(REPLACE(e.k_trim, CHAR(0 USING utf8mb4), '\\0'), 64) AS env_key,
       HEX(LEFT(e.k_trim, 64)) AS env_key_hex
FROM cl_cron_jobs AS j
JOIN JSON_TABLE(
       IF(JSON_VALID(j.environment), j.environment, '[]'), '$[*]'
       COLUMNS (
         idx FOR ORDINALITY,
         k   VARCHAR(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin PATH '$.key' NULL ON EMPTY
       )
     ) AS e0
JOIN LATERAL (
       SELECT e0.idx,
              REGEXP_REPLACE(COALESCE(e0.k, ''), '^[[:space:]]+|[[:space:]]+$', '') AS k_trim
     ) AS e
WHERE j.deleted_at = 0
  AND j.job_type = 2
  AND e.k_trim <> ''
  AND NOT REGEXP_LIKE(e.k_trim, '^[A-Za-z_][A-Za-z0-9_]*$', 'c')
ORDER BY j.id, e.idx;
```

## 查询 5（参考）：设置了 PYTHONPATH / NODE_OPTIONS / PERL5OPT 的命令行任务（只读）

这些变量目前允许，结果仅用于人工复核风险，不输出变量值。

```sql
-- 只读审计。需要用户批准后才能执行。
SELECT j.id, j.name, j.status, e.idx AS env_row, UPPER(e.k_trim) AS env_key
FROM cl_cron_jobs AS j
JOIN JSON_TABLE(
       IF(JSON_VALID(j.environment), j.environment, '[]'), '$[*]'
       COLUMNS (
         idx FOR ORDINALITY,
         k   VARCHAR(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin PATH '$.key' NULL ON EMPTY
       )
     ) AS e0
JOIN LATERAL (
       SELECT e0.idx,
              REGEXP_REPLACE(COALESCE(e0.k, ''), '^[[:space:]]+|[[:space:]]+$', '') AS k_trim
     ) AS e
WHERE j.deleted_at = 0
  AND j.job_type = 2
  AND UPPER(e.k_trim) IN ('PYTHONPATH', 'NODE_OPTIONS', 'PERL5OPT')
ORDER BY j.id, e.idx;
```

## 说明与局限

- `JSON_TABLE` 的输入统一写成 `IF(JSON_VALID(j.environment), j.environment, '[]')`：MySQL 不保证 `WHERE` 中的
  `JSON_VALID` 先于 `JSON_TABLE` 求值，非法 JSON 会让整条查询报错；非法 JSON 的行由查询 2 单独列出。
- `k` 列使用 `COLLATE utf8mb4_bin`，按码点比较，避免默认排序规则把全角字母、带重音字母或尾随空格视为相等；
  拒绝名单比较前已显式 `UPPER()`，与 Go 端“转大写后比较”一致。
- 去首尾空白使用 `REGEXP_REPLACE(k, '^[[:space:]]+|[[:space:]]+$', '')`。Go 的 `strings.TrimSpace` 去掉的是 Unicode 空白，
  ICU 的 `[[:space:]]` 与之基本一致，但个别罕见字符可能有差异，结果以运行时为准。
- `REGEXP_LIKE` 使用 `'c'`（区分大小写）标志，配合 `utf8mb4_bin`，`[A-Za-z]` 只匹配 ASCII 字母，与 Go 正则一致。
- HTTP、SSH、脚本任务不在审计范围内：新规则不校验它们的 `environment`。SSH 使用独立的 `ssh_environment` 列。
- 查询不输出 `value`，结果可以直接贴进工单；但仍应按内部数据处理。
