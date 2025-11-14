#!/bin/bash

# Simple test script to send a message to the chat REPL
echo -e "Hello there!\nquit" | timeout 15s env AGENTHUB_BROKER_ADDR=127.0.0.1 ./bin/chat_repl

echo "Test completed"