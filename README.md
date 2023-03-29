该仓库是学习分布式的仓库
1. geerpc     ----用go实现一个rpc框架
2. geecache   ----用go实现的一个分布式缓存
3. myweb      ----用go实现的一个轻量级的web框架

| project|Introduce |
|--|--|
|[geerpc](https://github.com/gueFDF/distributed_study/tree/main/geerpc) |用Go实现的一个rpc框架。此框架包括rpc服务端、支持并发的客户端以及一个简易的服务注册和发现中心；支持选择不同的序列化与反序列化方式；为防止服务挂死，添加了超时处理机制；支持 TCP、Unix、HTTP 等多种传输协议；支持多种负载均衡模式。|
|[geecache](https://github.com/gueFDF/distributed_study/tree/main/geecache)|用GO实现的一个分布式缓存系统，支持单机缓存和基于 HTTP 的分布式缓存，采用LRU缓存淘汰策略，使用 Go 锁机制防止缓存击穿，使用一致性哈希选择节点，实现负载均衡，使用 protobuf 优化节点间二进制通信。|
|[myweb](https://github.com/gueFDF/distributed_study/tree/main/mygin) |用Go实现的一个web框架。十分轻量级，采用trie前缀树实现了动态路由，支持中间件机制，实现了分组路由，支持模板

