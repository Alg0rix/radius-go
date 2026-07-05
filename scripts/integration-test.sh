#!/usr/bin/env bash
set -euo pipefail

# Integration test for radius-go via radiusctl + radclient.
# Assumes docker compose is running with the default secret.

SERVER="${SERVER:-http://localhost:8083}"
SECRET="${SECRET:-change-me-in-production}"
export RADIUS_SECRET="${SECRET}"
export RADIUS_SERVER="${SERVER}"
RADIUS_SECRET_RADIUS="testing123"
RADIUS_HOST="${RADIUS_HOST:-127.0.0.1}"
TEST_RUN_ID=$(date +%s%N | tail -c 6)
TEST_SESSION_ID="TEST-SESSION-${TEST_RUN_ID}"
VOUCHER_SESSION_ID="VOUCHER-SESSION-${TEST_RUN_ID}"

# Resolve container hostname to IP for HTTP API / RADIUS destination.
if [ "${RADIUS_HOST}" = "radius-go" ]; then
  API_HOST=$(getent hosts radius-go | awk '{print $1}' | head -n1)
  echo "Resolved radius-go API target to ${API_HOST}"
fi

# For RADIUS, the NAS IP must match the source address of the test packets
# (i.e. this container's IP), because the server looks up the secret by remote addr.
RADIUS_PACKET_SRC="${RADIUS_PACKET_SRC:-}"
if [ -z "${RADIUS_PACKET_SRC:-}" ]; then
  RADIUS_PACKET_SRC=$(ip route get "${API_HOST}" 2>/dev/null | awk '{for(i=1;i<=NF;i++) if ($i=="src") print $(i+1); exit}')
fi
if [ -z "${RADIUS_PACKET_SRC:-}" ]; then
  RADIUS_PACKET_SRC=$(hostname -i 2>/dev/null | awk '{print $1}' | head -n1)
fi
echo "RADIUS packet source / NAS IP: ${RADIUS_PACKET_SRC}"

echo "==> Building radiusctl"
if command -v go >/dev/null 2>&1; then
  go run ./cmd/radiusctl --help >/dev/null
else
  radiusctl --help >/dev/null
fi

radiusctl() {
  if command -v go >/dev/null 2>&1; then
    go run ./cmd/radiusctl "$@"
  else
    command radiusctl "$@"
  fi
}

echo "==> Checking server health"
curl -sf "${SERVER}/health" | jq .

echo "==> Server status"
radiusctl status

echo "==> Cleanup: delete existing test NAS/subscriber if present"
EXISTING_NAS=$(radiusctl nas list --json 2>/dev/null | jq -r '.[] | select(.name=="test-nas") | .id')
if [ -n "${EXISTING_NAS}" ]; then
  radiusctl nas delete --id "${EXISTING_NAS}"
fi

EXISTING_USER=$(radiusctl subscriber list --json 2>/dev/null | jq -r '.[] | select(.username=="testuser") | .id')
if [ -n "${EXISTING_USER}" ]; then
  radiusctl subscriber delete --id "${EXISTING_USER}"
fi

echo "==> Create NAS"
NAS_ID=$(radiusctl nas create --name test-nas --ip "${RADIUS_PACKET_SRC}" --secret "${RADIUS_SECRET_RADIUS}" --json | jq -r '.id')
echo "NAS ID: ${NAS_ID}"

echo "==> List NASes"
radiusctl nas list

echo "==> Create subscriber"
USER_ID=$(radiusctl subscriber create --username testuser --password testpass --json | jq -r '.id')
echo "User ID: ${USER_ID}"

echo "==> List subscribers"
radiusctl subscriber list

echo "==> Server status after seeding"
radiusctl status

# Use radclient if available (Ubuntu/Debian), fall back to radiusclient on Alpine.
RADCLIENT_BIN="${RADCLIENT_BIN:-radclient}"
if ! command -v "${RADCLIENT_BIN}" >/dev/null 2>&1; then
  if command -v radiusclient >/dev/null 2>&1; then
    RADCLIENT_BIN="radiusclient"
  fi
