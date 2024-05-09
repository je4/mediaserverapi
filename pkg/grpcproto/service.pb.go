// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v4.25.3
// source: service.proto

package grpcproto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

var File_service_proto protoreflect.FileDescriptor

var file_service_proto_rawDesc = []byte{
	0x0a, 0x0d, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x09, 0x67, 0x72, 0x70, 0x63, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x0c, 0x69, 0x6e, 0x67, 0x65,
	0x73, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x15, 0x64, 0x65, 0x66, 0x61, 0x75, 0x6c,
	0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x32,
	0x4e, 0x0a, 0x0a, 0x41, 0x50, 0x49, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x40, 0x0a,
	0x06, 0x49, 0x6e, 0x67, 0x65, 0x73, 0x74, 0x12, 0x18, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2e, 0x49, 0x6e, 0x67, 0x65, 0x73, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x1a, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x44, 0x65,
	0x66, 0x61, 0x75, 0x6c, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42,
	0x80, 0x01, 0x0a, 0x1c, 0x63, 0x68, 0x2e, 0x75, 0x6e, 0x69, 0x62, 0x61, 0x73, 0x2e, 0x75, 0x62,
	0x2e, 0x6d, 0x65, 0x64, 0x69, 0x61, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x2e, 0x61, 0x70, 0x69,
	0x42, 0x0c, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01,
	0x5a, 0x2e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6a, 0x65, 0x34,
	0x2f, 0x6d, 0x65, 0x64, 0x69, 0x61, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x61, 0x70, 0x69, 0x2f,
	0x76, 0x32, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0xa2, 0x02, 0x03, 0x55, 0x42, 0x42, 0xaa, 0x02, 0x19, 0x55, 0x6e, 0x69, 0x62, 0x61, 0x73, 0x2e,
	0x55, 0x42, 0x2e, 0x4d, 0x65, 0x64, 0x69, 0x61, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x2e, 0x41,
	0x50, 0x49, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var file_service_proto_goTypes = []interface{}{
	(*IngestRequest)(nil),   // 0: grpcproto.IngestRequest
	(*DefaultResponse)(nil), // 1: grpcproto.DefaultResponse
}
var file_service_proto_depIdxs = []int32{
	0, // 0: grpcproto.APIService.Ingest:input_type -> grpcproto.IngestRequest
	1, // 1: grpcproto.APIService.Ingest:output_type -> grpcproto.DefaultResponse
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_service_proto_init() }
func file_service_proto_init() {
	if File_service_proto != nil {
		return
	}
	file_ingest_proto_init()
	file_defaultResponse_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_service_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_service_proto_goTypes,
		DependencyIndexes: file_service_proto_depIdxs,
	}.Build()
	File_service_proto = out.File
	file_service_proto_rawDesc = nil
	file_service_proto_goTypes = nil
	file_service_proto_depIdxs = nil
}