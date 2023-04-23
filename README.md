# winchaos
## windows  chaos agent and experiment

### windows 交叉编译:
```bash
> CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build


> CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC="x86_64-w64-mingw32-gcc" go build -o agent.exe cmd/chaos_agent.go
```

### 运行程序
```bash
> ./chaosctl.sh install -k  0813d72a71ba41ed986e507e2e0ead1b  -p  chaos-default-app  -g  chaos-default-app-group  -P 19527    -t 192.168.123.93

> \agent.exe   --port 19527 --transport.endpoint 192.168.123.93 --license 0813d72a71ba41ed986e507e2e0ead1b --log.output stdout

or 
 doubleclick   start_agent.bat

```
