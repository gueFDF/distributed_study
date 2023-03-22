package geecache
import pb "geecache/geecachepb"

// 抽象接口，根据PickPeer()方法，根据传入key选择对应节点PerrGetter
type PerrPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// Get()方法用于从对应group查找缓存值
type PeerGetter interface {
	Get(in *pb.Request, out *pb.Response) error
}
    