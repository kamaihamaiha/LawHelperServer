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
scp data.tar.gz ubuntu@100.93.201.58:/opt/lawhelper/

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
After=network.target/

[Service]
WorkingDirectory=/opt/lawHelper
ExecStart=/opt/lawHelper/LawHelperServer
# 只允许本机访问，公网请求统一从 Nginx 进入
Environment=HTTP_ADDR=127.0.0.1:8080
Restart=always
RestartSec=5
n
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
curl http://127.0.0.1:8080/healthz

# 此时暂时只能在服务器本机测试，外部的 8080 端口不应再开放。

# 查看日志
journalctl -u lawHelper -f

如果服务之前已经按 `HTTP_ADDR=:8080` 部署，不需要重新编译。直接在服务器上修改 service 文件中的这一行：

```ini
Environment=HTTP_ADDR=127.0.0.1:8080
```

然后执行：

```shell
sudo systemctl daemon-reload
sudo systemctl restart lawHelper
curl http://127.0.0.1:8080/healthz
```

第 9 步：安装并配置 Nginx

以下命令以 Ubuntu/Debian 为例。先准备一个域名，例如 `law.example.com`，并在 DNS 中添加
A 记录指向腾讯云服务器公网 IP。没有域名时可以先用公网 IP 配置 HTTP，HTTPS 证书通常需要域名。

在服务器上安装 Nginx：

```shell
sudo apt update
sudo apt install -y nginx
```

创建站点配置：

```shell
sudo tee /etc/nginx/sites-available/lawHelper << 'EOF'
server {
    listen 80;
    listen [::]:80;
    server_name 100.93.201.58;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
EOF

sudo ln -s /etc/nginx/sites-available/lawHelper /etc/nginx/sites-enabled/lawHelper
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl enable nginx
sudo  
```

把配置中的 `law.example.com` 换成自己的域名。确认 DNS 已生效后，在服务器和其他电脑上验证：

```shell
curl http://127.0.0.1:8080/healthz       # Go 服务
curl http://101.43.43.3/healthz      # Nginx 代理
```
目前没有购买域名，也没有备案，因此无法开通https；所以先配置测试地址，

用 nginx 把 80 转发到 8080 //服务器，开了80端口，关闭8080端口
好处是访问时不用带端口号（http://你的IP 即可）。在服务器上：
sudo vim /etc/nginx/conf.d/lawhelper.conf

写入：

server {
listen 80;
server_name _;

      location / {
          proxy_pass http://127.0.0.1:8080;
          proxy_set_header Host $host;
          proxy_set_header X-Real-IP $remote_addr;
          proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
      }
}

然后：
```shell
sudo nginx -t
sudo systemctl reload nginx
```

第 10 步：配置 HTTPS（推荐）

确认域名已经解析到服务器，并且腾讯云安全组已放行 TCP `80` 后，在服务器上执行：

```shell
sudo apt install -y certbot python3-certbot-nginx
sudo certbot --nginx -d law.example.com
```

按提示选择将 HTTP 重定向到 HTTPS。证书自动续期检查：

```shell
sudo certbot renew --dry-run
```

完成后使用：

```shell
curl https://law.example.com/healthz
```

第 11 步：收紧腾讯云和服务器端口

在腾讯云安全组入站规则中保留：

- TCP `22`：仅允许你的固定公网 IP（如果条件允许）
- TCP `80`：用于 HTTP 或证书签发
- TCP `443`：用于 HTTPS

删除或禁用 TCP `8080` 的公网入站规则。因为 Go 已绑定到 `127.0.0.1:8080`，即使忘记删除安全组规则，
公网也不能直接连到 Go 服务。

如果服务器启用了 UFW，也同步执行：

```shell
sudo ufw allow OpenSSH
sudo ufw allow 'Nginx Full'
sudo ufw delete allow 8080/tcp  # 如果之前添加过
sudo ufw status
```

故障排查顺序：

```shell
sudo systemctl status lawHelper nginx
sudo nginx -t
curl http://127.0.0.1:8080/healthz
sudo tail -n 100 /var/log/nginx/error.log
journalctl -u lawHelper -n 100 --no-pager
```

如果修改了 `/etc/systemd/system/lawHelper.service`，需要重新加载并重启 Go 服务：

```shell
sudo systemctl daemon-reload
sudo systemctl restart lawHelper
```

  ---
user@你的服务器IP 替换成你实际的 SSH 连接信息。
