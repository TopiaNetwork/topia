package rpc

import (
	"github.com/TopiaNetwork/topia/eventhub"
	"reflect"
	"strings"
)

// filterString应该做个长度限制

// 1. 收到订阅请求，词法分析 (判断是否含有次字段，判断操作符是否是那几个之一，判断) 检查filterString是否合法
// 2.

type filter struct {
	eventName  string
	clauses    [][]condition // AND connects condition to []condition (clause), OR connects []condition to [][]condition
	directPass bool          //
}

func newFilter(eventName string, filterString string) (*filter, error) {
	p := new(parser)
	f, err := p.parseFilterString(eventName, filterString)
	if err != nil {
		return nil, err
	}

}

func (f *filter) pass(msg eventhub.EventMsg) bool {
	if f.directPass {
		return true
	}

	// msg.Data 应是结构体或结构体指针，否则应做特别处理
	var evDataV reflect.Value
	evDataT := reflect.TypeOf(msg.Data)
	if evDataT.Kind() == reflect.Ptr {
		evDataV = reflect.ValueOf(msg.Data).Elem()
		evDataT = reflect.TypeOf(msg.Data).Elem()
	} else {
		evDataV = reflect.ValueOf(msg.Data)
	}

	for _, clause := range f.clauses { // any clause pass leads to msg pass

		var clausePass = true

		for _, cond := range clause {
			if !cond.pass(evDataT,evDataV) {
				clausePass = false
				break
			}
		}

		if clausePass {
			return true
		}

	}
	return false
}

// filter condition
type condition struct {
	field    string // fieldName of eventMsg.Data
	operator string //
	data     interface{}
	dataType reflect.Kind
}

func (c *condition) pass(evDataT reflect.Type, evDataV reflect.Value) bool {
	// msg.Data 可能是个指针，也可能是个结构体。condition 所对应的field可能在结构体（指针）的内层

	// 1. 反射msg, 根据c.field名称找到对应字段
	// 2. 判断c.operator,

	// 根据字段名获取字段
	for i := 0; i < evDataT.NumField(); i++ {
		if strings.ToLower(evDataT.Field(i).Name) == c.field {
			if evDataV.Field(i).Kind() == c.dataType {
				switch c.dataType {
				case reflect.Int:
					evDataV.Field(i).Int()

				}

			}
			
		} else if
	}

}



type parser struct {
	i            int
	filterString string
}

func (p *parser) parseFilterString(eventName string, filterString string) (*filter, error) {
	if len(strings.TrimSpace(filterString)) == 0 {
		return &filter{eventName: eventName, directPass: true}, nil
	}
	// TODO 检查 filterString 长度

	p.filterString = filterString

}
