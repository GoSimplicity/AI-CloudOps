-- All schema, role, TYPE and fence checks finish before the first write.
-- Versions stay fixed-width decimal strings: Lua doubles never order uint64 IDs.
local ok,w=pcall(cjson.decode,ARGV[1])
if not ok or type(w)~='table' or type(w.roles)~='table' or #w.roles~=#KEYS then return 'INVALID' end
local prefix='aiops:monitor:config'
local paths={['1']='prometheus:main',['2']='alertmanager:main',['3']='prometheus:alert_rule',['4']='prometheus:record_rule'}
local publications={['1']='prometheus',['2']='alertmanager',['3']='alert_rule',['4']='record_rule'}
local function digits(s) return type(s)=='string' and s:match('^[1-9][0-9]*$')~=nil end
local function hex(s,n) return type(s)=='string' and #s==n and s:match('^[0-9a-f]+$')~=nil end
local function version(s) return type(s)=='string' and #s==20 and s:match('^[0-9]+$') and s~='00000000000000000000' and s<='18446744073709551615' end
local function byteshex(s) return (s:gsub('.',function(c)return string.format('%02x',string.byte(c))end)) end
local function index(n) return type(n)=='number' and n%1==0 and n>=1 and n<=#KEYS end
local function poolkey(suffix) return prefix..':projection:v1:pool:'..w.family..':'..w.pool..':'..suffix end
local pool=w.pool_fence~=0
if pool and (not digits(w.pool) or not paths[w.family] and w.family~='5') then return 'INVALID' end
local function array(t)
 if type(t)~='table' then return false end
 local n=0;for k,_ in pairs(t) do if type(k)~='number' or k%1~=0 or k<1 or k>#t then return false end;n=n+1 end
 return n==#t
end
if not array(w.roles) or not array(w.fences) or not array(w.artifacts) or not array(w.controls) then return 'INVALID' end
local seen={}
for i,r in ipairs(w.roles) do
 if type(r)~='table' or r.key~=KEYS[i] or seen[KEYS[i]] then return 'ROLE' end
 seen[KEYS[i]]=true
 local expected=nil
 local wanted=nil
 if r.role=='artifact' or r.role=='namespace_fence' then
  if type(r.ip)~='string' or not r.ip:match('^[0-9a-fA-F:.]+$') then return 'ROLE' end
  local logical=r.ip
  if r.kind=='5' then if not digits(r.group) then return 'ROLE' end;logical=logical..'_'..r.group
  elseif not paths[r.kind] or r.group~='' then return 'ROLE' end
  if r.role=='namespace_fence' then expected=prefix..':projection:v1:namespace:'..r.kind..':'..byteshex(logical);wanted='hash'
  elseif r.kind=='5' then expected='monitor:webhook_file:'..r.ip..':'..r.group;wanted='string'
  else expected=prefix..':'..paths[r.kind]..':'..r.ip;wanted='string' end
 elseif pool and r.role=='pool_fence' then expected=poolkey('fence');wanted='hash'
 elseif pool and r.role=='ack' then expected=poolkey('ack');wanted='string'
 elseif pool and w.family~='5' and r.role=='legacy_ack' then expected=prefix..':publication:'..publications[w.family]..':pool:'..w.pool;wanted='string'
 elseif pool and paths[w.family] and r.role=='members' then expected=prefix..':'..paths[w.family]..':pool:'..w.pool..':ips';wanted='set'
 elseif pool and w.family=='2' and r.role=='intents' then expected=prefix..':retirement:alertmanager:webhook:pool:'..w.pool;wanted='set'
 else return 'ROLE' end
 if expected~=KEYS[i] or wanted~=r.type then return 'ROLE' end
 local actual=redis.call('TYPE',KEYS[i]).ok
 if actual~='none' and actual~=wanted then return 'TYPE' end
