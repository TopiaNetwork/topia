package public

import "time"

func MyTest(i int, s string, f float64) (res string, e error) {
	// time.Sleep(5 * time.Second)
	return s + " is received,now return", nil
}

func MyTestWithSleep(i int, s string, f float64) (res string, e error) {
	time.Sleep(5 * time.Second)
	return s + " is received,and sleep for 5 second,now return", nil
}
