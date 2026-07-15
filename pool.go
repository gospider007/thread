package thread

import (
	"context"
	"errors"
	"sync"
	"time"
)

type groupContext struct {
	ctx context.Context
	cnl context.CancelFunc
}
type poolT interface {
	ShutdownContext() context.Context
	CloseWithError(error) error
}
type Pool[T poolT] struct {
	lock          sync.Mutex
	connPools     map[string]chan T
	groupContexts map[string]groupContext
	ctx           context.Context
	cnl           context.CancelFunc
}

func NewPool[T poolT](preCtx context.Context) *Pool[T] {
	if preCtx == nil {
		preCtx = context.TODO()
	}
	ctx, cnl := context.WithCancel(preCtx)
	return &Pool[T]{
		ctx:           ctx,
		cnl:           cnl,
		connPools:     make(map[string]chan T),
		groupContexts: make(map[string]groupContext),
	}
}

func (obj *Pool[T]) getPool(key string) chan T {
	obj.lock.Lock()
	defer obj.lock.Unlock()
	val, ok := obj.connPools[key]
	if ok {
		return val
	}
	val = make(chan T)
	obj.connPools[key] = val
	return val
}
func (obj *Pool[T]) SetGroupContext(group string) {
	if group == "" {
		return
	}
	obj.lock.Lock()
	defer obj.lock.Unlock()
	val, ok := obj.groupContexts[group]
	if ok {
		return
	}
	val = groupContext{}
	val.ctx, val.cnl = context.WithCancel(obj.ctx)
	obj.groupContexts[group] = val
}
func (obj *Pool[T]) getGroupContext(group string) context.Context {
	obj.lock.Lock()
	defer obj.lock.Unlock()
	val, ok := obj.groupContexts[group]
	if ok {
		return val.ctx
	}
	return nil
}
func (obj *Pool[T]) Context() context.Context {
	return obj.ctx
}
func (obj *Pool[T]) Close() {
	obj.cnl()
}
func (obj *Pool[T]) CloseContext(group string) {
	obj.lock.Lock()
	defer obj.lock.Unlock()
	val, ok := obj.groupContexts[group]
	if ok {
		val.cnl()
		delete(obj.groupContexts, group)
	}
}

func (obj *Pool[T]) GetConn(group string, key string) (T, error) {
	var t T
	if group != "" {
		if c := obj.getGroupContext(group); c == nil || c.Err() != nil {
			return t, errors.New("group closed")
		}
	}
	for {
		select {
		case conn := <-obj.getPool(key):
			select {
			case <-conn.ShutdownContext().Done():
			default:
				return conn, nil
			}
		default:
			return t, nil
		}
	}
}
func (obj *Pool[T]) PutConn(group string, key string, conn T, timeout time.Duration) {
	var groupCtx context.Context
	if group == "" {
		groupCtx = obj.ctx
	} else {
		if groupCtx = obj.getGroupContext(group); groupCtx == nil {
			conn.CloseWithError(nil)
			return
		}
	}
	if timeout == 0 {
		timeout = time.Second * 30
	}
	go func() {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		select {
		case <-groupCtx.Done():
			conn.CloseWithError(groupCtx.Err())
		case <-conn.ShutdownContext().Done():
			conn.CloseWithError(conn.ShutdownContext().Err())
		case <-timer.C:
			conn.CloseWithError(nil)
		case obj.getPool(key) <- conn:
		}
	}()
}
