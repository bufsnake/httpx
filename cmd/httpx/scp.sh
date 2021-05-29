CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-w -s' -gcflags '-N -l'
upx -9 httpx
scp httpx monkey:/root/bufsnake/httpx/
scp httpx bufsnake:/root/bufsnake/httpx/
scp httpx xiao13:/root/bufsnake/httpx/
