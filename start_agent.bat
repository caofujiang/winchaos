@echo off
set "DIR=C:\agent"
if not exist "%DIR%" (
    mkdir  C:\agent
    echo Created new directory "%DIR%".
)
cd   C:\agent
start  agent.exe   --port 19527 --transport.endpoint 192.168.123.93   --license 0813d72a71ba41ed986e507e2e0ead1b