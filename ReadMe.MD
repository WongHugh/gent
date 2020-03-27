#说明

###协议
基于gnet自定义协议

//增加固定头如 05 0a； 默认无头部
Header []byte

DecoderConfig
//增加尾部校验， 默认为累积和校验
FinalBytesToStrip bool


如需其它校验，需要重新实现如下两个方法

`type ICheck interface {
 	AddCheckData([]byte) (byte,error)
 	CheckData(data []byte) bool
 }`
