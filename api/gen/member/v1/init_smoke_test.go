package memberv1

import "testing"

// TestFileDescriptorInit 强制执行生成代码的 init（解析 rawDesc）。
// go build / go test 在无 import 侧效应时不会覆盖「rawDesc 损坏 → 进程启动即 panic」这类故障。
func TestFileDescriptorInit(t *testing.T) {
	// 步骤 1：触达包级 FileDescriptor，确保 filedesc.Builder.Build 已成功跑完
	if File_member_v1_auth_proto == nil {
		t.Fatal("File_member_v1_auth_proto is nil after init; regenerate with ./scripts/gen-proto.sh")
	}
	// 步骤 2：校验基础元数据，避免空壳 descriptor 蒙混过关
	if File_member_v1_auth_proto.Path() != "member/v1/auth.proto" {
		t.Fatalf("unexpected proto path: %q", File_member_v1_auth_proto.Path())
	}
	if File_member_v1_auth_proto.Messages().ByName("IntrospectResponse") == nil {
		t.Fatal("IntrospectResponse message missing from descriptor")
	}
}
