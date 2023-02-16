package filter

import (
	"errors"
	"fmt"
	"github.com/TopiaNetwork/topia/eventhub"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type filter struct {
	eventName  string
	clauses    [][]condition // AND connects condition to []condition (clause), OR connects []condition to [][]condition
	directPass bool          //
}

func NewFilter(eventName string, filterString string) (*filter, error) {
	p := new(parser)
	f, err := p.parseFilterString(eventName, filterString)
	if err != nil {
		return nil, err
	}

	err = f.completeAndValidateConditions()
	if err != nil {
		return nil, err
	}

	return f, nil

}

func (f *filter) Pass(msg eventhub.EventMsg) bool {
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
			if !cond.pass(evDataT, evDataV) {
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

//
func (f *filter) completeAndValidateConditions() error {
	for i, clause := range f.clauses {
		for j, cond := range clause {
			kind, ok := isValidFieldName(f.eventName, cond.field)
			if !ok {
				return errors.New("not a valid field name or eventMsg not been registered")
			}

			legal := cond.operator.isLegal()
			if !legal {
				return errors.New("illegal operator")
			}

			var dataString string
			if cond.operator != Exists {
				dataString, ok = cond.data.(string)
				if !ok {
					return errors.New("unexpected data type")
				}
			} else { // operator is Exists
				continue // no need to complete data and dataType
			}

			switch kind {
			case reflect.Int:
				intNum, err := strconv.Atoi(dataString)
				if err != nil {
					return err
				}
				if cond.operator == Contains || cond.operator == Exists {
					return fmt.Errorf("dataType: %s cannot support operation: %s", kind.String(), cond.operator.string())
				}
				f.clauses[i][j].data = intNum
			case reflect.Int8:
				parsedInt, err := strconv.ParseInt(dataString, 10, 8)
				if err != nil {
					return err
				}
				if cond.operator == Contains || cond.operator == Exists {
					return fmt.Errorf("dataType: %s cannot support operation: %s", kind.String(), cond.operator.string())
				}
				f.clauses[i][j].data = int8(parsedInt)
			case reflect.Int16:
				parsedInt, err := strconv.ParseInt(dataString, 10, 16)
				if err != nil {
					return err
				}
				if cond.operator == Contains || cond.operator == Exists {
					return fmt.Errorf("dataType: %s cannot support operation: %s", kind.String(), cond.operator.string())
				}
				f.clauses[i][j].data = int16(parsedInt)
			case reflect.Int32:
				parsedInt, err := strconv.ParseInt(dataString, 10, 32)
				if err != nil {
					return err
				}
				if cond.operator == Contains || cond.operator == Exists {
					return fmt.Errorf("dataType: %s cannot support operation: %s", kind.String(), cond.operator.string())
				}
				f.clauses[i][j].data = int32(parsedInt)
			case reflect.Int64:
				parsedInt, err := strconv.ParseInt(dataString, 10, 64)
				if err != nil {
					return err
				}
				if cond.operator == Contains || cond.operator == Exists {
					return fmt.Errorf("dataType: %s cannot support operation: %s", kind.String(), cond.operator.string())
				}
				f.clauses[i][j].data = int64(parsedInt)
			case reflect.Uint:
				parsedUint, err := strconv.ParseUint(dataString, 10, 64)
				if err != nil {
					return err
				}
				if cond.operator == Contains || cond.operator == Exists {
					return fmt.Errorf("dataType: %s cannot support operation: %s", kind.String(), cond.operator.string())
				}
				f.clauses[i][j].data = uint(parsedUint)
			case reflect.Uint8:
				parsedUint, err := strconv.ParseUint(dataString, 10, 8)
				if err != nil {
					return err
				}
				if cond.operator == Contains || cond.operator == Exists {
					return fmt.Errorf("dataType: %s cannot support operation: %s", kind.String(), cond.operator.string())
				}
				f.clauses[i][j].data = uint8(parsedUint)
			case reflect.Uint16:
				parsedUint, err := strconv.ParseUint(dataString, 10, 16)
				if err != nil {
					return err
				}
				if cond.operator == Contains || cond.operator == Exists {
					return fmt.Errorf("dataType: %s cannot support operation: %s", kind.String(), cond.operator.string())
				}
				f.clauses[i][j].data = uint16(parsedUint)
			case reflect.Uint32:
				parsedUint, err := strconv.ParseUint(dataString, 10, 32)
				if err != nil {
					return err
				}
				if cond.operator == Contains || cond.operator == Exists {
					return fmt.Errorf("dataType: %s cannot support operation: %s", kind.String(), cond.operator.string())
				}
				f.clauses[i][j].data = uint32(parsedUint)
			case reflect.Uint64:
				if cond.operator == Contains || cond.operator == Exists {
					return fmt.Errorf("dataType: %s cannot support operation: %s", kind.String(), cond.operator.string())
				}

				if strings.Contains(cond.field, "timestamp") {

					if parsedTime, err := time.Parse("2006-01-02T15:04:05.000", dataString); err == nil {
						f.clauses[i][j].data = uint64(parsedTime.UnixNano())
					} else if parsedTime, err = time.Parse("2006-01-02", dataString); err == nil {
						if cond.operator == Eq || cond.operator == Ne || cond.operator == Lt || cond.operator == Lte {
							f.clauses[i][j].data = uint64(parsedTime.UnixNano())
						} else if cond.operator == Gt || cond.operator == Gte {
							parsedTime = parsedTime.Add(24 * time.Hour).Add(-1 * time.Millisecond)
							f.clauses[i][j].data = uint64(parsedTime.UnixNano())
						} else {
							return errors.New("timestamp don't support operator: " + cond.operator.string())
						}
					} else {
						return errors.New("illegal time format: " + dataString)
					}

				} else {
					parsedUint, err := strconv.ParseUint(dataString, 10, 64)
					if err != nil {
						return err
					}
					f.clauses[i][j].data = uint64(parsedUint)
				}

			case reflect.Float32:
				parsedFloat, err := strconv.ParseFloat(dataString, 32)
				if err != nil {
					return err
				}
				if cond.operator == Contains || cond.operator == Exists {
					return fmt.Errorf("dataType: %s cannot support operation: %s", kind.String(), cond.operator.string())
				}
				f.clauses[i][j].data = float32(parsedFloat)
			case reflect.Float64:
				parsedFloat, err := strconv.ParseFloat(dataString, 64)
				if err != nil {
					return err
				}
				if cond.operator == Contains || cond.operator == Exists {
					return fmt.Errorf("dataType: %s cannot support operation: %s", kind.String(), cond.operator.string())
				}
				f.clauses[i][j].data = parsedFloat
			case reflect.String:
				if cond.operator != Eq && cond.operator != Ne && cond.operator != Contains {
					return fmt.Errorf("dataType: %s cannot support operation: %s", kind.String(), cond.operator.string())
				}
			case reflect.Slice:
				if cond.operator != Eq && cond.operator != Ne && cond.operator != Contains {
					return fmt.Errorf("dataType: %s cannot support operation: %s", kind.String(), cond.operator.string())
				}
				f.clauses[i][j].data = []byte(dataString)
			default:
				return errors.New("unexpected type: " + kind.String())
			}

			f.clauses[i][j].dataType = kind
		}
	}
	return nil
}
