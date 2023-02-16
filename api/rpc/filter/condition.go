package filter

import (
	"bytes"
	"reflect"
	"strings"
)

// filter condition
type condition struct {
	field    string   // lower cased fieldName of eventMsg.Data
	operator operator //
	data     interface{}
	dataType reflect.Kind
}

func (c *condition) pass(evDataT reflect.Type, evDataV reflect.Value) bool {
	// msg.Data 可能是个指针，也可能是个结构体。condition 所对应的field可能在结构体（指针）的内层

	// 1. 反射msg, 根据c.field名称找到对应字段
	// 2. 判断c.operator,

	// 根据字段名获取字段
	for i := 0; i < evDataT.NumField(); i++ {
		if evDataV.Field(i).Kind() == reflect.Struct {
			innerFieldV := evDataV.Field(i)
			return c.pass(innerFieldV.Type(), innerFieldV)
		} else if evDataV.Field(i).Kind() == reflect.Ptr && evDataV.Field(i).Elem().Kind() == reflect.Struct {
			innerFieldV := evDataV.Field(i).Elem()
			return c.pass(innerFieldV.Type(), innerFieldV)
		} else {
			if strings.ToLower(evDataT.Field(i).Name) == c.field {
				if c.operator == Exists {
					return true
				}

				if evDataV.Field(i).Kind() == c.dataType {
					switch c.dataType {
					case reflect.Int:
						switch c.operator {
						case Eq:
							return evDataV.Field(i).Int() == int64(c.data.(int))
						case Ne:
							return evDataV.Field(i).Int() != int64(c.data.(int))
						case Gt:
							return evDataV.Field(i).Int() > int64(c.data.(int))
						case Lt:
							return evDataV.Field(i).Int() < int64(c.data.(int))
						case Gte:
							return evDataV.Field(i).Int() >= int64(c.data.(int))
						case Lte:
							return evDataV.Field(i).Int() <= int64(c.data.(int))
						default:
							return false
						}
					case reflect.Int8:
						switch c.operator {
						case Eq:
							return evDataV.Field(i).Int() == int64(c.data.(int8))
						case Ne:
							return evDataV.Field(i).Int() != int64(c.data.(int8))
						case Gt:
							return evDataV.Field(i).Int() > int64(c.data.(int8))
						case Lt:
							return evDataV.Field(i).Int() < int64(c.data.(int8))
						case Gte:
							return evDataV.Field(i).Int() >= int64(c.data.(int8))
						case Lte:
							return evDataV.Field(i).Int() <= int64(c.data.(int8))
						default:
							return false
						}
					case reflect.Int16:
						switch c.operator {
						case Eq:
							return evDataV.Field(i).Int() == int64(c.data.(int16))
						case Ne:
							return evDataV.Field(i).Int() != int64(c.data.(int16))
						case Gt:
							return evDataV.Field(i).Int() > int64(c.data.(int16))
						case Lt:
							return evDataV.Field(i).Int() < int64(c.data.(int16))
						case Gte:
							return evDataV.Field(i).Int() >= int64(c.data.(int16))
						case Lte:
							return evDataV.Field(i).Int() <= int64(c.data.(int16))
						default:
							return false
						}
					case reflect.Int32:
						switch c.operator {
						case Eq:
							return evDataV.Field(i).Int() == int64(c.data.(int32))
						case Ne:
							return evDataV.Field(i).Int() != int64(c.data.(int32))
						case Gt:
							return evDataV.Field(i).Int() > int64(c.data.(int32))
						case Lt:
							return evDataV.Field(i).Int() < int64(c.data.(int32))
						case Gte:
							return evDataV.Field(i).Int() >= int64(c.data.(int32))
						case Lte:
							return evDataV.Field(i).Int() <= int64(c.data.(int32))
						default:
							return false
						}
					case reflect.Int64:
						switch c.operator {
						case Eq:
							return evDataV.Field(i).Int() == c.data.(int64)
						case Ne:
							return evDataV.Field(i).Int() != c.data.(int64)
						case Gt:
							return evDataV.Field(i).Int() > c.data.(int64)
						case Lt:
							return evDataV.Field(i).Int() < c.data.(int64)
						case Gte:
							return evDataV.Field(i).Int() >= c.data.(int64)
						case Lte:
							return evDataV.Field(i).Int() <= c.data.(int64)
						default:
							return false
						}
					case reflect.Uint:
						switch c.operator {
						case Eq:
							return evDataV.Field(i).Uint() == uint64(c.data.(uint))
						case Ne:
							return evDataV.Field(i).Uint() != uint64(c.data.(uint))
						case Gt:
							return evDataV.Field(i).Uint() > uint64(c.data.(uint))
						case Lt:
							return evDataV.Field(i).Uint() < uint64(c.data.(uint))
						case Gte:
							return evDataV.Field(i).Uint() >= uint64(c.data.(uint))
						case Lte:
							return evDataV.Field(i).Uint() <= uint64(c.data.(uint))
						default:
							return false
						}
					case reflect.Uint8:
						switch c.operator {
						case Eq:
							return evDataV.Field(i).Uint() == uint64(c.data.(uint8))
						case Ne:
							return evDataV.Field(i).Uint() != uint64(c.data.(uint8))
						case Gt:
							return evDataV.Field(i).Uint() > uint64(c.data.(uint8))
						case Lt:
							return evDataV.Field(i).Uint() < uint64(c.data.(uint8))
						case Gte:
							return evDataV.Field(i).Uint() >= uint64(c.data.(uint8))
						case Lte:
							return evDataV.Field(i).Uint() <= uint64(c.data.(uint8))
						default:
							return false
						}
					case reflect.Uint16:
						switch c.operator {
						case Eq:
							return evDataV.Field(i).Uint() == uint64(c.data.(uint16))
						case Ne:
							return evDataV.Field(i).Uint() != uint64(c.data.(uint16))
						case Gt:
							return evDataV.Field(i).Uint() > uint64(c.data.(uint16))
						case Lt:
							return evDataV.Field(i).Uint() < uint64(c.data.(uint16))
						case Gte:
							return evDataV.Field(i).Uint() >= uint64(c.data.(uint16))
						case Lte:
							return evDataV.Field(i).Uint() <= uint64(c.data.(uint16))
						default:
							return false
						}
					case reflect.Uint32:
						switch c.operator {
						case Eq:
							return evDataV.Field(i).Uint() == uint64(c.data.(uint32))
						case Ne:
							return evDataV.Field(i).Uint() != uint64(c.data.(uint32))
						case Gt:
							return evDataV.Field(i).Uint() > uint64(c.data.(uint32))
						case Lt:
							return evDataV.Field(i).Uint() < uint64(c.data.(uint32))
						case Gte:
							return evDataV.Field(i).Uint() >= uint64(c.data.(uint32))
						case Lte:
							return evDataV.Field(i).Uint() <= uint64(c.data.(uint32))
						default:
							return false
						}
					case reflect.Uint64:
						switch c.operator {
						case Eq:
							return evDataV.Field(i).Uint() == uint64(c.data.(uint64))
						case Ne:
							return evDataV.Field(i).Uint() != uint64(c.data.(uint64))
						case Gt:
							return evDataV.Field(i).Uint() > uint64(c.data.(uint64))
						case Lt:
							return evDataV.Field(i).Uint() < uint64(c.data.(uint64))
						case Gte:
							return evDataV.Field(i).Uint() >= uint64(c.data.(uint64))
						case Lte:
							return evDataV.Field(i).Uint() <= uint64(c.data.(uint64))
						default:
							return false
						}
					case reflect.Float32:
						switch c.operator {
						case Eq:
							return evDataV.Field(i).Float() == float64(c.data.(float32))
						case Ne:
							return evDataV.Field(i).Float() != float64(c.data.(float32))
						case Gt:
							return evDataV.Field(i).Float() > float64(c.data.(float32))
						case Lt:
							return evDataV.Field(i).Float() < float64(c.data.(float32))
						case Gte:
							return evDataV.Field(i).Float() >= float64(c.data.(float32))
						case Lte:
							return evDataV.Field(i).Float() <= float64(c.data.(float32))
						default:
							return false
						}
					case reflect.Float64:
						switch c.operator {
						case Eq:
							return evDataV.Field(i).Float() == c.data.(float64)
						case Ne:
							return evDataV.Field(i).Float() != c.data.(float64)
						case Gt:
							return evDataV.Field(i).Float() > c.data.(float64)
						case Lt:
							return evDataV.Field(i).Float() < c.data.(float64)
						case Gte:
							return evDataV.Field(i).Float() >= c.data.(float64)
						case Lte:
							return evDataV.Field(i).Float() <= c.data.(float64)
						default:
							return false
						}
					case reflect.String:
						switch c.operator {
						case Eq:
							return evDataV.Field(i).String() == c.data.(string)
						case Ne:
							return evDataV.Field(i).String() != c.data.(string)
						case Contains:
							return strings.Contains(evDataV.Field(i).String(), c.data.(string))
						default:
							return false
						}
					case reflect.Slice:
						if evDataV.Field(i).Type().Elem().Kind() == reflect.Uint8 { // []byte
							switch c.operator {
							case Eq:
								return bytes.Equal(evDataV.Field(i).Bytes(), c.data.([]byte))
							case Ne:
								return !bytes.Equal(evDataV.Field(i).Bytes(), c.data.([]byte))
							case Contains:
								return strings.Contains(string(evDataV.Field(i).Bytes()), string(c.data.([]byte)))
							default:
								return false
							}
						}
					}

				}

			}
		}

	}

	return false

}