fi

radiusclient_cmd() {
  local host port method secret
  host="$1"
  port="$2"
  method="$3"
  secret="$4"
  if [ "${RADCLIENT_BIN}" = "radiusclient" ]; then
    local tmpdir rc
    tmpdir=$(mktemp -d)
    rc="${tmpdir}/radiusclient.conf"
    cat > "${rc}" <<EOF
authserver ${host}:${port}
acctserver ${host}:${port}
servers ${tmpdir}/servers
EOF
    cat > "${tmpdir}/servers" <<EOF
${host} ${secret}
EOF
    ${RADCLIENT_BIN} -f "${rc}" -p 0 "${method}"
    rm -rf "${tmpdir}"
  else
    ${RADCLIENT_BIN} -x "${host}:${port}" "${method}" "${secret}"
  fi
}

echo "==> RADIUS auth test (should get Access-Accept)"
echo "User-Name=testuser,User-Password=testpass" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1812 auth "${RADIUS_SECRET_RADIUS}"

echo "==> RADIUS auth failure test (should get Access-Reject)"
echo "User-Name=testuser,User-Password=wrongpass" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1812 auth "${RADIUS_SECRET_RADIUS}" || true

echo "==> RADIUS accounting start"
echo "User-Name=testuser,Acct-Session-Id=${TEST_SESSION_ID},NAS-IP-Address=${RADIUS_PACKET_SRC},Acct-Status-Type=Start" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1813 acct "${RADIUS_SECRET_RADIUS}"

echo "==> List sessions"
radiusctl session list

echo "==> RADIUS accounting interim-update"
echo "User-Name=testuser,Acct-Session-Id=${TEST_SESSION_ID},NAS-IP-Address=${RADIUS_PACKET_SRC},Acct-Input-Octets=1024,Acct-Output-Octets=2048,Acct-Session-Time=60,Acct-Status-Type=Interim-Update" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1813 acct "${RADIUS_SECRET_RADIUS}"

echo "==> List sessions after interim"
radiusctl session list

echo "==> Server status after auth + accounting"
radiusctl status

echo "==> RADIUS accounting stop"
echo "User-Name=testuser,Acct-Session-Id=${TEST_SESSION_ID},NAS-IP-Address=${RADIUS_PACKET_SRC},Acct-Input-Octets=2048,Acct-Output-Octets=4096,Acct-Session-Time=120,Acct-Status-Type=Stop" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1813 acct "${RADIUS_SECRET_RADIUS}"

echo "==> Voucher feature tests"

echo "===> List voucher packages (should be empty)"
radiusctl voucher package list

echo "===> Create voucher package"
PKG_ID=$(radiusctl voucher package create --name "TestPackage" --description "Integration test package" --price 10.00 --speed-up 5120 --speed-down 10240 --data-cap-bytes 1073741824 --time-limit-type usage --time-limit-seconds 3600 --max-concurrent 1 --json | jq -r '.id')
echo "Package ID: ${PKG_ID}"

echo "===> List voucher packages"
radiusctl voucher package list

echo "===> Generate voucher"
VOUCHER=$(radiusctl voucher generate --package-id "${PKG_ID}" --count 1 --json | jq -r '.[0]')
VOUCHER_USER=$(echo "${VOUCHER}" | jq -r '.username')
VOUCHER_PASS=$(echo "${VOUCHER}" | jq -r '.password')
echo "Voucher username: ${VOUCHER_USER}"

echo "===> List vouchers"
radiusctl voucher list

echo "===> Check voucher balance before auth"
radiusctl voucher balance --code "${VOUCHER_USER}"

echo "===> RADIUS auth with voucher (should get Access-Accept)"
echo "User-Name=${VOUCHER_USER},User-Password=${VOUCHER_PASS}" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1812 auth "${RADIUS_SECRET_RADIUS}"

echo "===> Check voucher balance after auth"
radiusctl voucher balance --code "${VOUCHER_USER}"

