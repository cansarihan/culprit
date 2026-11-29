package patch

import "testing"

const sample = `diff --git a/parse.go b/parse.go
index 111..222 100644
--- a/parse.go
+++ b/parse.go
@@ -10,0 +11,3 @@ func read(b []byte) {
+	if n > len(b) {
+		return errShort
+	}
@@ -20 +23 @@
-	x := b[n]
+	x := b[n-1]
diff --git a/old.go b/old.go
deleted file mode 100644
--- a/old.go
+++ /dev/null
@@ -1,2 +0,0 @@
-package old
-func Gone() {}
`

func TestParse(t *testing.T) {
	files := Parse([]byte(sample))
	if len(files) != 2 {
		t.Fatalf("files = %d, want 2", len(files))
	}
	p := files[0]
	if p.Path != "parse.go" || len(p.Added) != 4 || len(p.Removed) != 1 {
		t.Errorf("parse.go = %+v", p)
	}
	if p.Added[0] != "\tif n > len(b) {" {
		t.Errorf("first added line = %q", p.Added[0])
	}
	if files[1].Path != "old.go" || len(files[1].Removed) != 2 || files[1].Churn() != 2 {
		t.Errorf("deleted file = %+v", files[1])
	}
}

func TestParseMarksNewFiles(t *testing.T) {
	files := Parse([]byte("diff --git a/n.go b/n.go\nnew file mode 100644\n--- /dev/null\n+++ b/n.go\n@@ -0,0 +1 @@\n+package n\n"))
	if len(files) != 1 || !files[0].New || files[0].Path != "n.go" {
		t.Fatalf("new file not marked: %+v", files)
	}
	if Parse([]byte(sample))[0].New {
		t.Error("a modified file is not new")
	}
}
