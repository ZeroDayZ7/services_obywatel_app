-- KEYS[1] = login:2fa:{token}
-- ARGV[1] = max attempts
-- ARGV[2] = ttl seconds

local data = redis.call("GET", KEYS[1])
if not data then
  return { "NOT_FOUND" }
end

local session = cjson.decode(data)

if session.attempts >= tonumber(ARGV[1]) then
  return { "LOCKED" }
end

session.attempts = session.attempts + 1
redis.call("SET", KEYS[1], cjson.encode(session), "EX", ARGV[2])

return { "ATTEMPT_UPDATED", session.attempts }