echo "===> Voucher accounting start"
echo "User-Name=${VOUCHER_USER},Acct-Session-Id=${VOUCHER_SESSION_ID},NAS-IP-Address=${RADIUS_PACKET_SRC},Acct-Status-Type=Start" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1813 acct "${RADIUS_SECRET_RADIUS}"

echo "===> List sessions"
radiusctl session list

echo "===> Voucher accounting stop"
echo "User-Name=${VOUCHER_USER},Acct-Session-Id=${VOUCHER_SESSION_ID},NAS-IP-Address=${RADIUS_PACKET_SRC},Acct-Input-Octets=1048576,Acct-Output-Octets=2097152,Acct-Session-Time=300,Acct-Status-Type=Stop" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1813 acct "${RADIUS_SECRET_RADIUS}"

echo "===> Check voucher balance after accounting"
# Give the async AddUsageDelta goroutine time to persist.
for i in $(seq 1 10); do
  BALANCE=$(radiusctl voucher balance --code "${VOUCHER_USER}" --json)
  DATA_USED=$(echo "${BALANCE}" | jq -r '.data_bytes_used')
  if [ "${DATA_USED}" -gt 0 ]; then
    echo "${BALANCE}" | jq .
    break
  fi
  sleep 0.5
done
if [ "${DATA_USED:-0}" -eq 0 ]; then
  echo "WARNING: voucher usage not yet persisted to DB"
  radiusctl voucher balance --code "${VOUCHER_USER}"
fi

echo "===> Delete voucher subscriber"
VOUCHER_USER_ID=$(radiusctl subscriber list --json | jq -r ".[] | select(.username==\"${VOUCHER_USER}\") | .id")
radiusctl subscriber delete --id "${VOUCHER_USER_ID}"

echo "===> Delete voucher package"
radiusctl voucher package delete --id "${PKG_ID}"

echo "==> Voucher limit tests"

echo "===> Data-cap voucher"
PKG_DATA=$(radiusctl voucher package create --name "DataCapPackage" --price 1 --data-cap-bytes 1048576 --time-limit-type usage --json | jq -r '.id')
VOUCHER_DATA=$(radiusctl voucher generate --package-id "${PKG_DATA}" --count 1 --json | jq -r '.[0]')
VOUCHER_DATA_USER=$(echo "${VOUCHER_DATA}" | jq -r '.username')
VOUCHER_DATA_PASS=$(echo "${VOUCHER_DATA}" | jq -r '.password')
echo "Auth data-cap voucher (should accept)"
echo "User-Name=${VOUCHER_DATA_USER},User-Password=${VOUCHER_DATA_PASS}" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1812 auth "${RADIUS_SECRET_RADIUS}"
echo "Send accounting start"
echo "User-Name=${VOUCHER_DATA_USER},Acct-Session-Id=DATA-CAP-001,NAS-IP-Address=${RADIUS_PACKET_SRC},Acct-Status-Type=Start" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1813 acct "${RADIUS_SECRET_RADIUS}"
echo "Send accounting stop exceeding data cap"
echo "User-Name=${VOUCHER_DATA_USER},Acct-Session-Id=DATA-CAP-001,NAS-IP-Address=${RADIUS_PACKET_SRC},Acct-Input-Octets=2097152,Acct-Output-Octets=0,Acct-Session-Time=10,Acct-Status-Type=Stop" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1813 acct "${RADIUS_SECRET_RADIUS}"
echo "Auth again (should reject)"
if echo "User-Name=${VOUCHER_DATA_USER},User-Password=${VOUCHER_DATA_PASS}" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1812 auth "${RADIUS_SECRET_RADIUS}"; then
  echo "ERROR: expected Access-Reject for data-cap exceeded voucher"
  exit 1
else
  echo "OK: voucher rejected after data cap exceeded"
fi
VOUCHER_DATA_USER_ID=$(radiusctl subscriber list --json | jq -r ".[] | select(.username==\"${VOUCHER_DATA_USER}\") | .id")
radiusctl subscriber delete --id "${VOUCHER_DATA_USER_ID}"
radiusctl voucher package delete --id "${PKG_DATA}"

