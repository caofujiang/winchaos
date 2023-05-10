# winchaos
## windows  chaos agent and experiment

### windows 交叉编译:
```bash
编译 os.exe:
> CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC="x86_64-w64-mingw32-gcc" go build -o os.exe web/category/event/cmd.go
编译 agent.exe:
> CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC="x86_64-w64-mingw32-gcc" go build -o agent.exe cmd/chaos_agent.go
```

### 运行程序
```bash
linux:
> ./chaosctl.sh install -k  0813d72a71ba41ed986e507e2e0ead1b  -p  chaos-default-app  -g  chaos-default-app-group  -P 19527    -t 192.168.123.93
windows-server:
> .\agent.exe   --port 19527 --transport.endpoint 192.168.123.93 --license 0813d72a71ba41ed986e507e2e0ead1b --log.output stdout

or 
 doubleclick   start_agent.bat

```
## 实验接口
#### 1、执行.bat和ps1的接口
```
curl --location 'http://192.168.123.214:19527/chaosblade' \
--header 'Content-Type: application/x-www-form-urlencoded' \
--data '{
"params": {
"cmd": "create script execute",
"ts": "601366376892",
"cmd2": "script-execute",
"downloadUrl": "http://192.168.123.93/blade-cps/script/download/custom/hello.ps1",
"file-args": "arg1:arg2:333333:44455",
"timeout": "20"
}
}'

返回：
    {"Code":200,"Success":true,"Error":"","Result":"f79ffd7e00d67694"}
``` 
#### 2、查询所有实验结果的接口

可以查询所有实验的接口，也可以根据uid查询单个实验接口:

```
curl --location 'http://localhost:19527/chaosblade' \
--header 'Content-Type: application/x-www-form-urlencoded' \
--data '{
    "params":
       {
          "cmd":"status --type create",
          "cmd2":"status",
          "uid": "1688571c2b9904e1",
          "limit": "5",
          "status":"",   //Destroyed ,Created,Success,Error
          "asc": "true"
      }
}'

返回：
   {
    "Code": 200,
    "Success": true,
    "Error": "",
    "Result": {
        "Uid": "1688571c2b9904e1",
        "Command": "create script execute",
        "CmdType": "script",
        "SubCommand": "/Users/admin/code/winchaos/2.bat",
        "Flag": "",
        "Status": "Destroyed",
        "Error": "",
        "CreateTime": "2023-04-23T19:15:17.274552+08:00",
        "UpdateTime": "2023-04-23T19:16:18.558339+08:00"
    }
}
``` 
#### 3、执行cpu演练的接口
```
curl --location 'http://192.168.123.214:19527/chaosblade' \
--header 'Content-Type: application/x-www-form-urlencoded' \
--data '{
    "params":{
        "cmd": "create cpu fullload --cpu-percent 80 --timeout 120",
        "ts": "601366376892",
        "cmd2": "cpu-fullload",
        "cpu-count": "1",
        "cpu-percent": "10",
        "timeOut": "20"
    }
}'

返回：
{"Code":200,"Success":true,"Error":"","Result":"8b75e5311b78df4e"}

```

#### 4、执行destroy演练的接口
```
curl --location 'http://localhost:19527/chaosblade' \
--header 'Content-Type: application/x-www-form-urlencoded' \
--data '{
    "params": 
      {
        "cmd":"destroy 7aee71baa484bf44"
      }
}'
返回：
  {"Code":200,"Success":true,"Error":"","Result":"success"}

```

#### 5. 执行mem cache演练接口
```markdown
curl --location --request POST 'http://192.168.123.214:19527/chaosblade' \
--header 'Content-Type: application/x-www-form-urlencoded' \
--data-raw '{
    "params":{
        "cmd": "create create mem load --mode ram --mem-percent 50 --timeout 120",
        "ts": "601366376892",
        "cmd2": "mem-load",
        "mode": "cache",
        "mem-percent": "50",
        "timeout": "50"
    }
}'
返回：
{"Code":200,"Success":true,"Error":"","Result":"f820b85136709cb6"}
```


#### 6.执行destroy演练的接口
```markdown
curl --location --request POST 'http://192.168.123.214:19527/chaosblade' \
--header 'Content-Type: application/x-www-form-urlencoded' \
--data-raw '{
    "params": 
      {
        "cmd":"destroy 23903fa200f9f9b8"
      }
}'
返回：
{"Code":200,"Success":true,"Error":"","Result":"success"}
```

#### 7. 执行mem ram演练接口

```markdown
curl --location --request POST 'http://192.168.123.214:19527/chaosblade' \
--header 'Content-Type: application/x-www-form-urlencoded' \
--data-raw '{
"params":{
"cmd": "create create mem load --mode ram --mem-percent 50 --timeout 120",
"ts": "601366376892",
"cmd2": "mem-load",
"mode": "ram",
"mem-percent": "50",
"timeout": "50"
}
}'

返回：
{"Code":200,"Success":true,"Error":"","Result":"bf3f3fee5f28f498"}
```

#### 6.执行destroy ram演练的接口
```markdown
curl --location --request POST 'http://192.168.123.214:19527/chaosblade' \
--header 'Content-Type: application/x-www-form-urlencoded' \
--data-raw '{
"params":
{
"cmd":"destroy bf3f3fee5f28f498"
}
}'
返回：
{"Code":200,"Success":true,"Error":"","Result":"success"}
```

