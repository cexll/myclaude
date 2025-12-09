package main

import (
	"testing"
)

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