echo "===> Usage-time voucher"
PKG_TIME=$(radiusctl voucher package create --name "UsageTimePackage" --price 1 --time-limit-type usage --time-limit-seconds 10 --json | jq -r '.id')
VOUCHER_TIME=$(radiusctl voucher generate --package-id "${PKG_TIME}" --count 1 --json | jq -r '.[0]')
VOUCHER_TIME_USER=$(echo "${VOUCHER_TIME}" | jq -r '.username')
VOUCHER_TIME_PASS=$(echo "${VOUCHER_TIME}" | jq -r '.password')
echo "Auth usage-time voucher (should accept)"
echo "User-Name=${VOUCHER_TIME_USER},User-Password=${VOUCHER_TIME_PASS}" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1812 auth "${RADIUS_SECRET_RADIUS}"
echo "Send accounting start"
echo "User-Name=${VOUCHER_TIME_USER},Acct-Session-Id=USAGE-TIME-001,NAS-IP-Address=${RADIUS_PACKET_SRC},Acct-Status-Type=Start" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1813 acct "${RADIUS_SECRET_RADIUS}"
echo "Send accounting stop exceeding usage time"
echo "User-Name=${VOUCHER_TIME_USER},Acct-Session-Id=USAGE-TIME-001,NAS-IP-Address=${RADIUS_PACKET_SRC},Acct-Input-Octets=0,Acct-Output-Octets=0,Acct-Session-Time=30,Acct-Status-Type=Stop" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1813 acct "${RADIUS_SECRET_RADIUS}"
echo "Auth again (should reject)"
if echo "User-Name=${VOUCHER_TIME_USER},User-Password=${VOUCHER_TIME_PASS}" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1812 auth "${RADIUS_SECRET_RADIUS}"; then
  echo "ERROR: expected Access-Reject for usage-time exceeded voucher"
  exit 1
else
  echo "OK: voucher rejected after usage time exceeded"
fi
VOUCHER_TIME_USER_ID=$(radiusctl subscriber list --json | jq -r ".[] | select(.username==\"${VOUCHER_TIME_USER}\") | .id")
radiusctl subscriber delete --id "${VOUCHER_TIME_USER_ID}"
radiusctl voucher package delete --id "${PKG_TIME}"

echo "===> pfSense Hotspot accounting"
PFSENSE_SESSION_ID="pfSense-${TEST_RUN_ID}"
PFSENSE_USERNAME="pfsenseuser-${TEST_RUN_ID}"
echo "Create pfSense-style subscriber"
PFSENSE_USER_ID=$(radiusctl subscriber create --username "${PFSENSE_USERNAME}" --password "pfsensepass" --json | jq -r '.id')
echo "pfSense auth (should accept)"
echo "User-Name=${PFSENSE_USERNAME},User-Password=pfsensepass,NAS-Identifier=pfSense,Called-Station-Id=00:11:22:33:44:55,Calling-Station-Id=aa:bb:cc:dd:ee:ff,Framed-IP-Address=10.20.30.40" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1812 auth "${RADIUS_SECRET_RADIUS}"
echo "pfSense accounting start"
echo "User-Name=${PFSENSE_USERNAME},Acct-Session-Id=${PFSENSE_SESSION_ID},NAS-IP-Address=${RADIUS_PACKET_SRC},NAS-Identifier=pfSense,Called-Station-Id=00:11:22:33:44:55,Calling-Station-Id=aa:bb:cc:dd:ee:ff,Framed-IP-Address=10.20.30.40,Acct-Status-Type=Start" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1813 acct "${RADIUS_SECRET_RADIUS}"
echo "pfSense interim update (with gigawords)"
echo "User-Name=${PFSENSE_USERNAME},Acct-Session-Id=${PFSENSE_SESSION_ID},NAS-IP-Address=${RADIUS_PACKET_SRC},NAS-Identifier=pfSense,Called-Station-Id=00:11:22:33:44:55,Calling-Station-Id=aa:bb:cc:dd:ee:ff,Framed-IP-Address=10.20.30.40,Acct-Status-Type=Interim-Update,Acct-Input-Octets=1073741824,Acct-Output-Octets=536870912,Acct-Input-Gigawords=1,Acct-Output-Gigawords=0,Acct-Session-Time=300" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1813 acct "${RADIUS_SECRET_RADIUS}"
echo "pfSense accounting stop"
echo "User-Name=${PFSENSE_USERNAME},Acct-Session-Id=${PFSENSE_SESSION_ID},NAS-IP-Address=${RADIUS_PACKET_SRC},NAS-Identifier=pfSense,Called-Station-Id=00:11:22:33:44:55,Calling-Station-Id=aa:bb:cc:dd:ee:ff,Framed-IP-Address=10.20.30.40,Acct-Status-Type=Stop,Acct-Input-Octets=2147483648,Acct-Output-Octets=1073741824,Acct-Input-Gigawords=2,Acct-Output-Gigawords=1,Acct-Session-Time=600" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1813 acct "${RADIUS_SECRET_RADIUS}"
echo "Verify session stopped"
if radiusctl session list --json | jq -e ".[] | select(.username==\"${PFSENSE_USERNAME}\" and .session_status==\"stopped\")" >/dev/null; then
  echo "OK: pfSense session recorded as stopped"
