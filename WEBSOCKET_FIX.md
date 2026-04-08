# WebSocket Connection Fix Guide

## Problem
WebSocket connections fail with `NS_ERROR_WEBSOCKET_CONNECTION_REFUSED` when accessing from remote devices.

## Root Cause
The CORS middleware wasn't properly configured for same-origin WebSocket requests (when frontend is served from the same domain as the API).

## Solution

### Step 1: Update the Code
The CORS middleware has been updated to:
1. Allow same-origin requests (when there's no Origin header)
2. Include WebSocket-specific headers in CORS configuration

**File changed:** `api-gateway/internal/cores/middleware/cors.go`

### Step 2: Rebuild and Restart
Run these commands on your server:

```bash
# Stop the services
docker compose down

# Rebuild the api-gateway with the new CORS fix
docker compose build api-gateway

# Start everything again
docker compose up -d

# Check if services are running
docker compose ps
```

### Step 3: Verify the Fix
After restarting, check the logs:

```bash
# Watch api-gateway logs
docker compose logs -f api-gateway

# In another terminal, watch Caddy logs
docker compose logs -f caddy
```

Then try to connect from a remote device and watch the logs for any errors.

### Step 4: Run Diagnostics (if still not working)
If WebSocket still doesn't work, run the diagnostic script:

```bash
chmod +x debug-websocket.sh
./debug-websocket.sh > websocket-debug.log 2>&1
```

Send the `websocket-debug.log` file for further analysis.

## Additional Checks

### Check if port 80 is accessible from outside
On the server:
```bash
# Check if firewall allows port 80
sudo ufw status
# If port 80 is not allowed, add it:
sudo ufw allow 80/tcp
```

### Check Docker network
```bash
# Ensure api-gateway can reach other services
docker compose exec api-gateway ping -c 3 caddy
docker compose exec api-gateway ping -c 3 realtime-service
```

### Test WebSocket from command line
From your laptop (replace IP with server IP):
```bash
# This should return 401 Unauthorized (which means WebSocket endpoint is reachable)
curl -i -N -H "Connection: Upgrade" -H "Upgrade: websocket" \
  -H "Sec-WebSocket-Version: 13" -H "Sec-WebSocket-Key: test" \
  http://95.165.87.48/ws
```

## What Changed

### Before (WRONG)
- CORS middleware required `ALLOWED_ORIGIN` env var in production
- Same-origin requests (no Origin header) were blocked
- WebSocket upgrade headers were not included in CORS config

### After (CORRECT)
- Same-origin requests are now allowed (when frontend is served from same domain)
- WebSocket upgrade headers are included: `Upgrade`, `Connection`, `Sec-WebSocket-Key`, `Sec-WebSocket-Version`, etc.
- CORS properly handles both cross-origin and same-origin scenarios

## Expected Behavior After Fix

1. ✅ Login works from remote devices
2. ✅ WebSocket connects successfully
3. ✅ Real-time messages work
4. ✅ No more `NS_ERROR_WEBSOCKET_CONNECTION_REFUSED`

## If Still Not Working

The issue might be:
1. **Firewall blocking WebSocket**: Check `sudo iptables -L -n`
2. **Caddy not proxying correctly**: Check Caddy logs
3. **realtime-service not running**: Check `docker compose ps realtime-service`
4. **Network issue**: Try from server itself: `curl http://localhost/ws`

Run the diagnostic script and share the output.
