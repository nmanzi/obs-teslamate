#!/bin/bash

# Test script to demonstrate real-time MapEnabled config switching
# This script toggles the MapEnabled setting and shows how the views switch

echo "Tesla Location Server - Real-time Config Test"
echo "=============================================="
echo ""

# Function to get current config
get_config() {
    curl -s http://localhost:8081/config | jq -r '.map_enabled'
}

# Function to set config
set_config() {
    local map_enabled=$1
    curl -s -X POST http://localhost:8081/config \
        -H "Content-Type: application/json" \
        -d "{\"show_route\": true, \"mapbox_token\": \"$MAPBOX_TOKEN\", \"map_enabled\": $map_enabled, \"timezonedb_token\": \"$TIMEZONEDB_TOKEN\"}" \
        > /dev/null
}

echo "1. Starting with current config..."
current_state=$(get_config)
echo "   Current MapEnabled state: $current_state"
echo ""

echo "2. Testing config switch to 'false'..."
set_config false
sleep 1
new_state=$(get_config)
echo "   New MapEnabled state: $new_state"
echo "   -> Views should now show offline content within 2 seconds"
echo ""

echo "3. Waiting 5 seconds to observe the change..."
sleep 5
echo ""

echo "4. Testing config switch back to 'true'..."
set_config true
sleep 1
final_state=$(get_config)
echo "   Final MapEnabled state: $final_state"
echo "   -> Views should now show live content within 2 seconds"
echo ""

echo "Test complete!"
echo ""
echo "Instructions:"
echo "1. Open http://localhost:8081 in your browser"
echo "2. Open http://localhost:8081/overlay in another tab"
echo "3. Run this script to see real-time switching"
echo "4. The changes should appear within 2 seconds without page refresh"