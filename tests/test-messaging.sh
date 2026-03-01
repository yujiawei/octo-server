#!/usr/bin/env bash
source "$(dirname "$0")/lib.sh"
reset_counters
log_section "Messaging Tests"

login_and_get_token
if [[ $? -ne 0 ]]; then
  fail "login failed for messaging tests"
  print_summary
  exit 0
fi
pass "login succeeded (uid=$USER_UID)"

# Send via Bot API
perform_request POST "/v1/bot/sendMessage"   "{\"channel_id\":\"$GROUP_ID\",\"channel_type\":2,\"payload\":{\"type\":1,\"content\":\"test-msg\"}}"   -H "Authorization: Bearer $BOT_TOKEN"
expect_http 200 "bot send message" && pass "bot send message"

# Channel sync
perform_request POST "/v1/message/channel/sync"   "{\"channel_id\":\"$GROUP_ID\",\"channel_type\":2,\"start_message_seq\":0,\"end_message_seq\":0,\"pull_mode\":1,\"limit\":10,\"login_uid\":\"$USER_UID\",\"device_uuid\":\"cli\"}"   -H "token: $USER_TOKEN"
expect_http 200 "channel history" && pass "channel history"

# Ping
perform_request GET "/v1/ping"
expect_http 200 "server ping" && pass "server ping"

print_summary
