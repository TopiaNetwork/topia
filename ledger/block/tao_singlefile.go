package block

import (
	"io/ioutil"
)

func main() {

	content := []byte("transaction1\ntransaction2\n")
	err := ioutil.WriteFile("test.txt", content, 0644)
	if err != nil {
		panic(err)
	}
}
