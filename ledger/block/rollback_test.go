package block

import (
	"fmt"
	"github.com/TopiaNetwork/topia/chain/types"
	"os"
	"strconv"

	//"os"
	"testing"
)


var blocknum uint64 = 123456;
var blocknum_array = []uint64{123456,123457,123458}

//func init(t *testing.T) {
//
//}

func TestNewRollback(t *testing.T) {
	rollback,_ := NewRollback(types.BlockNum(blocknum))
	fmt.Println("",rollback)
	datafile := newtestfile(strconv.FormatUint(blocknum,10),1)
	newtestfile(strconv.FormatUint(blocknum,10),1)


	file, err := os.OpenFile(datafile.File.Name(), os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)

	if err != nil{
		panic(err)
	}

	var to = FileItem{
		1,
		file,
		FILE_HEADER_SIZE,
		0,
		New(),
	}
	for i:=0; i < 1000;i++ {
		err = to.Writedata(&block_all)
		blockhead1.Height = blockhead1.Height + 1
		block_all = types.Block{
			&blockhead1,
			&blockdata,
			struct{}{},
			nil,
			100,
		}
	}

}


func TestFileItem_AddRollback(t *testing.T) {
	newtestfile(strconv.FormatUint(blocknum,10),2)

	rollback,_ := NewRollback(types.BlockNum(blocknum))

	rollback.AddRollback(types.BlockNum(blocknum))
}

func TestRemoveBlockhead(t *testing.T) {
	datafile := newtestfile(strconv.FormatUint(blocknum,10),0)

	// bug need to fix
	err := RemoveBlockhead(datafile,blocknum)
	if err != nil {
		panic(err)
	}


}

func TestRemoveBlockdata(t *testing.T) {
	datafile := newtestfile(strconv.FormatUint(blocknum,10),0)

	err := RemoveBlockdata(datafile,datafile.Offset)
	if err != nil {
		panic(err)
	}

}

func TestRemoveindex(t *testing.T) {

	datafile := newtestfile(strconv.FormatUint(blocknum,10),1)

	err := Removeindex(datafile, blocknum_array)
	panic(err)
}


//func TestFileItem_AddRollback(t *testing.T) {
//	type fields struct {
//		Filetype     FileType
//		File         *os.File
//		Offset       uint64
//		HeaderOffset uint64
//		Bloom        *BloomFilter
//	}
//	type args struct {
//		blocknum types.BlockNum
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//
//
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			Rfile := &FileItem{
//				Filetype:     tt.fields.Filetype,
//				File:         tt.fields.File,
//				Offset:       tt.fields.Offset,
//				HeaderOffset: tt.fields.HeaderOffset,
//				Bloom:        tt.fields.Bloom,
//			}
//			if err := Rfile.AddRollback(tt.args.blocknum); (err != nil) != tt.wantErr {
//				t.Errorf("AddRollback() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}


//func TestFileItem_EmptyRollback(t *testing.T) {
//	type fields struct {
//		Filetype     FileType
//		File         *os.File
//		Offset       uint64
//		HeaderOffset uint64
//		Bloom        *BloomFilter
//	}
//	type args struct {
//		blocknum types.BlockNum
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			RollFile := &FileItem{
//				Filetype:     tt.fields.Filetype,
//				File:         tt.fields.File,
//				Offset:       tt.fields.Offset,
//				HeaderOffset: tt.fields.HeaderOffset,
//				Bloom:        tt.fields.Bloom,
//			}
//			if err := RollFile.EmptyRollback(tt.args.blocknum); (err != nil) != tt.wantErr {
//				t.Errorf("EmptyRollback() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}


//func TestNewRollback1(t *testing.T) {
//	type args struct {
//		blocknum types.BlockNum
//	}
//	tests := []struct {
//		name    string
//		args    args
//		want    *FileItem
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got, err := NewRollback(tt.args.blocknum)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("NewRollback() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("NewRollback() got = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}


//func TestRemoveBlockdata1(t *testing.T) {
//	type args struct {
//		indexfile *FileItem
//		offset    uint64
//	}
//	tests := []struct {
//		name    string
//		args    args
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if err := RemoveBlockdata(tt.args.indexfile, tt.args.offset); (err != nil) != tt.wantErr {
//				t.Errorf("RemoveBlockdata() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}


//func TestRemoveBlockhead1(t *testing.T) {
//	type args struct {
//		datafile *FileItem
//		offset   uint64
//	}
//	tests := []struct {
//		name    string
//		args    args
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if err := RemoveBlockhead(tt.args.datafile, tt.args.offset); (err != nil) != tt.wantErr {
//				t.Errorf("RemoveBlockhead() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}
//
//
//func TestRemoveindex1(t *testing.T) {
//	type args struct {
//		Indexfile *FileItem
//		blocknums []types.BlockNum
//	}
//	tests := []struct {
//		name    string
//		args    args
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if err := Removeindex(tt.args.Indexfile, tt.args.blocknums); (err != nil) != tt.wantErr {
//				t.Errorf("Removeindex() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}