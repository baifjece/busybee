package expr

import (
	"fmt"
	"time"

	engine "github.com/fagongzi/expr"
	"github.com/fagongzi/log"
)

// func.name
type funcVar struct {
	valueType string
	dynaFunc  func() interface{}
}

func newFuncVar(name string, valueType string) (engine.Expr, error) {
	expr := &funcVar{
		valueType: valueType,
	}

	switch name {
	case "year":
		expr.dynaFunc = yearFunc
	case "month":
		expr.dynaFunc = monthFunc
	case "day":
		expr.dynaFunc = dayFunc
	default:
		return nil, fmt.Errorf("func %s not support", name)
	}

	return expr, nil
}

func (v *funcVar) Exec(data interface{}) (interface{}, error) {
	ctx, ok := data.(Ctx)
	if !ok {
		log.Fatalf("BUG: invalid expr ctx type %T", ctx)
	}

	value := v.dynaFunc()
	switch v.valueType {
	case stringVar:
		return toString(value)
	case int64Var:
		return toInt64(value)
	default:
		return nil, fmt.Errorf("not support var type %s", v.valueType)
	}
}

func yearFunc() interface{} {
	return int64(time.Now().Year())
}

func monthFunc() interface{} {
	return int64(time.Now().Month())
}

func dayFunc() interface{} {
	return int64(time.Now().Day())
}
