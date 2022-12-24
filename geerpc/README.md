# 消息编码

使用 encoding/gob 实现消息的编解码(序列化与反序列化)

将请求和响应中的参数和返回值抽象为body,剩余信息放在Header中

Header的抽象
```go
type Header struct {
	ServiceMethod string
	Seq           uint64  //requse ID
	Error         string
}
```