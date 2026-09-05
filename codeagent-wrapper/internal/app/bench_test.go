package wrapper

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	config "codeagent-wrapper/internal/config"
)

var (
	benchCmdSink      any
	benchConfigSink   *Config
	benchMessageSink  string
	benchThreadIDSink string
)

// BenchmarkStartup_NewRootCommand measures CLI startup overhead (command+flags construction).
func BenchmarkStartup_NewRootCommand(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchCmdSink = newRootCommand()
	}
}

// BenchmarkConfigParse_ParseArgs measures config parsing from argv/env (steady-state).
func BenchmarkConfigParse_ParseArgs(b *testing.B) {
	home := b.TempDir()
	b.Setenv("HOME", home)
	b.Setenv("USERPROFILE", home)

	configDir := filepath.Join(home, ".codeagent")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "models.json"), []byte(`{
  "agents": {
    "develop": { "backend": "codex", "model": "gpt-test" }
  }
}`), 0o644); err != nil {
		b.Fatal(err)
	}

	config.ResetModelsConfigCacheForTest()
	b.Cleanup(config.ResetModelsConfigCacheForTest)

	origArgs := os.Args
	os.Args = []string{"codeagent-wrapper", "--agent", "develop", "task"}
	b.Cleanup(func() { os.Args = origArgs })

	if _, err := parseArgs(); err != nil {
		b.Fatalf("warmup parseArgs() error: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg, err := parseArgs()
		if err != nil {
			b.Fatalf("parseArgs() error: %v", err)
		}
		benchConfigSink = cfg
	}
}

// BenchmarkJSONParse_ParseJSONStreamInternal measures line-delimited JSON stream parsing.
func BenchmarkJSONParse_ParseJSONStreamInternal(b *testing.B) {
	stream := []byte(
		`{"type":"thread.started","thread_id":"t"}` + "\n" +
			`{"type":"item.completed","item":{"type":"agent_message","text":"hello"}}` + "\n" +
			`{"type":"thread.completed","thread_id":"t"}` + "\n",
	)
	b.SetBytes(int64(len(stream)))

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		message, threadID := parseJSONStreamInternal(bytes.NewReader(stream), nil, nil, nil, nil)
		benchMessageSink = message
		benchThreadIDSink = threadID
	}
}

// BenchmarkLoggerWrite 测试日志写入性能
func BenchmarkLoggerWrite(b *testing.B) {
	logger, err := NewLogger()
	if err != nil {
		b.Fatal(err)
	}
	defer logger.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark log message")
	}
	b.StopTimer()
	logger.Flush()
}

// BenchmarkLoggerConcurrentWrite 测试并发日志写入性能
func BenchmarkLoggerConcurrentWrite(b *testing.B) {
	logger, err := NewLogger()
	if err != nil {
		b.Fatal(err)
	}
	defer logger.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("concurrent benchmark log message")
		}
	})
	b.StopTimer()
	logger.Flush()
}
