#!/bin/bash

echo "=== WebSocket Debugging Script ==="
echo ""

echo "1. Checking if api-gateway is running and listening on port 8083:"
docker compose ps api-gateway
echo ""

echo "2. Checking if Caddy is running:"
docker compose ps caddy
echo ""

echo "3. Checking port 80 (Caddy):"
sudo netstat -tulpn | grep :80
echo ""

echo "4. Checking recent api-gateway logs:"
docker compose logs --tail=50 api-gateway
echo ""

echo "5. Checking recent Caddy logs:"
docker compose logs --tail=50 caddy
echo ""

echo "6. Testing WebSocket endpoint from inside the server (without auth - should get 401):"
curl -i -N -H "Connection: Upgrade" -H "Upgrade: websocket" -H "Sec-WebSocket-Version: 13" -H "Sec-WebSocket-Key: test" http://localhost/ws
echo ""

echo "7. Checking if firewall is blocking connections:"
sudo iptables -L -n | grep -E "(80|8083)"
echo ""

echo "8. Checking Docker networks:"
docker network ls
docker network inspect newmessenger_default 2>/dev/null || docker network inspect $(docker network ls -q | head -1)
echo ""

echo "=== End of diagnostics ==="
