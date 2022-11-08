package utils

import (
	"github.com/champly/lib4go/encoding"
	"google.golang.org/protobuf/types/known/structpb"
	//"github.com/gogo/protobuf/types"
)

// func ConvertJSON2Struct(str string) *structpb.Struct {
// 	res, _ := encoding.JSON2Struct(str)
// 	return res
// }

func ConvertYaml2Struct(str string) *structpb.Struct {
	res, _ := encoding.YAML2Struct(str)
	return res
}
