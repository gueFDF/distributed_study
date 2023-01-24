package ziface

import "io"

//封包数据和拆包数据,处理黏包问题的抽象

type IDataPack interface {
	GetHeadLen() uint32                            //获取包头长度方法
	Pack(msg IMessage) ([]byte, error)             //封包方法
	Unpack(reader io.ReadCloser) (IMessage, error) //拆包方法
}
