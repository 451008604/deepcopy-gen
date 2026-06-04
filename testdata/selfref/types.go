package selfref

type TreeNode struct {
	Value    int
	Children []*TreeNode
	Parent   *TreeNode
}

type GraphNode struct {
	Name     string
	Neighbors []*GraphNode
	Metadata  map[string]interface{}
}

type LinkedNode struct {
	Value int
	Next  *LinkedNode
}

type TreeWithMap struct {
	Label    string
	Children map[string]*TreeWithMap
}