else
  echo "ERROR: pfSense session not found or not stopped"
  exit 1
fi
radiusctl subscriber delete --id "${PFSENSE_USER_ID}"

echo "===> Calendar-expiry voucher"
PKG_CAL=$(radiusctl voucher package create --name "CalendarPackage" --price 1 --time-limit-type calendar --time-limit-seconds 2 --json | jq -r '.id')
VOUCHER_CAL=$(radiusctl voucher generate --package-id "${PKG_CAL}" --count 1 --json | jq -r '.[0]')
VOUCHER_CAL_USER=$(echo "${VOUCHER_CAL}" | jq -r '.username')
VOUCHER_CAL_PASS=$(echo "${VOUCHER_CAL}" | jq -r '.password')
echo "Auth calendar voucher (should accept)"
echo "User-Name=${VOUCHER_CAL_USER},User-Password=${VOUCHER_CAL_PASS}" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1812 auth "${RADIUS_SECRET_RADIUS}"
echo "Wait for calendar expiry"
sleep 3
echo "Auth again (should reject)"
if echo "User-Name=${VOUCHER_CAL_USER},User-Password=${VOUCHER_CAL_PASS}" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1812 auth "${RADIUS_SECRET_RADIUS}"; then
  echo "ERROR: expected Access-Reject for expired calendar voucher"
  exit 1
else
  echo "OK: voucher rejected after calendar expiry"
fi
VOUCHER_CAL_USER_ID=$(radiusctl subscriber list --json | jq -r ".[] | select(.username==\"${VOUCHER_CAL_USER}\") | .id")
radiusctl subscriber delete --id "${VOUCHER_CAL_USER_ID}"
radiusctl voucher package delete --id "${PKG_CAL}"

echo "===> Bandwidth voucher"
PKG_BW=$(radiusctl voucher package create --name "BandwidthPackage" --price 1 --speed-up 1024 --speed-down 2048 --json | jq -r '.id')
VOUCHER_BW=$(radiusctl voucher generate --package-id "${PKG_BW}" --count 1 --json | jq -r '.[0]')
VOUCHER_BW_USER=$(echo "${VOUCHER_BW}" | jq -r '.username')
VOUCHER_BW_PASS=$(echo "${VOUCHER_BW}" | jq -r '.password')
echo "Auth bandwidth voucher and check rate-limit attribute"
echo "User-Name=${VOUCHER_BW_USER},User-Password=${VOUCHER_BW_PASS}" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1812 auth "${RADIUS_SECRET_RADIUS}" | grep -q "Mikrotik-Rate-Limit = \"1024K/2048K\"" && echo "OK: bandwidth attributes correct" || echo "WARNING: bandwidth attribute mismatch"
VOUCHER_BW_USER_ID=$(radiusctl subscriber list --json | jq -r ".[] | select(.username==\"${VOUCHER_BW_USER}\") | .id")
radiusctl subscriber delete --id "${VOUCHER_BW_USER_ID}"
radiusctl voucher package delete --id "${PKG_BW}"

