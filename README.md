
------
#### 0x01 road

改编自[pitaya v1.1.1](https://github.com/topfreegames/pitaya)版本的golang网络库。




为什么不继续使用pitaya？包括：

1. pitaya使用一个全局的handlerService对象，该对象会把所有接受到的消息，随机分发到一个goroutine中处理。这样做的一个结果是：来自同一个玩家的消息很可能在不同的goroutine处理，从而无法保证按接收消息的顺序处理，这在很多场景中是不可接受的。
2. pitaya使用了gorilla这个websocket库，其中的flushFrame()方法必须同步执行，否则会引发panic。目前pitaya中提供的Kick()等接口直接调用了WSConn的Write()方法，这有可能导致未知道的panic。
3. 当session.OnClose()回调方法中，如果意外调用session.Kick()方法会导致deadlock


因为，几经考虑，我决定基于pitaya的代码改一个简化的版本出来。



-----

#### 0x09 参考文献

1. [A Million WebSockets and Go](https://www.freecodecamp.org/news/million-websockets-and-go-cc58418460bb/)
2. https://github.com/eranyanay/1m-go-websockets
3. https://github.com/smallnest/epoller
4. https://github.com/smallnest/1m-go-tcp-server/

