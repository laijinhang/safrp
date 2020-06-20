# safrp cn/[en](#)
safrp（Simple and fast reverse proxy）是基于Go语言开发的一个轻量级和可定制化功能的内网穿透软件，支持 TCP、UDP、SSH、HTTP、HTTPS、WebSocket 协议的数据转发。
可根据场景在服务端或内网中进行限流、IP请求记录的内网穿透软件


该项目的目的：借助廉价的带公网IP的云服务器使得内网闲置电脑能够扩充它的用处

目前正在开发v0.3.0版本

### 功能
目前仅实现HTTP协议数据的转发
### v0.3.1（计划）
1. 支持配置一个服务端与多个客户端

2. 一个客户端支持配置多种服务

3. safrp客户端和safrp服务端功能插件化，可支持自定义功能开发，如：自定义限流功能、自定义IP记录、IP黑名单、IP白名单等其他自定义插件
### v0.3.1（开发中）
1. 支持udp协议数据转发
### [v0.3.0](https://github.com/laijinhang/safrp/releases/tag/v0.3.0)
1. 支持配置一个服务端与多个客户端

2. 一个客户端支持配置一种服务

3. safrp客户端和safrp服务端功能插件化，可支持自定义功能开发，如：自定义限流功能、自定义IP记录、IP黑名单、IP白名单等其他自定义插件
### [v0.2.0](https://github.com/laijinhang/safrp/releases/tag/v0.2.0)
1. 一个服务端只能服务于一个客户端

2. safrp服务端与safrp客户端支持多条连接复用

### [v0.1.0](https://github.com/laijinhang/safrp/releases/tag/v0.1.0)
1. 一个服务端只能服务于一个客户端

2. safrp服务端与safrp客户端之间的通信只复用一条连接

### 上手指南
##### 1. 一个服务端与一个客户端
1、服务端配置
编辑server/safrp.ini文件
```
[pipe1]
ip=0.0.0.0
extranet_port=对外端口
server_port=对safrp客户端连接的端口
protocol=支持协议
pipe_num=30
password=服务端密码
```
2、客户端配置
编辑client/client.ini文件
```
[server]
ip=127.0.0.1
port=8002
password=123456

[http]
ip=0.0.0.0
port=8888
```
##### 2. 一个服务端与多个服务端 

1、服务端配置
编辑server/safrp.ini文件
```
[pipe1]
ip=0.0.0.0
extranet_port=对外端口
server_port=对safrp客户端连接的端口
protocol=支持协议
pipe_num=30
password=服务端密码

[pipe2]
ip=0.0.0.0
extranet_port=对外端口
server_port=对safrp客户端连接的端口
protocol=支持协议
pipe_num=30
password=服务端密码
```
2、客户端1配置
编辑client/client.ini文件
```
[server]
ip=safrp服务端地址
port=8002
password=123456

[http]
ip=0.0.0.0
port=8888
```
3、客户端2配置
编辑client/client.ini文件
```
[server]
ip=safrp服务端地址
port=8003
password=123456

[http]
ip=0.0.0.0
port=8888
```
### 安装要求
windows、类unix、mac系统
### 安装步骤
### 测试
### 部署
### 分支说明
* master：基础稳定的通用版内网穿透软件
* dev：master的开发版本
* release：在线上服务的版本，具体的解决方案
