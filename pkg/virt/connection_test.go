package virt

import (
	"reflect"
	"testing"
)

func TestBuildURIs(t *testing.T) {
	tests := []struct {
		name      string
		ips       []string
		template  string
		wantURIs  []string
		wantAddrs []string
		wantErr   bool
	}{
		{
			name:      "默认 TCP 模板展开多个地址",
			ips:       []string{"10.0.0.1", "10.0.0.2"},
			wantURIs:  []string{"qemu+tcp://10.0.0.1/system", "qemu+tcp://10.0.0.2/system"},
			wantAddrs: []string{"10.0.0.1", "10.0.0.2"},
		},
		{
			name:      "自定义模板带端口",
			ips:       []string{"10.0.0.1"},
			template:  "qemu+tcp://{host}:16509/system",
			wantURIs:  []string{"qemu+tcp://10.0.0.1:16509/system"},
			wantAddrs: []string{"10.0.0.1"},
		},
		{
			name:      "SSH 模板作为可选覆盖",
			ips:       []string{"10.0.0.1"},
			template:  "qemu+ssh://root@{host}/system?no_verify=1",
			wantURIs:  []string{"qemu+ssh://root@10.0.0.1/system?no_verify=1"},
			wantAddrs: []string{"10.0.0.1"},
		},
		{
			name:      "忽略空白地址",
			ips:       []string{"10.0.0.1", "  ", ""},
			template:  "qemu+tcp://{host}/system",
			wantURIs:  []string{"qemu+tcp://10.0.0.1/system"},
			wantAddrs: []string{"10.0.0.1"},
		},
		{
			name:     "模板缺少地址",
			template: "qemu+tcp://{host}/system",
			wantErr:  true,
		},
		{
			name:      "固定 URI 忽略地址",
			ips:       []string{"10.0.0.1"},
			template:  "qemu:///system",
			wantURIs:  []string{"qemu:///system"},
			wantAddrs: []string{""},
		},
		{
			name:      "固定 URI 本地连接",
			template:  "qemu:///system",
			wantURIs:  []string{"qemu:///system"},
			wantAddrs: []string{""},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			uris, addrs, err := buildURIs(test.ips, test.template)
			if test.wantErr {
				if err == nil {
					t.Fatal("期望返回错误，实际没有")
				}
				return
			}
			if err != nil {
				t.Fatalf("buildURIs 返回错误: %v", err)
			}
			if !reflect.DeepEqual(uris, test.wantURIs) {
				t.Errorf("uris = %v，期望 %v", uris, test.wantURIs)
			}
			if !reflect.DeepEqual(addrs, test.wantAddrs) {
				t.Errorf("addrs = %v，期望 %v", addrs, test.wantAddrs)
			}
		})
	}
}
