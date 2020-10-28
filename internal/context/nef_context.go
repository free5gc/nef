package context

type NefContext struct {
	afCtx map[string]*afContext
}

type afContext struct {
	subsc    map[string]*subscription
	pfdTrans map[string]*pfdTransaction
}

type subscription struct {
}

type pfdTransaction struct {
}

func NewNefContext() *NefContext {
	n := &NefContext{}
	n.afCtx = make(map[string]*afContext)
	return n
}