end
if type(w.fences)~='table' or type(w.artifacts)~='table' or type(w.controls)~='table' then return 'INVALID' end
if w.mode~='prepare' and w.mode~='artifact' and w.mode~='members' and w.mode~='intent_add' and w.mode~='intent_remove' and w.mode~='ack' then return 'INVALID' end
if pool and (not index(w.pool_fence) or w.roles[w.pool_fence].role~='pool_fence') then return 'ROLE' end
local fenceSeen={}
for _,f in ipairs(w.fences) do
 if not index(f.key) or fenceSeen[f.key] or (w.roles[f.key].role~='namespace_fence' and w.roles[f.key].role~='pool_fence') or not version(f.version) or not hex(f.nonce,32) or f.nonce==string.rep('0',32) or not hex(f.signature,64) then return 'INVALID' end
 fenceSeen[f.key]=f
 local current=redis.call('HGET',KEYS[f.key],'version')
 if current then
  if not version(current) then return 'CORRUPT' end
  if current>f.version then return 'STALE' end
  if current==f.version and (redis.call('HGET',KEYS[f.key],'nonce')~=f.nonce or redis.call('HGET',KEYS[f.key],'signature')~=f.signature) then return 'CONFLICT' end
  if current<f.version and w.mode~='prepare' then return 'UNPREPARED' end
 elseif redis.call('TYPE',KEYS[f.key]).ok~='none' then return 'CORRUPT'
 elseif w.mode~='prepare' then return 'UNPREPARED' end
end
if pool and not fenceSeen[w.pool_fence] then return 'INVALID' end
local artifacts={}
local rawSeen={}
local namespacePayload={}
for _,a in ipairs(w.artifacts) do
 if not digits(a.id) or artifacts[a.id] or not index(a.raw) or w.roles[a.raw].role~='artifact' or not fenceSeen[a.fence] or w.roles[a.fence].role~='namespace_fence' or not hex(a.signature,64) or a.signature~=fenceSeen[a.fence].signature or not hex(a.sha1,40) then return 'INVALID' end
 if w.roles[a.raw].kind~=w.roles[a.fence].kind or w.roles[a.raw].ip~=w.roles[a.fence].ip or w.roles[a.raw].group~=w.roles[a.fence].group then return 'ROLE' end
 if rawSeen[a.raw] or namespacePayload[a.fence] then return 'INVALID' end
 rawSeen[a.raw]=true;namespacePayload[a.fence]=a
 if a.action~='render' and a.action~='retire' and a.action~='invalidate' then return 'INVALID' end
 if not pool and a.action=='render' then return 'ROLE' end
 artifacts[a.id]=a
end
for _,f in ipairs(w.fences) do
 if w.roles[f.key].role=='namespace_fence' and not namespacePayload[f.key] then return 'INVALID' end
 local a=namespacePayload[f.key]
 if a and redis.call('HGET',KEYS[f.key],'version')==f.version then
  if redis.call('HGET',KEYS[f.key],'raw')~=KEYS[a.raw] or redis.call('HGET',KEYS[f.key],'action')~=a.action or redis.call('HGET',KEYS[f.key],'body')~=a.sha1 then return 'CONFLICT' end
 end
