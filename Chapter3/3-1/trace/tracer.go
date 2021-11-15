package trace

import (
	"fmt"
	"io"
)

//	Tracerはコード内の出来事を記録できるオブジェクトを表すインターフェースです。
type Tracer interface {
	//	任意の型の引数を何個でも（ゼロ個でも可）受け取ることができる。
	Trace(...interface{})
}

type tracer struct {
	out io.Writer
}

type nilTracer struct {

}

func (t *nilTracer) Trace(a ...interface{}) {

}

func Off() Tracer {
	return &nilTracer{}
}

func (t *tracer) Trace(a ...interface{}) {
	t.out.Write([]byte(fmt.Sprint(a...)))
	t.out.Write([]byte("\n"))
}

func New(w io.Writer) Tracer {
	return &tracer{out: w}
}
