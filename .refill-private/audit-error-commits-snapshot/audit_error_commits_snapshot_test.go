package auditerrorcommitssnapshot_test

import (
	"os"
	"ovencheck/internal/core"
	"path/filepath"
	"testing"
)

func TestCreateAuditFailureDoesNotCommitSnapshot(t *testing.T) {
	snapshotPath := filepath.Join(t.TempDir(), "snapshot.json")
	if err := os.Mkdir(snapshotPath+".audit.jsonl", 0755); err != nil {
		t.Fatal(err)
	}

	store, err := core.NewStore(snapshotPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Create("一号烘炉", "K-01", "高铝砖", "操作员", "工程师"); err == nil {
		t.Fatal("预期审计资源失效时创建命令返回错误")
	}

	restarted, err := core.NewStore(snapshotPath)
	if err != nil {
		t.Fatal(err)
	}
	if batches := restarted.List(); len(batches) != 0 {
		t.Fatalf("TestCreateAuditFailureDoesNotCommitSnapshot: Create 返回错误后重启仍加载到 %d 个批次", len(batches))
	}
}