end
local function listfingerprint(values)
 local out={};for _,v in ipairs(values) do out[#out+1]=#v..':'..v end;return #values..';'..table.concat(out)
end
local function controlfingerprint(c)
 return redis.sha1hex(c.id..'|'..c.kind..'|'..c.key..'|'..c.signature..'|'..tostring(c.require_empty)..'|'..listfingerprint(c.desired)..'|'..listfingerprint(c.add)..'|'..listfingerprint(c.remove))
end
local controls={}
local kinds={}
for _,c in ipairs(w.controls) do
 if not pool or not digits(c.id) or controls[c.id] or kinds[c.kind] or not hex(c.signature,64) or not array(c.desired) or not array(c.add) or not array(c.remove) then return 'INVALID' end
 kinds[c.kind]=true
 local desiredSeen={}
 for _,ip in ipairs(c.desired) do if type(ip)~='string' or not ip:match('^[0-9a-fA-F:.]+$') or desiredSeen[ip] then return 'INVALID' end;desiredSeen[ip]=true end
 for _,entry in ipairs(c.add) do if type(entry)~='string' then return 'INVALID' end end
 for _,entry in ipairs(c.remove) do if type(entry)~='string' then return 'INVALID' end end

 if c.kind=='members_full' and (#c.add~=0 or #c.remove~=0 or c.require_empty~=false) then return 'INVALID' end
 if c.kind=='webhook_intents' and #c.desired~=0 then return 'INVALID' end
 if (c.kind=='ack_full' or c.kind=='ack_webhook_subset') and (c.key~=0 or #c.desired~=0 or #c.add~=0 or #c.remove~=0 or c.require_empty~=false) then return 'INVALID' end
 if c.kind=='members_full' then if w.family=='5' or not index(c.key) or w.roles[c.key].role~='members' or type(c.desired)~='table' then return 'ROLE' end
 elseif c.kind=='webhook_intents' then if w.family~='2' or not index(c.key) or w.roles[c.key].role~='intents' or type(c.add)~='table' or type(c.remove)~='table' or c.require_empty~=true then return 'ROLE' end
 elseif c.kind=='ack_full' then if w.family=='5' then return 'ROLE' end
 elseif c.kind=='ack_webhook_subset' then if w.family~='5' then return 'ROLE' end
 else return 'ROLE' end
 if redis.call('HGET',KEYS[w.pool_fence],'version')==fenceSeen[w.pool_fence].version and redis.call('HGET',KEYS[w.pool_fence],'expected:'..c.id)~=controlfingerprint(c) then return 'CONFLICT' end
 controls[c.id]=c
end
if pool then
 if not index(w.ack) or w.roles[w.ack].role~='ack' or type(w.ack_receipt)~='string' or not hex(w.ack_signature,64) then return 'INVALID' end
 if w.family=='5' then
  if #w.controls~=1 or not kinds['ack_webhook_subset'] or w.legacy~=0 or w.legacy_sha~='' then return 'ROLE' end
 else
  local expected=2;if w.family=='2' then expected=3 end
  if #w.controls~=expected or not kinds['members_full'] or not kinds['ack_full'] or (w.family=='2' and not kinds['webhook_intents']) or not index(w.legacy) or w.roles[w.legacy].role~='legacy_ack' or not hex(w.legacy_sha,64) then return 'ROLE' end
 end
 local current=redis.call('HGET',KEYS[w.pool_fence],'version')
 if current==fenceSeen[w.pool_fence].version and (redis.call('HGET',KEYS[w.pool_fence],'ack_body')~=redis.sha1hex(w.ack_receipt) or redis.call('HGET',KEYS[w.pool_fence],'control_count')~=tostring(#w.controls) or redis.call('HGET',KEYS[w.pool_fence],'ack_signature')~=w.ack_signature) then return 'CONFLICT' end
 for _,c in ipairs(w.controls) do if c.kind=='ack_full' or c.kind=='ack_webhook_subset' then if c.signature~=w.ack_signature then return 'CONFLICT' end end end
elseif #w.controls~=0 or w.ack~=0 or w.legacy~=0 then return 'ROLE' end
-- Bind every behavior-bearing field to the prepared pool version. A caller
-- cannot retain the same opaque SQL signature while swapping an output key,
-- dropping an artifact, changing the SHA, or adding a previously unseen key.
local function framefingerprint()
 local parts={}
 local function add(value) local s=tostring(value);parts[#parts+1]=#s..':'..s end
 add(w.family);add(w.pool);add(w.pool_fence);add(w.ack);add(w.legacy);add(w.legacy_sha);add(w.ack_signature);add(w.ack_receipt)
 add(#w.roles)
 for _,r in ipairs(w.roles) do add(r.role);add(r.key);add(r.type);add(r.kind);add(r.ip);add(r.group) end
 add(#w.fences)
 for _,f in ipairs(w.fences) do add(f.key);add(f.version);add(f.nonce);add(f.signature) end
 add(#w.artifacts)
 for _,a in ipairs(w.artifacts) do add(a.id);add(a.raw);add(a.fence);add(a.action);add(a.sha1);add(a.signature) end
 add(#w.controls)
 for _,c in ipairs(w.controls) do add(controlfingerprint(c)) end
 return redis.sha1hex(table.concat(parts))
end
local frame=framefingerprint()
if pool and redis.call('HGET',KEYS[w.pool_fence],'version')==fenceSeen[w.pool_fence].version and redis.call('HGET',KEYS[w.pool_fence],'frame')~=frame then return 'CONFLICT' end
local function allartifacts()
 for _,a in ipairs(w.artifacts) do
  if redis.call('HGET',KEYS[a.fence],'done')~=a.signature then return false end
  if a.action=='render' then local body=redis.call('GET',KEYS[a.raw]);if not body or redis.sha1hex(body)~=a.sha1 then return false end
  elseif redis.call('EXISTS',KEYS[a.raw])~=0 then return false end
 end
 return true
end
local action=nil
if w.mode=='artifact' then action=artifacts[w.target];if not action then return 'INVALID' end
 if action.action=='render' and redis.sha1hex(ARGV[2])~=action.sha1 then return 'BODY' end
elseif w.mode=='members' then action=controls[w.target];if not action or action.kind~='members_full' then return 'ROLE' end
 for _,ip in ipairs(action.desired) do if type(ip)~='string' then return 'INVALID' end end
elseif w.mode=='intent_add' or w.mode=='intent_remove' then action=controls[w.target];if not action or action.kind~='webhook_intents' then return 'ROLE' end
 for _,entry in ipairs(action.add) do if type(entry)~='string' then return 'INVALID' end end
 for _,entry in ipairs(action.remove) do if type(entry)~='string' then return 'INVALID' end end
 if w.mode=='intent_remove' and (not allartifacts() or redis.call('HGET',KEYS[w.pool_fence],'intent_add:'..action.id)~=action.signature) then return 'INCOMPLETE' end
elseif w.mode=='ack' then
 if not pool or not index(w.ack) or w.roles[w.ack].role~='ack' or type(w.ack_receipt)~='string' or not hex(w.ack_signature,64) then return 'INVALID' end
 if w.family=='5' then if w.legacy~=0 or w.legacy_sha~='' then return 'ROLE' end
 elseif not index(w.legacy) or w.roles[w.legacy].role~='legacy_ack' or not hex(w.legacy_sha,64) then return 'ROLE' end
 if not allartifacts() then return 'INCOMPLETE' end
 for _,c in ipairs(w.controls) do
  if c.kind=='members_full' then
   if redis.call('HGET',KEYS[w.pool_fence],'control:'..c.id)~=c.signature then return 'INCOMPLETE' end
   local desired={};for _,ip in ipairs(c.desired) do desired[ip]=true end
   local actual=redis.call('SMEMBERS',KEYS[c.key]);if #actual~=#c.desired then return 'INCOMPLETE' end
   for _,ip in ipairs(actual) do if not desired[ip] then return 'INCOMPLETE' end end
  elseif c.kind=='webhook_intents' then
   if redis.call('HGET',KEYS[w.pool_fence],'control:'..c.id)~=c.signature or redis.call('SCARD',KEYS[c.key])~=0 then return 'INCOMPLETE' end
  end
 end
end
-- No validation below this line: Redis Lua does not roll back earlier writes.
if w.mode=='prepare' then
 for _,f in ipairs(w.fences) do
  local current=redis.call('HGET',KEYS[f.key],'version')
  if not current or current<f.version then
   redis.call('DEL',KEYS[f.key]);redis.call('HSET',KEYS[f.key],'version',f.version,'nonce',f.nonce,'signature',f.signature)
   local a=namespacePayload[f.key]
   if a then redis.call('HSET',KEYS[f.key],'raw',KEYS[a.raw],'action',a.action,'body',a.sha1) end
   if f.key==w.pool_fence then
    redis.call('HSET',KEYS[f.key],'frame',frame,'ack_body',redis.sha1hex(w.ack_receipt),'ack_signature',w.ack_signature,'control_count',tostring(#w.controls))
    for _,c in ipairs(w.controls) do redis.call('HSET',KEYS[f.key],'expected:'..c.id,controlfingerprint(c)) end
   end
  end
 end
elseif w.mode=='artifact' then
 if action.action=='render' then redis.call('SET',KEYS[action.raw],ARGV[2]) else redis.call('DEL',KEYS[action.raw]) end
 redis.call('HSET',KEYS[action.fence],'done',action.signature)
elseif w.mode=='members' then
 redis.call('DEL',KEYS[action.key]);for _,ip in ipairs(action.desired) do redis.call('SADD',KEYS[action.key],ip) end
 redis.call('HSET',KEYS[w.pool_fence],'control:'..action.id,action.signature)
elseif w.mode=='intent_add' then
 if redis.call('HGET',KEYS[w.pool_fence],'control:'..action.id)~=action.signature then
  for _,entry in ipairs(action.add) do redis.call('SADD',KEYS[action.key],entry) end
  redis.call('HSET',KEYS[w.pool_fence],'intent_add:'..action.id,action.signature)
 end
elseif w.mode=='intent_remove' then
 for _,entry in ipairs(action.remove) do redis.call('SREM',KEYS[action.key],entry) end
 redis.call('HSET',KEYS[w.pool_fence],'control:'..action.id,action.signature)
elseif w.mode=='ack' then
 if w.legacy~=0 then redis.call('SET',KEYS[w.legacy],w.legacy_sha) end
 redis.call('SET',KEYS[w.ack],w.ack_receipt);redis.call('HSET',KEYS[w.pool_fence],'ack',w.ack_signature)
end
return 'OK'
