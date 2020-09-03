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
    * Info(format string, a ...interface{})
    * Warn(format string, a ...interface{})
    * Error(format string, a ...interface{})
    * Fatal(format string, a ...interface{})

## 命令行参数
```conf
-server.net = "tcp"
-server.addr = ""
-client.net = "tcp"
-client.addr = ""
-remux.reuse.max.sec = 60   使用时间达到该时长的连接不可重用
-remux.reuse.max.cnt = 16   重用次数达到该数量的连接不可重用
-remux.idle.max.sec = 10    闲置达到该时长的连接需要关闭
-remux.conn.max.cnt = 1024  同一时刻不可超过该连接数
-remux.net = "tls"          ReMux底层连接类型, 必须是net支持类型或"tls"
-eagersocks.net = "remux"   EagerSocks底层连接类型, 必须是net支持类型, "tls"或"remux"
-tls.server.key = ""
-tls.server.cert = ""
-tls.client.ca = ""
-tls.client.key = ""
-tls.client.cert = ""
-sla.log.level = "info"
-sla.log.path = "/var/log/tunnels.log"
-sla.dingtalk.token = ""
-sla.dingtalk.secret = ""
```

## 日志格式
```
2020.09.01 17:15:07 [info] [conn] + 8f7ed46a = 2
2020.09.01 17:15:07 [debg] [sess] + 8f7ed46a:1 = 1
2020.09.01 17:15:07 [info] [sock] + google.com:443 = 1
2020.09.01 17:15:12 [erro] [conn] - 8f7ed46a = 1 unexpected EOF
2020.09.01 17:15:12 [debg] [sess] - 8f7ed46a:1 = 0
2020.09.01 17:15:12 [info] [sock] - google.com:443 = 0
2020.09.01 17:15:39 [info] [conn] - 3905e1ca = 0
```

## 数据格式
```
Session编号  数据长度  数据
  uint16    uint16        Session编号为0表示无Session开始
  1-65535   0-65535       数据长度为0表示结束session
```

## 调度策略
* 当原有连接不可用时才建立新的连接

## 有限状态机
### TLS->SOCKS
```
socks <-
        \
socks <-- [buf] <- tls
        /
socks <-
```
* 读tls
    * 若tls读出错
        * conn.close()
* 写socks
    * 若socks写出错
        * sess.close(), 这样socks也会读出错, 由读来往tls里写0

### SOCKS->TLS
```
socks -> [buf] -
                \
socks -> [buf] --> tls
                /
socks -> [buf] -
```
* 读socks
    * 若socks读出错
        * 往tls里写0关闭
        * sess.close()
* 写tls
    * 需要加锁，防止一边写一边关
    * 若tls写出错
        * conn.close()

### 函数细节
* sess.write()
    * 和sess.close()同步，防止一边写一边关
* conn.closeSess()
    * 关掉sess的lowerConn，但不会往tls写0，由sess读线程负责写0
    * 保证只会执行一次
* conn.close()
    * 调用所有的sess.close()
    * 关掉conn的lowerConn，但不会往tls写0
    * 因为只有conn出错或没有sess了才需要close，conn出错的话对面也能感知到
    * 保证只会执行一次
