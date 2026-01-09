-- KEYS[1] = session key (login:2fa:{token})
-- ARGV[1] = provided code (PLAIN)
-- ARGV[2] = max attempts
-- ARGV[3] = ttl seconds

local data = redis.call("GET", KEYS[1])
if not data then
  return {err = "NOT_FOUND"}
end

local session = cjson.decode(data)

if session.attempts >= tonumber(ARGV[2]) then
  return {err = "LOCKED"}
end

-- bcrypt porównujesz w Go, Lua tylko steruje flow
session.attempts = session.attempts + 1
redis.call("SET", KEYS[1], cjson.encode(session), "EX", ARGV[3])

return {ok = "ATTEMPT_UPDATED", attempts = session.attempts}
