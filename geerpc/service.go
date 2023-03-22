package geerpc

import (
	"go/ast"
	"log"
	"reflect"
	"sync/atomic"
)

// TODO :通过反射实现service

type methodType struct {
	method    reflect.Method //方法本身
	ArgType   reflect.Type   //第一个参数类型
	ReplyType reflect.Type   //第二个参数类型
	numCalls  uint64         //方法调用次数
}
 

                                                                                                                      
func (m *methodType) NumCalls() uint64 {
	return atomic.LoadUint64(&m.numCalls)
}

func (m *methodType) newArgv() reflect.Value {
	var argv reflect.Value
	//arg may be a pointer type ,or a value type
	if m.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(m.ArgType.Elem())
	} else {
		argv = reflect.New(m.ArgType).Elem()
	}
	return argv
}

func (m *methodType) newReplyv() reflect.Value {
	//必须是指针
	replyv := reflect.New(m.ReplyType.Elem())
	switch m.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(m.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(m.ReplyType.Elem(), 0, 0))
	}
	return replyv
}

type service struct {
	name   string        //服务名
	typ    reflect.Type  //类型
	rcvr   reflect.Value //实例本身
	method map[string]*methodType
}

func newService(rcvr interface{}) *service {
	s := new(service)
	s.rcvr = reflect.ValueOf(rcvr)
	s.name = reflect.Indirect(s.rcvr).Type().Name()
	s.typ = reflect.TypeOf(rcvr)

	//判断该服务名是否可访问（是否开头大写）
	if !ast.IsExported(s.name) {
		log.Fatalf("rpc server: %s is not a valid service name", s.name)
	}
	s.registerMethods()
	return s
}

func (s *service) registerMethods() {
	s.method = make(map[string]*methodType)
	for i := 0; i < s.typ.NumMethod(); i++ {
		method := s.typ.Method(i)
		mType := method.Type
		//入参包括自己一共要有三个（argv和replyv），返回值一个(err)
		if mType.NumIn() != 3 || mType.NumOut() != 1 {
			continue
		} 
		//返回值必须是error
		if mType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}
		//获取参数类型
		argType, replyType := mType.In(1), mType.In(2)

		if !IsExportedOrBuildinType(argType) || !IsExportedOrBuildinType(replyType) {
			continue
		}
		s.method[method.Name] = &methodType{
			method:    method,
			ArgType:   argType,
			ReplyType: replyType,
		}
		log.Printf("rpc sever:register%s.%s\n", s.name, method.Name)

	}
}

func IsExportedOrBuildinType(t reflect.Type) bool {
	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}

// 调用方法
func (s *service) call(m *methodType, argv, replyv reflect.Value) error {
	atomic.AddUint64(&m.numCalls, 1)
	f := m.method.Func

	returnValues := f.Call([]reflect.Value{s.rcvr, argv, replyv})
	if errInter := returnValues[0].Interface(); errInter != nil {
		return errInter.(error)
	}
	
	return nil
}


