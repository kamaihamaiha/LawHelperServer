## 发布

- 云服务器: 
  - tailscale 地址: 100.93.201.58 // Lhs@T_2026

第 1 步：打包源码

cd /Users/zhangkx/dev/code/my_code/LawHelperServer
tar czf LawHelperServer-src.tar.gz --exclude='.git' --exclude='data' --exclude='.claude' --exclude='docs' .

▎ 排除了 .git 和 data 目录（data 稍后单独传）。

第 2 步：上传到服务器

scp LawHelperServer-src.tar.gz user@你的服务器IP:/opt/lawhelper/

第 3 步：SSH 登录服务器

ssh user@你的服务器IP

第 4 步：在服务器上解压并编译

cd /opt/lawhelper
tar xzf LawHelperServer-src.tar.gz

# 确认有 Go 环境
go version

# 如果没有 Go，安装：
# 
# sudo tar -C /usr/local -xzf go1.26.1.linux-amd64.tar.gz
# export PATH=$PATH:/usr/local/go/bin

# 编译
CGO_ENABLED=1 go build -o LawHelperServer .
## 设置国内代理 go env -w GOPROXY=https://goproxy.cn,direct

第 5 步：上传 data 目录

回到本机，另开一个终端：
如果文件太多，先压缩: tar czf data.tar.gz data/
scp data.tar.gz ubuntu@100.93.201.58:/opt/lawHelper/

然后在服务器上解压：

cd /opt/lawHelper
tar xzf data.tar.gz

cd /Users/zhangkx/dev/code/my_code/LawHelperServer
scp -r data/ user@你的服务器IP:/opt/lawHelper/

▎ 现在阶段 law_json 一起传上去，后面再改 Tailscale 远程读取。

第 6 步：创建 systemd 服务

在服务器上：

sudo tee /etc/systemd/system/lawHelper.service << 'EOF'
[Unit]
Description=Law Helper Server
After=network.target

[Service]
WorkingDirectory=/opt/lawHelper
ExecStart=/opt/lawHelper/LawHelperServer
Environment=HTTP_ADDR=:8080
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

第 7 步：启动服务

sudo systemctl daemon-reload
sudo systemctl enable lawHelper
sudo systemctl start lawHelper
sudo systemctl status lawHelper   # 确认 running

第 8 步：验证

# 服务器上测试
curl http://localhost:8080/api/v1/laws/types

# 其他电脑上测试
http://101.43.43.3:8080/healthz

# 查看日志
journalctl -u lawHelper -f

  ---
user@你的服务器IP 替换成你实际的 SSH 连接信息。