echo "===> PPPoE access test"
PPPOE_SESSION_ID="PPPoE-${TEST_RUN_ID}"
PPPOE_USERNAME="pppoeuser-${TEST_RUN_ID}"
echo "Create PPPoE-style subscriber"
PPPOE_USER_ID=$(radiusctl subscriber create --username "${PPPOE_USERNAME}" --password "pppoepass" --service-type login --json | jq -r '.id')
echo "PPPoE auth (Login-Service should be present)"
echo "User-Name=${PPPOE_USERNAME},User-Password=pppoepass,NAS-Identifier=PPPoE-BRAS,Service-Type=Login-User,Framed-Protocol=PPP,Framed-MTU=1480,Called-Station-Id=00:11:22:33:44:55,Calling-Station-Id=aa:bb:cc:dd:ee:ff" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1812 auth "${RADIUS_SECRET_RADIUS}"
echo "PPPoE accounting start"
echo "User-Name=${PPPOE_USERNAME},Acct-Session-Id=${PPPOE_SESSION_ID},NAS-IP-Address=${RADIUS_PACKET_SRC},NAS-Identifier=PPPoE-BRAS,Framed-Protocol=PPP,Framed-IP-Address=100.64.10.20,Calling-Station-Id=aa:bb:cc:dd:ee:ff,Called-Station-Id=00:11:22:33:44:55,Acct-Status-Type=Start" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1813 acct "${RADIUS_SECRET_RADIUS}"
echo "PPPoE interim update"
echo "User-Name=${PPPOE_USERNAME},Acct-Session-Id=${PPPOE_SESSION_ID},NAS-IP-Address=${RADIUS_PACKET_SRC},NAS-Identifier=PPPoE-BRAS,Framed-Protocol=PPP,Framed-IP-Address=100.64.10.20,Calling-Station-Id=aa:bb:cc:dd:ee:ff,Called-Station-Id=00:11:22:33:44:55,Acct-Input-Octets=16777216,Acct-Output-Octets=33554432,Acct-Session-Time=120,Acct-Status-Type=Interim-Update" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1813 acct "${RADIUS_SECRET_RADIUS}"
echo "PPPoE accounting stop"
echo "User-Name=${PPPOE_USERNAME},Acct-Session-Id=${PPPOE_SESSION_ID},NAS-IP-Address=${RADIUS_PACKET_SRC},NAS-Identifier=PPPoE-BRAS,Framed-Protocol=PPP,Framed-IP-Address=100.64.10.20,Calling-Station-Id=aa:bb:cc:dd:ee:ff,Called-Station-Id=00:11:22:33:44:55,Acct-Input-Octets=33554432,Acct-Output-Octets=67108864,Acct-Session-Time=300,Acct-Status-Type=Stop" | radiusclient_cmd "${API_HOST:-${RADIUS_HOST}}" 1813 acct "${RADIUS_SECRET_RADIUS}"
echo "Verify PPPoE session stopped"
if radiusctl session list --json | jq -e ".[] | select(.username==\"${PPPOE_USERNAME}\" and .session_status==\"stopped\")" >/dev/null; then
  echo "OK: PPPoE session recorded as stopped"
else
  echo "ERROR: PPPoE session not found or not stopped"
  exit 1
fi
radiusctl subscriber delete --id "${PPPOE_USER_ID}"

echo "==> Cleanup sessions"
radiusctl session cleanup

echo "==> List sessions after stop/cleanup"
radiusctl session list

echo "==> Delete test subscriber"
radiusctl subscriber delete --id "${USER_ID}"

echo "==> Delete test NAS"
radiusctl nas delete --id "${NAS_ID}"

echo "==> Final status"
radiusctl status

echo "==> Integration test completed successfully"
