package cuba

type Bucket[I any] interface {
	Push(I)
	PushAll([]I)
	Pop() I
	IsEmpty() bool
	Empty()
}

type Stack[D any] struct {
	data []D
}

func NewStack[D any]() *Stack[D] {
	return &Stack[D]{}
}

func (stack *Stack[D]) Push(item D) {
	stack.data = append(stack.data, item)
}

func (stack *Stack[D]) PushAll(items []D) {
	stack.data = append(stack.data, items...)
}

func (stack *Stack[D]) Pop() D {
	item := stack.data[len(stack.data)-1]
	stack.data = stack.data[:len(stack.data)-1]
	return item
}

func (stack *Stack[D]) IsEmpty() bool {
	return len(stack.data) == 0
}

func (stack *Stack[D]) Empty() {
	stack.data = nil
}

type Queue[D any] struct {
	data *List[D]
}

func NewQueue[D any]() *Queue[D] {
	return &Queue[D]{
		data: NewList[D]().Init(),
	}
}

func (queue *Queue[D]) Push(item D) {
	queue.data.PushBack(item)
}

func (queue *Queue[D]) PushAll(items []D) {
	for _, item := range items {
		queue.Push(item)
	}
}

func (queue *Queue[D]) Pop() D {
	return queue.data.Remove(queue.data.Front())
}

func (queue *Queue[D]) IsEmpty() bool {
	return queue.data.Len() == 0
}

func (queue *Queue[D]) Empty() {
	queue.data = NewList[D]().Init()
}
