package gin

import "strings"

type node struct {
	pattern  string  //待匹配路由 eg:/p/:lang
	part     string  //路由中的一部分eg:lang
	children []*node //子节点,例如[doc,tutorial,intro]
	isWild   bool    //是否精确匹配，part含有 ：或 *时为true
}

// 返回第一个匹配成功的节点用于插入
func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		if child.part == part || child.isWild {
			return child
		}
	}
	return nil
}

// 返回所有匹配成功的节点
func (n *node) matchChildren(part string) []*node {
	nodes := make([]*node, 0)
	for _, child := range n.children {
		if child.part == part || child.isWild {
			nodes = append(nodes, child)
		}
	}
	return nodes
}

// 插入
func (n *node) insert(pattern string, parts []string, height int) {
	//匹配到最后一层，进行插入
	if len(parts) == height {
		n.pattern = pattern
		return
	}
	//当前层需要匹配的part
	part := parts[height]
	//进行匹配，只需要匹配一个
	chile := n.matchChild(part)
	if chile == nil { //没有匹配到，创建新节点进行插入
		chile = &node{part: part, isWild: part[0] == ':' || part[0] == '*'}
		n.children = append(n.children, chile)
	}
	chile.insert(pattern, parts, height+1)
}

// 访问
func (n *node) search(parts []string, height int) *node {
	//匹配到最后一层
	if len(parts) == height || strings.HasPrefix(n.part, "*") {
		if n.pattern == "" {
			return nil
		}
		return n
	}
	part := parts[height]
	children := n.matchChildren(part)
	for _, child := range children {
		result := child.search(parts, height+1)
		if result != nil {
			return result
		}
	}
	return nil
}
