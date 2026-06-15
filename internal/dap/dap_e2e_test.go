package dap

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func debugpyAvailable() bool {
	if _, err := exec.LookPath("python"); err != nil {
		return false
	}
	return exec.Command("python", "-c", "import debugpy").Run() == nil
}

func TestE2EDebugPy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	if !debugpyAvailable() {
		t.Skip("debugpy not installed (pip install debugpy)")
	}

	dir := t.TempDir()
	script := filepath.Join(dir, "target.py")
	if err := os.WriteFile(script, []byte(`import sys

def compute(x):
    total = 0           # line 3
    for i in range(x):
        total += i
    return total

if __name__ == "__main__":
    x = int(sys.argv[1]) if len(sys.argv) > 1 else 5
    result = compute(x)
    print("result:", result)
`), 0644); err != nil {
		t.Fatal(err)
	}

	mgr := NewDapManager(dir)
	summary, err := mgr.Launch(script, []string{"3"}, "debugpy", dir)
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	t.Logf("launched: session=%s status=%s", summary.ID, summary.Status)

	session := mgr.GetSession(summary.ID)
	if session == nil {
		t.Fatal("session not found")
	}

	// Set breakpoint (inside compute, line 3)
	bps, err := session.SetBreakpoint(script, 3, "", 10*time.Second)
	if err != nil {
		t.Fatalf("SetBreakpoint: %v", err)
	}
	if len(bps) == 0 || !bps[0].Verified {
		t.Fatalf("breakpoint not verified: %+v", bps)
	}

	// Complete the launch (send configurationDone)
	if err := session.CompleteLaunch(10 * time.Second); err != nil {
		t.Fatalf("CompleteLaunch: %v", err)
	}
	t.Log("CompleteLaunch OK — program should be running")

	// Wait a moment for the program to hit the breakpoint
	time.Sleep(1 * time.Second)

	// Check if we got stopped
	summary2 := session.Summary()
	t.Logf("status after wait: %s, stop reason: %s", summary2.Status, summary2.StopLocation.Reason)

	// Try stack trace
	frames, err := session.StackTrace(5, 10*time.Second)
	if err != nil {
		t.Logf("StackTrace: %v", err)
	} else {
		for _, f := range frames {
			t.Logf("  frame %d: %s line=%d", f.Id, f.Name, f.Line)
		}
	}

	// Continue
	outcome, err := session.Continue(15 * time.Second)
	if err != nil {
		t.Fatalf("Continue: %v", err)
	}
	t.Logf("continue: state=%s", outcome.State)

	output := session.GetOutput()
	t.Logf("output: %s", output)

	session.Terminate(5 * time.Second)
	t.Logf("final: %v", fmt.Sprintf("%v", session.Summary().Status))
}
