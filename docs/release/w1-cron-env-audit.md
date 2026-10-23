# W1：定时任务环境变量存量核查（只读 SQL 草稿）

> **需要用户批准后才能执行。** 本文件只是草稿，编写时没有连接任何数据库，也没有执行过。
> 语句全部是只读的 `SELECT`，不会修改数据。执行前请在只读账号或从库上先确认。

## 背景

新规则（`internal/cron/task/task_env.go`）只对命令行任务（`job_type = 2`）校验 `environment`：

- key（去首尾空白后）和 value 都为空的行：保存前丢弃，执行时跳过（不拒绝）；
- 拒绝：key 为空但 value 非空；key 含 `=` 或 NUL；
- 拒绝名单（key 去首尾空白后转大写比较）：前缀 `LD_`、`DYLD_`、`BASH_FUNC_`、`MALLOC_`；
  精确名 `BASH_ENV`、`ENV`、`IFS`、`SHELLOPTS`、`BASHOPTS`、`PS4`、`GCONV_PATH`、`GLIBC_TUNABLES`、`LOCPATH`。

存量中命中拒绝规则的命令行任务，升级后**执行时会被拒绝**，需要编辑后重新保存。

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

```sql
-- 只读审计。需要用户批准后才能执行。
SELECT j.id,
       j.name,
       j.status,
       e.idx                      AS env_row,
       CASE
         WHEN e.k_trim = '' AND COALESCE(e.v, '') <> ''          THEN 'key为空但value非空'
         WHEN LOCATE('=', e.k_trim) > 0
           OR LOCATE(CHAR(0 USING utf8mb4), e.k_trim) > 0         THEN 'key含=或NUL'
         ELSE '拒绝名单'
       END                        AS reason,
       LEFT(REPLACE(e.k_trim, CHAR(0 USING utf8mb4), '\\0'), 64) AS env_key
FROM cl_cron_jobs AS j
JOIN JSON_TABLE(
       j.environment, '$[*]'
       COLUMNS (
         idx FOR ORDINALITY,
         k   VARCHAR(512) PATH '$.key'   NULL ON EMPTY,
         v   TEXT         PATH '$.value' NULL ON EMPTY
       )
     ) AS e0
JOIN LATERAL (
       SELECT e0.idx,
              e0.v,
              TRIM(BOTH '\r' FROM TRIM(BOTH '\n' FROM TRIM(BOTH '\t' FROM TRIM(COALESCE(e0.k, ''))))) AS k_trim
     ) AS e
WHERE j.deleted_at = 0
  AND j.job_type = 2
  AND j.environment IS NOT NULL
  AND j.environment NOT IN ('', '[]')
  AND JSON_VALID(j.environment)
  AND NOT (e.k_trim = '' AND COALESCE(e.v, '') = '')            -- 空行：会被跳过，不算违规
  AND (
        (e.k_trim = '' AND COALESCE(e.v, '') <> '')
     OR LOCATE('=', e.k_trim) > 0
     OR LOCATE(CHAR(0 USING utf8mb4), e.k_trim) > 0
     OR UPPER(e.k_trim) IN ('BASH_ENV','ENV','IFS','SHELLOPTS','BASHOPTS','PS4',
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
JOIN JSON_TABLE(j.environment, '$[*]'
       COLUMNS (k VARCHAR(512) PATH '$.key' NULL ON EMPTY,
                v TEXT         PATH '$.value' NULL ON EMPTY)) AS e
WHERE j.deleted_at = 0
  AND j.job_type = 2
  AND JSON_VALID(j.environment)
  AND TRIM(COALESCE(e.k, '')) = ''
  AND COALESCE(e.v, '') = '';
```

## 说明与局限

- Go 的 `strings.TrimSpace` 会去掉所有 Unicode 空白。SQL 只去掉空格、`\t`、`\n`、`\r`，
  因此 key 带其他罕见空白字符时，结果与运行时可能有细微差异。
- HTTP、SSH、脚本任务不在审计范围内：新规则不校验它们的 `environment`。SSH 使用独立的 `ssh_environment` 列。
- 查询不输出 `value`，结果可以直接贴进工单；但仍应按内部数据处理。
