## 简介

> 判断 http/https 并截图

## Usage

```bash
Usage of ./httpx:
  -allow-jump
    	allow jump
  -chrome-path string
    	chrome browser path
  -cidr string
    	cidr file, example:
    	127.0.0.1
    	127.0.0.5-20
    	127.0.0.2-127.0.0.20
    	127.0.0.1/18
  -disable-screenshot
    	disable screenshot
  -display-error
    	display error
  -headless-proxy string
    	chrome browser proxy
  -output string
    	output database file name (default "202109132330.db")
  -path string
    	specify request path for probe or screenshot
  -proxy string
    	config probe proxy, example: http://127.0.0.1:8080
  -server
    	read the database by starting the web service
  -silent
    	silent output
  -target string
    	single target, example:
    	127.0.0.1
    	127.0.0.1:8080
    	http://127.0.0.1
  -targets string
    	multiple goals, examlpe:
    	127.0.0.1
    	127.0.0.1:8080
    	http://127.0.0.1
  -thread int
    	config probe thread (default 10)
  -timeout int
    	config probe http request timeout (default 10)
```

> example:

```bash
▶ cat domains.txt | ./httpx
```

```bash
▶ ./httpx -target http://127.0.0.1
```

```bash
▶ ./httpx -targets domains.txt
```

```bash
▶ ./httpx -output TEST.db -server # 启动服务并访问 http://127.0.0.1:9100/
```

## Screenshot

> 模板改自xray模板

![image-20210912210531051](.images/image-20210912210531051.png)

![image-20210723135945748](.images/image-20210723135945748.png)
