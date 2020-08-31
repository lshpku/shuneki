# TunneLS
基于TLS的可重用多路复用急切Socks隧道

## 概念说明

### ReMux (Reusable-multiplexing) 可重用多路复用
* 基于TLS的可重用和多路复用的流
* 每当客户端发起一个请求，会根据策略重用或新建一个底层流，并进行多路复用

### EagerSocks 急切Socks
* 原生Socks协议要求客户端和服务器先握手再传数据，这在网络状况不佳时会大大增加延迟
* EagerSocks取消了握手，默认连接成功，客户端一连接就可以传数据

## 模块
* main
    * Client interface
        * ListenAndServe() error
    * Server interface
        * ListenAndServe() error
    * NewClient() Client
    * NewServer() Server
* remux
    * Dialer interface
        * Dial() (io.ReadWriteCloser, error)
    * Listener interface
        * Accept() (io.ReadWriteCloser, error)
        * Close() error
    * NewDialer(network, address string) (Dialer, error)
    * Listen(network, address string) (Listener, error)
* eagersocks
    * Request struct
    * Listener interface
        * Accept() (io.ReadWriteCloser, *Request, error)
        * Close() error
    * Listen(network, address string) (Listener, error)
    * ListenSocks(network, address string) (Listener, error)
* sla
    * IncTimeout(err error)
    * IncConn(connId uint32, total int)
    * DecConn(connId uint32, total int)
    * IncSess(connId uint32, sessId uint16, addr string, total int)
    * DecSess(connId uint32, sessId uint16, addr string, total int)

## 命令行参数
```
-server.net
-server.addr
-client.net
-client.addr
-remux.reuse.max.sec = 60   使用时间达到该时长的连接不可重用
-remux.reuse.max.cnt = 16   重用次数达到该数量的连接不可重用
-remux.idle.max.sec  = 10   闲置超过该时长的连接需要关闭
-remux.conn.max.cnt  = 1024 不可超过该连接数
-tls.server.key
-tls.server.cert
-tls.client.ca
-tls.client.key
-tls.client.cert
-sla.dingtalk.token
-sla.dingtalk.secret
```

## 日志格式
```
CONN + 84ad76fb = 8
SESS + 84ad76fb:1 google.com:443 = 1
SESS + 3321d0e6:9 youtube.com:443 = 4
SESS + 84ad76fb:2 yahoo.com:80 = 2
SESS - 84ad76fb:1 google.com:443 = 1
SESS - 84ad76fb:2 yahoo.com:80 = 0
CONN - 84ad76fb = 7
```
* 日志均由Client记录

## 数据格式
```
Session编号  数据长度  数据
  uint16    uint16        Session编号为0表示无Session开始
  1-65535   0-65535       数据长度为0表示结束session
```

## 调度策略
* 当原有连接不可用时才建立新的连接
* 连接和Session均由Client管理
