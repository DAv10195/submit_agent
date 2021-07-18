package agent

import (
	"bytes"
	"context"
	"fmt"
	commons "github.com/DAv10195/submit_commons"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

const javaCode = `public class HelloWorld{
     public static void main(String []args){
        System.out.println("Hello World");
     }
}
`

func TestMossOutputRule(t *testing.T) {
	mossParserContext, mossParserCancel := context.WithCancel(context.Background())
	mossParserCommand := exec.CommandContext(mossParserContext, "sh", "-c", fmt.Sprintf("java -jar mossparser.jar"))
	mossParserCommand.Dir = os.Getenv("MOSSPARSER_HOME")
	defer func() {
		mossParserCancel()
	}()
	if err := mossParserCommand.Start(); err != nil {
		t.Fatalf("error starting command for test: %v", err)
	}
	tmpDir := os.TempDir()
	dir1, dir2 := filepath.Join(tmpDir, "mossdir1"), filepath.Join(tmpDir, "mossdir2")
	if err := os.MkdirAll(dir1, 0700); err != nil {
		t.Fatalf("error creating dir for test: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(dir1); err != nil {
			panic(err)
		}
	}()
	if err := os.MkdirAll(dir2, 0700); err != nil {
		t.Fatalf("error creating dir for test: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(dir1); err != nil {
			panic(err)
		}
	}()
	f, err := os.Create(filepath.Join(dir1, "main.java"))
	if err != nil {
		t.Fatalf("error creating file for test: %v", err)
	}
	if _, err := f.WriteString(javaCode); err != nil {
		t.Fatalf("error writing file for test: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("error closing file for test: %v", err)
	}
	f, err = os.Create(filepath.Join(dir2, "main.java"))
	if err != nil {
		t.Fatalf("error creating file for test: %v", err)
	}
	if _, err := f.WriteString(javaCode); err != nil {
		t.Fatalf("error writing file for test: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("error closing file for test: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", "moss -l java -d mossdir1/*.java mossdir2/*.java")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	cmd.Dir = tmpDir
	cmd.Env = []string{fmt.Sprintf("PATH=%s", os.Getenv("MOSS_HOME"))}
	timer := time.AfterFunc(3 * time.Minute, func() {
		cancel()
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
			t.Fatalf("error killing timedout command for test: %v", err)
		}
	})
	defer timer.Stop()
	if err := cmd.Start(); err != nil {
		t.Fatalf("error starting command for test: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("error waiting for command for test: %v", err)
	}
	timer.Stop()
	labels := make(map[string]interface{})
	labels[commons.MossLink] = ""
	a := &Agent{config: &Config{MossParserHost: "localhost", MossParserPort: 4567}}
	if _, err := a.mossOutputRule(output.String(), labels); err != nil {
		t.Fatalf("error applying moss output rule: %v", err)
	}
	if labels[commons.MossLink].(string) == "" {
		t.Fatalf("moss rule didn't set the %s label", commons.MossLink)
	}
}
