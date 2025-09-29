-- JWT认证和RBAC权限检查插件
local core = require("apisix.core")
local jwt = require("resty.jwt")
local redis = require("resty.redis")
local http = require("resty.http")
local json = require("cjson")

local plugin_name = "jwt-auth-rbac"

-- 插件配置模式
local schema = {
    type = "object",
    properties = {
        jwt_secret = {type = "string"},
        redis_host = {type = "string", default = "redis"},
        redis_port = {type = "integer", default = 6379, minimum = 1, maximum = 65535},
        redis_password = {type = "string"},
        redis_timeout = {type = "integer", default = 3000},
        backend_url = {type = "string"},
        backend_timeout = {type = "integer", default = 5000},
        skip_paths = {
            type = "array",
            items = {type = "string"}
        },
        skip_prefixes = {
            type = "array", 
            items = {type = "string"}
        }
    },
    required = {"jwt_secret", "backend_url"}
}

-- 默认跳过认证的路径
local default_skip_paths = {
    "/api/user/login",
    "/api/user/logout",
    "/api/user/refresh_token",
    "/api/user/signup",
    "/api/user/profile",
    "/api/user/codes",
    "/api/not_auth/getBindIps",
    "/api/not_auth/getTreeNodeBindIps",
    "/favicon.ico",
    "/"
}

-- 默认跳过认证的路径前缀
local default_skip_prefixes = {
    "/swagger/",
    "/api/monitor/prometheus_configs/",
    "/api/tree/local/terminal",
    "/api/ai/chat/ws"
}

local _M = {
    version = 0.1,
    priority = 2530,
    name = plugin_name,
    schema = schema
}

-- 检查路径是否应该跳过认证
local function should_skip_auth(path, skip_paths, skip_prefixes)
    -- 检查完整路径匹配
    for _, skip_path in ipairs(skip_paths or default_skip_paths) do
        if path == skip_path then
            return true
        end
    end
    
    -- 检查路径前缀匹配
    for _, prefix in ipairs(skip_prefixes or default_skip_prefixes) do
        if core.string.has_prefix(path, prefix) then
            return true
        end
    end
    
    return false
end

-- 从请求中提取JWT Token
local function extract_token(ctx)
    local auth_header = core.request.header(ctx, "Authorization")
    if auth_header then
        local token = auth_header:match("Bearer%s+(.+)")
        if token then
            return token
        end
    end
    
    -- 从查询参数中获取（用于WebSocket）
    local args = core.request.get_uri_args(ctx)
    if args.token then
        return args.token
    end
    
    return nil
end

-- 验证JWT Token
local function verify_jwt(token, secret)
    if not token or not secret then
        return nil, "missing token or secret"
    end
    
    local jwt_obj = jwt:verify(secret, token)
    if not jwt_obj.valid then
        return nil, "invalid JWT token"
    end
    
    return jwt_obj.payload, nil
end

-- 连接Redis
local function connect_redis(conf)
    local red = redis:new()
    red:set_timeouts(conf.redis_timeout, conf.redis_timeout, conf.redis_timeout)
    
    local ok, err = red:connect(conf.redis_host, conf.redis_port)
    if not ok then
        return nil, "failed to connect to redis: " .. err
    end
    
    if conf.redis_password and conf.redis_password ~= "" then
        local res, err = red:auth(conf.redis_password)
        if not res then
            return nil, "failed to authenticate with redis: " .. err
        end
    end
    
    return red, nil
end

-- 检查会话
local function check_session(conf, session_id)
    if not session_id or session_id == "" then
        return false, "missing session id"
    end
    
    local red, err = connect_redis(conf)
    if not red then
        core.log.error("Redis connection failed: ", err)
        return false, err
    end
    
    local session_key = "session:" .. session_id
    local res, err = red:get(session_key)
    
    -- 关闭Redis连接
    local ok, err2 = red:setkeepalive(10000, 100)
    if not ok then
        core.log.warn("Failed to set Redis keepalive: ", err2)
    end
    
    if not res or res == ngx.null then
        return false, "session not found"
    end
    
    if err then
        return false, "redis error: " .. err
    end
    
    return true, nil
end

-- 检查用户权限
local function check_permissions(conf, user_id, path, method)
    -- 管理员用户跳过权限检查
    if user_id == 1 then  -- 假设管理员用户ID为1
        return true, nil
    end
    
    -- 调用后端API检查权限
    local httpc = http.new()
    httpc:set_timeout(conf.backend_timeout)
    
    local check_url = conf.backend_url .. "/api/internal/check_permission"
    
    local res, err = httpc:request_uri(check_url, {
        method = "POST",
        headers = {
            ["Content-Type"] = "application/json",
            ["X-Internal-Request"] = "true"
        },
        body = json.encode({
            user_id = user_id,
            path = path,
            method = method
        })
    })
    
    if not res then
        core.log.error("Failed to check permissions: ", err)
        return false, "permission check failed"
    end
    
    if res.status == 200 then
        local body = json.decode(res.body)
        return body.allowed == true, body.message
    else
        return false, "permission denied"
    end
end

-- 检查插件配置
function _M.check_schema(conf)
    return core.schema.check(schema, conf)
end

-- 请求阶段处理
function _M.access(conf, ctx)
    local path = ctx.var.uri
    local method = ctx.var.request_method
    
    core.log.info("JWT Auth RBAC Plugin: Processing request ", path, " ", method)
    
    -- 检查是否应该跳过认证
    if should_skip_auth(path, conf.skip_paths, conf.skip_prefixes) then
        core.log.info("Skipping auth for path: ", path)
        return
    end
    
    -- 提取JWT Token
    local token = extract_token(ctx)
    if not token then
        core.log.warn("No JWT token found in request")
        return 401, {message = "未登录或登录已过期"}
    end
    
    -- 验证JWT Token
    local payload, err = verify_jwt(token, conf.jwt_secret)
    if not payload then
        core.log.warn("JWT verification failed: ", err)
        return 401, {message = "Token验证失败"}
    end
    
    -- 检查Token是否过期
    local now = ngx.time()
    if payload.exp and payload.exp < now then
        core.log.warn("JWT token expired")
        return 401, {message = "Token已过期"}
    end
    
    -- 检查会话
    local session_valid, session_err = check_session(conf, payload.ssid)
    if not session_valid then
        core.log.warn("Session check failed: ", session_err)
        return 401, {message = "会话已失效"}
    end
    
    -- 检查权限（除了管理员和服务账号）
    if payload.username ~= "admin" and payload.account_type ~= 2 then
        local allowed, perm_err = check_permissions(conf, payload.uid, path, method)
        if not allowed then
            core.log.warn("Permission check failed: ", perm_err)
            return 403, {message = "无权限访问该接口"}
        end
    end
    
    -- 将用户信息添加到请求头，供后端服务使用
    core.request.set_header(ctx, "X-User-ID", payload.uid)
    core.request.set_header(ctx, "X-User-Name", payload.username)
    core.request.set_header(ctx, "X-Session-ID", payload.ssid)
    core.request.set_header(ctx, "X-Account-Type", payload.account_type or 1)
    
    core.log.info("JWT Auth RBAC Plugin: Authentication successful for user ", payload.username)
end

return _M
