package beeorm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEngine(t *testing.T) {
	engine, def := prepareTables(t, &Registry{}, 5, "", "2.0")
	defer def()
	source := engine.GetRegistry().GetSourceRegistry()
	assert.NotNil(t, source)
	assert.PanicsWithError(t, "unregistered mysql pool 'test'", func() {
		engine.GetMysql("test")
	})
	assert.PanicsWithError(t, "unregistered local cache pool 'test'", func() {
		engine.GetLocalCache("test")
	})
	assert.PanicsWithError(t, "unregistered redis cache pool 'test'", func() {
		engine.GetRedis("test")
	})

	engine.EnableQueryDebug()
	assert.Len(t, engine.queryLoggersRedis, 1)
	assert.Len(t, engine.queryLoggersDB, 1)
	assert.Len(t, engine.queryLoggersLocalCache, 1)
	engine.EnableQueryDebugCustom(true, true, false)
	assert.Len(t, engine.queryLoggersRedis, 1)
	assert.Len(t, engine.queryLoggersDB, 1)
	assert.Len(t, engine.queryLoggersLocalCache, 1)
}

func BenchmarkEngine(b *testing.B) {
	registry := &Registry{}
	validatedRegistry, def, _ := registry.Validate()
	defer def()
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		validatedRegistry.CreateEngine()
	}
}
