package beeorm

import (
	"context"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type redisSearchEntity struct {
	ORM               `orm:"redisSearch=search"`
	ID                uint               `orm:"searchable;sortable"`
	Age               uint64             `orm:"searchable;sortable"`
	Balance           int64              `orm:"sortable"`
	Weight            float64            `orm:"searchable"`
	AgeNullable       *uint64            `orm:"searchable;sortable"`
	BalanceNullable   *int64             `orm:"searchable;sortable"`
	Enum              string             `orm:"enum=beeorm.TestEnum;required;searchable"`
	EnumNullable      string             `orm:"enum=beeorm.TestEnum;searchable"`
	Name              string             `orm:"searchable"`
	NameStem          string             `orm:"searchable;stem"`
	Set               []string           `orm:"set=beeorm.TestEnum;required;searchable"`
	SetNullable       []string           `orm:"set=beeorm.TestEnum;searchable"`
	Bool              bool               `orm:"searchable;sortable"`
	BoolNullable      *bool              `orm:"searchable"`
	WeightNullable    *float64           `orm:"searchable"`
	Date              time.Time          `orm:"searchable"`
	DateTime          time.Time          `orm:"time;searchable"`
	DateNullable      *time.Time         `orm:"searchable"`
	DateTimeNullable  *time.Time         `orm:"time;searchable"`
	Ref               *redisSearchEntity `orm:"searchable"`
	Another           string
	AnotherNumeric    int64
	AnotherTag        bool
	Balance32         int32                  `orm:"sortable"`
	AgeNullable32     *uint32                `orm:"searchable;sortable"`
	BalanceNullable32 *int32                 `orm:"sortable"`
	ReferenceMany     []*redisNoSearchEntity `orm:"searchable"`
	FakeDelete        bool                   `orm:"searchable"`
}

type redisSearchEntityNoSearchableFakeDelete struct {
	ORM        `orm:"redisSearch=search"`
	ID         uint   `orm:"searchable;sortable"`
	Age        uint64 `orm:"searchable;sortable"`
	FakeDelete bool
}

type redisNoSearchEntity struct {
	ORM
	ID uint
}

type redisSearchOnlySortPKEntity struct {
	ORM `orm:"redisSearch=search"`
	ID  uint `orm:"sortable"`
}

type redisSearchFakeDeleteEntity struct {
	ORM        `orm:"redisSearch=search"`
	ID         uint `orm:"sortable"`
	FakeDelete bool `orm:"searchable"`
}

type redisSearchAggregateEntity struct {
	ORM        `orm:"redisSearch=search"`
	ID         uint   `orm:"sortable;searchable"`
	Age        int    `orm:"searchable"`
	Size       int    `orm:"searchable"`
	Name       string `orm:"sortable"`
	FakeDelete bool   `orm:"searchable"`
}

func TestEntityRedisSearchIndexer(t *testing.T) {
	testEntityRedisSearchIndexer(t, "", "2.0")
}

func TestEntityRedisSearchIndexer22(t *testing.T) {
	testEntityRedisSearchIndexer(t, "", "2.2")
}

func TestEntityRedisSearchIndexerNamespace(t *testing.T) {
	testEntityRedisSearchIndexer(t, "test", "2.0")
}

func testEntityRedisSearchIndexer(t *testing.T, redisNamespace, version string) {
	var entity *redisSearchEntity
	registry := &Registry{}
	registry.RegisterEnumStruct("beeorm.TestEnum", TestEnum)
	engine, def := prepareTables(t, registry, 5, redisNamespace, version, entity, &redisNoSearchEntity{})
	defer def()
	indexer := NewBackgroundConsumer(engine)
	indexer.DisableLoop()
	indexer.blockTime = time.Millisecond
	flusher := engine.NewFlusher()
	for i := 1; i <= entityIndexerPage+100; i++ {
		e := &redisSearchEntity{Age: uint64(i)}
		flusher.Track(e)
	}
	flusher.Flush()
	engine.GetRedis().FlushDB()
	engine.GetRedis("search").FlushDB()
	schema := engine.GetRegistry().GetTableSchemaForEntity(entity)
	schema.ReindexRedisSearchIndex(engine)

	assert.True(t, indexer.Digest(context.Background()))
	query := NewRedisSearchQuery()
	total := engine.RedisSearchCount(entity, query)
	assert.Equal(t, uint64(entityIndexerPage+100), total)

	e := &redisSearchEntity{ID: 10}
	engine.Load(e)
	engine.Delete(e)
	engine.GetRedis().FlushDB()
	engine.GetRedis("search").FlushDB()
	schema.ReindexRedisSearchIndex(engine)
	assert.True(t, indexer.Digest(context.Background()))
	query = NewRedisSearchQuery()
	total = engine.RedisSearchCount(entity, query)
	assert.Equal(t, uint64(entityIndexerPage+99), total)
	query = NewRedisSearchQuery()
	query.WithFakeDeleteRows()
	total = engine.RedisSearchCount(entity, query)
	assert.Equal(t, uint64(entityIndexerPage+100), total)
}

func TestEntityRedisSearchIndexerNoFakeDelete(t *testing.T) {
	var entity *redisSearchEntityNoSearchableFakeDelete
	registry := &Registry{}
	registry.RegisterEnumStruct("beeorm.TestEnum", TestEnum)
	engine, def := prepareTables(t, registry, 5, "", "2.0", entity)
	defer def()
	indexer := NewBackgroundConsumer(engine)
	indexer.DisableLoop()
	indexer.blockTime = time.Millisecond
	flusher := engine.NewFlusher()
	for i := 1; i <= entityIndexerPage+100; i++ {
		e := &redisSearchEntityNoSearchableFakeDelete{Age: uint64(i)}
		flusher.Track(e)
	}
	flusher.Flush()
	engine.GetRedis().FlushDB()
	engine.GetRedis("search").FlushDB()
	schema := engine.GetRegistry().GetTableSchemaForEntity(entity)
	schema.ReindexRedisSearchIndex(engine)

	assert.True(t, indexer.Digest(context.Background()))
	query := NewRedisSearchQuery()
	total := engine.RedisSearchCount(entity, query)
	assert.Equal(t, uint64(entityIndexerPage+100), total)

	e := &redisSearchEntityNoSearchableFakeDelete{ID: 10}
	engine.Load(e)
	engine.Delete(e)
	engine.GetRedis().FlushDB()
	engine.GetRedis("search").FlushDB()
	schema.ReindexRedisSearchIndex(engine)
	assert.True(t, indexer.Digest(context.Background()))
	query = NewRedisSearchQuery()
	total = engine.RedisSearchCount(entity, query)
	assert.Equal(t, uint64(entityIndexerPage+99), total)
	query = NewRedisSearchQuery()
	query.WithFakeDeleteRows()
	total = engine.RedisSearchCount(entity, query)
	assert.Equal(t, uint64(entityIndexerPage+99), total)
}

func TestEntityRedisSearch(t *testing.T) {
	testEntityRedisSearch(t, "")
}

func TestEntityRedisSearchNamespace(t *testing.T) {
	testEntityRedisSearch(t, "test")
}

func testEntityRedisSearch(t *testing.T, redisNamespace string) {
	var entity *redisSearchEntity
	registry := &Registry{}
	registry.RegisterEnumStruct("beeorm.TestEnum", TestEnum)
	engine, def := prepareTables(t, registry, 5, redisNamespace, "2.0", entity, &redisNoSearchEntity{}, &redisNoSearchEntity{})

	alters := engine.GetRedisSearchIndexAlters()
	assert.Len(t, alters, 0)

	assert.Len(t, engine.GetRedisSearch().ListIndices(), 1)

	indexer := NewBackgroundConsumer(engine)
	indexer.DisableLoop()
	indexer.blockTime = time.Millisecond

	flusher := engine.NewFlusher()
	now := time.Now()

	list := make([]*redisSearchEntity, 0)
	for i := 1; i <= 50; i++ {
		e := &redisSearchEntity{Age: uint64(i)}
		list = append(list, e)
		e.Weight = 100.3 + float64(i)
		e.Balance = 20 - int64(i)
		e.Enum = TestEnum.A
		e.Set = []string{"a"}
		e.Name = "dog " + strconv.Itoa(i)
		e.NameStem = "carrot " + strconv.Itoa(i)
		if i > 20 {
			v := uint64(i)
			e.AgeNullable = &v
			v2 := int64(i)
			e.BalanceNullable = &v2
			e.Enum = TestEnum.B
			e.Set = []string{"a", "b"}
			e.SetNullable = []string{"a", "b"}
			e.EnumNullable = TestEnum.B
			e.Name = "Cat " + strconv.Itoa(i)
			e.NameStem = "Orange " + strconv.Itoa(i)
			b := false
			e.BoolNullable = &b
			f := 10.2
			e.WeightNullable = &f
			e.Date = now
			e.DateTime = now
			e.DateNullable = &now
			e.DateTimeNullable = &now
			e.ReferenceMany = []*redisNoSearchEntity{{ID: 1}}
		}
		if i > 40 {
			e.Enum = TestEnum.C
			e.EnumNullable = TestEnum.C
			e.Set = []string{"a", "b", "c"}
			e.SetNullable = []string{"a", "b", "c"}
			e.Name = "cats " + strconv.Itoa(i)
			e.NameStem = "oranges " + strconv.Itoa(i)
			e.Bool = true
			b := true
			e.BoolNullable = &b
			f := 20.2
			e.WeightNullable = &f
			e.Date = now.Add(time.Hour * 48)
			e.DateTime = e.Date
			e.ReferenceMany = []*redisNoSearchEntity{{ID: 1}, {ID: 2}}
		}
		if i > 45 {
			e.ReferenceMany = []*redisNoSearchEntity{{ID: 1}, {ID: 3}}
		}
		flusher.Track(e)
	}
	flusher.Flush()
	list[0].Ref = list[30]
	list[1].Ref = list[30]
	list[2].Ref = list[31]
	list[3].Ref = list[31]
	list[4].Ref = list[31]
	flusher.Flush()

	indices := engine.GetRedisSearch("search").ListIndices()
	assert.Len(t, indices, 1)
	assert.Equal(t, "beeorm.redisSearchEntity", indices[0])
	info := engine.GetRedisSearch("search").Info(indices[0])
	assert.False(t, info.Indexing)
	assert.True(t, info.Options.NoFreqs)
	assert.False(t, info.Options.NoFields)
	assert.True(t, info.Options.NoOffsets)
	assert.False(t, info.Options.MaxTextFields)
	prefix := ""
	if redisNamespace != "" {
		prefix = redisNamespace + ":"
	}
	assert.Equal(t, []string{prefix + "7499e:"}, info.Definition.Prefixes)
	assert.Len(t, info.Fields, 25)
	assert.Equal(t, "ID", info.Fields[0].Name)
	assert.Equal(t, "NUMERIC", info.Fields[0].Type)
	assert.True(t, info.Fields[0].Sortable)
	assert.False(t, info.Fields[0].NoIndex)
	assert.Equal(t, "Age", info.Fields[1].Name)
	assert.Equal(t, "NUMERIC", info.Fields[1].Type)
	assert.True(t, info.Fields[1].Sortable)
	assert.False(t, info.Fields[1].NoIndex)
	assert.Equal(t, "Balance", info.Fields[2].Name)
	assert.Equal(t, "NUMERIC", info.Fields[2].Type)
	assert.True(t, info.Fields[2].Sortable)
	assert.True(t, info.Fields[2].NoIndex)
	assert.Equal(t, "Weight", info.Fields[3].Name)
	assert.Equal(t, "NUMERIC", info.Fields[3].Type)
	assert.False(t, info.Fields[3].Sortable)
	assert.False(t, info.Fields[3].NoIndex)
	assert.Equal(t, "AgeNullable", info.Fields[4].Name)
	assert.Equal(t, "NUMERIC", info.Fields[4].Type)
	assert.True(t, info.Fields[4].Sortable)
	assert.False(t, info.Fields[4].NoIndex)
	assert.Equal(t, "BalanceNullable", info.Fields[5].Name)
	assert.Equal(t, "NUMERIC", info.Fields[5].Type)
	assert.True(t, info.Fields[5].Sortable)
	assert.False(t, info.Fields[5].NoIndex)
	assert.Equal(t, "Enum", info.Fields[6].Name)
	assert.Equal(t, "TAG", info.Fields[6].Type)
	assert.False(t, info.Fields[6].Sortable)
	assert.False(t, info.Fields[6].NoIndex)
	assert.Equal(t, "EnumNullable", info.Fields[7].Name)
	assert.Equal(t, "TAG", info.Fields[7].Type)
	assert.False(t, info.Fields[7].Sortable)
	assert.False(t, info.Fields[7].NoIndex)
	assert.Equal(t, "Name", info.Fields[8].Name)
	assert.Equal(t, "TEXT", info.Fields[8].Type)
	assert.False(t, info.Fields[8].Sortable)
	assert.False(t, info.Fields[8].NoIndex)
	assert.True(t, info.Fields[8].NoStem)
	assert.Equal(t, 1.0, info.Fields[8].Weight)
	assert.Equal(t, "NameStem", info.Fields[9].Name)
	assert.Equal(t, "TEXT", info.Fields[9].Type)
	assert.False(t, info.Fields[9].Sortable)
	assert.False(t, info.Fields[9].NoIndex)
	assert.False(t, info.Fields[9].NoStem)
	assert.Equal(t, 1.0, info.Fields[9].Weight)
	assert.Equal(t, "Set", info.Fields[10].Name)
	assert.Equal(t, "TAG", info.Fields[10].Type)
	assert.False(t, info.Fields[10].Sortable)
	assert.False(t, info.Fields[10].NoIndex)
	assert.Equal(t, "SetNullable", info.Fields[11].Name)
	assert.Equal(t, "TAG", info.Fields[11].Type)
	assert.False(t, info.Fields[11].Sortable)
	assert.False(t, info.Fields[11].NoIndex)
	assert.Equal(t, "Bool", info.Fields[12].Name)
	assert.Equal(t, "TAG", info.Fields[12].Type)
	assert.True(t, info.Fields[12].Sortable)
	assert.False(t, info.Fields[12].NoIndex)
	assert.Equal(t, "BoolNullable", info.Fields[13].Name)
	assert.Equal(t, "TAG", info.Fields[13].Type)
	assert.False(t, info.Fields[13].Sortable)
	assert.False(t, info.Fields[13].NoIndex)
	assert.Equal(t, "WeightNullable", info.Fields[14].Name)
	assert.Equal(t, "NUMERIC", info.Fields[14].Type)
	assert.False(t, info.Fields[14].Sortable)
	assert.False(t, info.Fields[14].NoIndex)
	assert.Equal(t, "Date", info.Fields[15].Name)
	assert.Equal(t, "NUMERIC", info.Fields[15].Type)
	assert.False(t, info.Fields[15].Sortable)
	assert.False(t, info.Fields[15].NoIndex)
	assert.Equal(t, "DateTime", info.Fields[16].Name)
	assert.Equal(t, "NUMERIC", info.Fields[16].Type)
	assert.False(t, info.Fields[16].Sortable)
	assert.False(t, info.Fields[16].NoIndex)
	assert.Equal(t, "DateNullable", info.Fields[17].Name)
	assert.Equal(t, "NUMERIC", info.Fields[17].Type)
	assert.False(t, info.Fields[17].Sortable)
	assert.False(t, info.Fields[17].NoIndex)
	assert.Equal(t, "DateTimeNullable", info.Fields[18].Name)
	assert.Equal(t, "NUMERIC", info.Fields[18].Type)
	assert.False(t, info.Fields[18].Sortable)
	assert.False(t, info.Fields[18].NoIndex)
	assert.Equal(t, "Ref", info.Fields[19].Name)
	assert.Equal(t, "NUMERIC", info.Fields[19].Type)
	assert.False(t, info.Fields[19].Sortable)
	assert.False(t, info.Fields[19].NoIndex)
	assert.Equal(t, "Balance32", info.Fields[20].Name)
	assert.Equal(t, "NUMERIC", info.Fields[20].Type)
	assert.True(t, info.Fields[20].Sortable)
	assert.True(t, info.Fields[20].NoIndex)
	assert.Equal(t, "AgeNullable32", info.Fields[21].Name)
	assert.Equal(t, "NUMERIC", info.Fields[21].Type)
	assert.True(t, info.Fields[21].Sortable)
	assert.False(t, info.Fields[21].NoIndex)
	assert.Equal(t, "BalanceNullable32", info.Fields[22].Name)
	assert.Equal(t, "NUMERIC", info.Fields[22].Type)
	assert.True(t, info.Fields[22].Sortable)
	assert.True(t, info.Fields[22].NoIndex)
	assert.Equal(t, "ReferenceMany", info.Fields[23].Name)
	assert.Equal(t, "TEXT", info.Fields[23].Type)
	assert.False(t, info.Fields[23].Sortable)
	assert.False(t, info.Fields[23].NoIndex)
	assert.True(t, info.Fields[23].NoStem)
	assert.Equal(t, "FakeDelete", info.Fields[24].Name)
	assert.Equal(t, "TAG", info.Fields[24].Type)
	assert.False(t, info.Fields[23].Sortable)
	assert.False(t, info.Fields[23].NoIndex)

	query := NewRedisSearchQuery()
	query.Sort("Age", false)
	ids, total := engine.RedisSearchIds(entity, query, NewPager(1, 10))
	assert.Equal(t, uint64(50), total)
	assert.Equal(t, uint64(1), ids[0])
	assert.Equal(t, uint64(2), ids[1])
	assert.Equal(t, uint64(10), ids[9])
	assert.Len(t, ids, 10)
	query.FilterIntMinMax("Age", 6, 8)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 10))
	assert.Equal(t, uint64(3), total)
	assert.Len(t, ids, 3)
	assert.Equal(t, uint64(6), ids[0])
	assert.Equal(t, uint64(7), ids[1])
	assert.Equal(t, uint64(8), ids[2])

	query = &RedisSearchQuery{}
	query.Sort("ID", true)
	query.FilterInt("ID", 4, 6, 2)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 10))
	assert.Equal(t, uint64(3), total)
	assert.Len(t, ids, 3)
	assert.Equal(t, uint64(6), ids[0])
	assert.Equal(t, uint64(4), ids[1])
	assert.Equal(t, uint64(2), ids[2])

	query = &RedisSearchQuery{}
	query.Sort("ID", true)
	query.FilterInt("ID", 4, 6, 2)
	query.FilterInt("Age", 2, 4)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 10))
	assert.Equal(t, uint64(2), total)
	assert.Len(t, ids, 2)
	assert.Equal(t, uint64(4), ids[0])
	assert.Equal(t, uint64(2), ids[1])

	query = &RedisSearchQuery{}
	query.Sort("ID", true)
	query.FilterString("Name", "dog")
	query.FilterString("NameStem", "carrot")
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 10))
	assert.Equal(t, uint64(20), total)
	assert.Len(t, ids, 10)
	assert.Equal(t, uint64(20), ids[0])
	assert.Equal(t, uint64(11), ids[9])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterIntGreaterEqual("Age", 20)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 10))
	assert.Equal(t, uint64(31), total)
	assert.Len(t, ids, 10)
	assert.Equal(t, uint64(20), ids[0])
	assert.Equal(t, uint64(21), ids[1])
	assert.Equal(t, uint64(29), ids[9])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterIntLessEqual("Age", 20)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 30))
	assert.Equal(t, uint64(20), total)
	assert.Len(t, ids, 20)
	assert.Equal(t, uint64(20), ids[19])
	assert.Equal(t, uint64(19), ids[18])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterIntGreater("Age", 20)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 10))
	assert.Equal(t, uint64(30), total)
	assert.Len(t, ids, 10)
	assert.Equal(t, uint64(21), ids[0])
	assert.Equal(t, uint64(22), ids[1])
	assert.Equal(t, uint64(30), ids[9])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterIntLess("Age", 20)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 30))
	assert.Equal(t, uint64(19), total)
	assert.Len(t, ids, 19)
	assert.Equal(t, uint64(19), ids[18])
	assert.Equal(t, uint64(18), ids[17])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterInt("Age", 18)
	query.FilterInt("Age", 38)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 30))
	assert.Equal(t, uint64(2), total)
	assert.Len(t, ids, 2)
	assert.Equal(t, uint64(18), ids[0])
	assert.Equal(t, uint64(38), ids[1])

	query = &RedisSearchQuery{}
	query.Sort("ID", false)
	query.FilterManyReferenceIn("ReferenceMany", 1)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 30))
	assert.Equal(t, uint64(30), total)
	assert.Len(t, ids, 30)
	assert.Equal(t, uint64(21), ids[0])
	assert.Equal(t, uint64(50), ids[29])

	query = &RedisSearchQuery{}
	query.Sort("ID", false)
	query.FilterManyReferenceIn("ReferenceMany", 3, 2)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 30))
	assert.Equal(t, uint64(10), total)
	assert.Len(t, ids, 10)
	assert.Equal(t, uint64(41), ids[0])
	assert.Equal(t, uint64(50), ids[9])

	query = &RedisSearchQuery{}
	query.Sort("ID", false)
	query.FilterManyReferenceNotIn("ReferenceMany", 2, 3)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(40), total)
	assert.Len(t, ids, 40)
	assert.Equal(t, uint64(1), ids[0])
	assert.Equal(t, uint64(40), ids[39])

	query = &RedisSearchQuery{}
	query.Sort("ID", false)
	query.FilterManyReferenceIn("ReferenceMany", 1)
	query.FilterManyReferenceIn("ReferenceMany", 3)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(5), total)
	assert.Len(t, ids, 5)
	assert.Equal(t, uint64(46), ids[0])
	assert.Equal(t, uint64(50), ids[4])

	query = &RedisSearchQuery{}
	query.Sort("Balance", false)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 3))
	assert.Equal(t, uint64(50), total)
	assert.Len(t, ids, 3)
	assert.Equal(t, uint64(50), ids[0])
	assert.Equal(t, uint64(49), ids[1])
	assert.Equal(t, uint64(48), ids[2])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterFloat("Weight", 101.3)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 10))
	assert.Equal(t, uint64(1), total)
	assert.Len(t, ids, 1)
	assert.Equal(t, uint64(1), ids[0])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterFloatMinMax("Weight", 105, 116.3)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 20))
	assert.Equal(t, uint64(12), total)
	assert.Len(t, ids, 12)
	assert.Equal(t, uint64(5), ids[0])
	assert.Equal(t, uint64(16), ids[11])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterFloatGreater("Weight", 148.3)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 20))
	assert.Equal(t, uint64(2), total)
	assert.Len(t, ids, 2)
	assert.Equal(t, uint64(49), ids[0])
	assert.Equal(t, uint64(50), ids[1])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterFloatGreaterEqual("Weight", 148.3)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 20))
	assert.Equal(t, uint64(3), total)
	assert.Len(t, ids, 3)
	assert.Equal(t, uint64(48), ids[0])
	assert.Equal(t, uint64(49), ids[1])
	assert.Equal(t, uint64(50), ids[2])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterFloatLess("Weight", 103.3)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 20))
	assert.Equal(t, uint64(2), total)
	assert.Len(t, ids, 2)
	assert.Equal(t, uint64(1), ids[0])
	assert.Equal(t, uint64(2), ids[1])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterFloatLessEqual("Weight", 103.3)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 20))
	assert.Equal(t, uint64(3), total)
	assert.Len(t, ids, 3)
	assert.Equal(t, uint64(1), ids[0])
	assert.Equal(t, uint64(2), ids[1])
	assert.Equal(t, uint64(3), ids[2])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterIntGreaterEqual("AgeNullable", 0)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 10))
	assert.Equal(t, uint64(30), total)
	assert.Len(t, ids, 10)
	assert.Equal(t, uint64(21), ids[0])
	assert.Equal(t, uint64(22), ids[1])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterIntNull("AgeNullable")
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 10))
	assert.Equal(t, uint64(20), total)
	assert.Len(t, ids, 10)
	assert.Equal(t, uint64(1), ids[0])
	assert.Equal(t, uint64(2), ids[1])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterIntGreaterEqual("BalanceNullable", 0)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 10))
	assert.Equal(t, uint64(30), total)
	assert.Len(t, ids, 10)
	assert.Equal(t, uint64(21), ids[0])
	assert.Equal(t, uint64(22), ids[1])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterTag("Enum", "a", "c")
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(30), total)
	assert.Len(t, ids, 30)
	assert.Equal(t, uint64(1), ids[0])
	assert.Equal(t, uint64(41), ids[20])
	assert.Equal(t, uint64(50), ids[29])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterTag("Enum", "a", "c")
	query.FilterTag("EnumNullable", "c")
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(10), total)
	assert.Len(t, ids, 10)
	assert.Equal(t, uint64(41), ids[0])
	assert.Equal(t, uint64(50), ids[9])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterNotTag("Enum", "a", "c")
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(20), total)
	assert.Len(t, ids, 20)
	assert.Equal(t, uint64(21), ids[0])
	assert.Equal(t, uint64(40), ids[19])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterNotString("Name", "dog")
	query.FilterNotString("NameStem", "carrot")
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(30), total)
	assert.Len(t, ids, 30)
	assert.Equal(t, uint64(21), ids[0])
	assert.Equal(t, uint64(50), ids[29])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterNotInt("Age", 30)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(49), total)
	assert.Len(t, ids, 49)
	assert.Equal(t, uint64(29), ids[28])
	assert.Equal(t, uint64(31), ids[29])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterNotInt("Age", 30, 31)
	query.FilterNotInt("ID", 7)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(47), total)
	assert.Len(t, ids, 47)
	assert.Equal(t, uint64(29), ids[27])
	assert.Equal(t, uint64(32), ids[28])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterTag("EnumNullable", "", "c")
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(30), total)
	assert.Len(t, ids, 30)
	assert.Equal(t, uint64(1), ids[0])
	assert.Equal(t, uint64(41), ids[20])
	assert.Equal(t, uint64(50), ids[29])

	total = engine.RedisSearchCount(entity, query)
	assert.Equal(t, uint64(30), total)

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.Query("dog")
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(20), total)
	assert.Len(t, ids, 20)
	assert.Equal(t, uint64(1), ids[0])
	assert.Equal(t, uint64(20), ids[19])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.Query("dog 20")
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(1), total)
	assert.Len(t, ids, 1)
	assert.Equal(t, uint64(20), ids[0])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.Query("cat")
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(20), total)
	assert.Len(t, ids, 20)
	assert.Equal(t, uint64(21), ids[0])
	assert.Equal(t, uint64(40), ids[19])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.Query("orange")
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(30), total)
	assert.Len(t, ids, 30)
	assert.Equal(t, uint64(21), ids[0])
	assert.Equal(t, uint64(50), ids[29])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterTag("Set", "b", "c")
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(30), total)
	assert.Len(t, ids, 30)
	assert.Equal(t, uint64(21), ids[0])
	assert.Equal(t, uint64(50), ids[29])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterTag("SetNullable", "NULL", "c")
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(30), total)
	assert.Len(t, ids, 30)
	assert.Equal(t, uint64(1), ids[0])
	assert.Equal(t, uint64(20), ids[19])
	assert.Equal(t, uint64(50), ids[29])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterBool("Bool", true)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(10), total)
	assert.Len(t, ids, 10)
	assert.Equal(t, uint64(41), ids[0])
	assert.Equal(t, uint64(50), ids[9])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterBool("Bool", false)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(40), total)
	assert.Len(t, ids, 40)
	assert.Equal(t, uint64(1), ids[0])
	assert.Equal(t, uint64(40), ids[39])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterTag("BoolNullable", "NULL", "true")
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 30))
	assert.Equal(t, uint64(30), total)
	assert.Len(t, ids, 30)
	assert.Equal(t, uint64(1), ids[0])
	assert.Equal(t, uint64(50), ids[29])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterFloatGreaterEqual("WeightNullable", 0)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 10))
	assert.Equal(t, uint64(30), total)
	assert.Len(t, ids, 10)
	assert.Equal(t, uint64(21), ids[0])
	assert.Equal(t, uint64(22), ids[1])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterFloatNull("WeightNullable")
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 30))
	assert.Equal(t, uint64(20), total)
	assert.Len(t, ids, 20)
	assert.Equal(t, uint64(1), ids[0])
	assert.Equal(t, uint64(20), ids[19])

	newNow := time.Now()
	newNow = newNow.Add(time.Second * 5)
	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterDate("Date", newNow)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 30))
	assert.Equal(t, uint64(20), total)
	assert.Len(t, ids, 20)
	assert.Equal(t, uint64(21), ids[0])
	assert.Equal(t, uint64(40), ids[19])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterDateGreater("Date", newNow)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 30))
	assert.Equal(t, uint64(10), total)
	assert.Len(t, ids, 10)
	assert.Equal(t, uint64(41), ids[0])
	assert.Equal(t, uint64(50), ids[9])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterDateMinMax("Date", newNow, newNow)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 30))
	assert.Equal(t, uint64(20), total)
	assert.Len(t, ids, 20)
	assert.Equal(t, uint64(21), ids[0])
	assert.Equal(t, uint64(40), ids[19])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterDateGreaterEqual("Date", newNow)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 30))
	assert.Equal(t, uint64(30), total)
	assert.Len(t, ids, 30)
	assert.Equal(t, uint64(21), ids[0])
	assert.Equal(t, uint64(50), ids[29])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterDateLessEqual("Date", newNow)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(40), total)
	assert.Len(t, ids, 40)
	assert.Equal(t, uint64(1), ids[0])
	assert.Equal(t, uint64(40), ids[39])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterDateLess("Date", newNow)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(20), total)
	assert.Len(t, ids, 20)
	assert.Equal(t, uint64(1), ids[0])
	assert.Equal(t, uint64(20), ids[19])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterNotDate("Date", newNow)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 30))
	assert.Equal(t, uint64(30), total)
	assert.Len(t, ids, 30)
	assert.Equal(t, uint64(1), ids[0])
	assert.Equal(t, uint64(50), ids[29])

	query = &RedisSearchQuery{}
	query.Sort("ID", false)
	query.FilterNotDateNull("DateNullable")
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 30))
	assert.Equal(t, uint64(30), total)
	assert.Len(t, ids, 30)
	assert.Equal(t, uint64(21), ids[0])
	assert.Equal(t, uint64(50), ids[29])

	newNow = now.Add(time.Microsecond * 3)
	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterDateTime("DateTime", newNow)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 30))
	assert.Equal(t, uint64(20), total)
	assert.Len(t, ids, 20)
	assert.Equal(t, uint64(21), ids[0])
	assert.Equal(t, uint64(40), ids[19])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterDateTimeGreater("DateTime", newNow)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 30))
	assert.Equal(t, uint64(10), total)
	assert.Len(t, ids, 10)
	assert.Equal(t, uint64(41), ids[0])
	assert.Equal(t, uint64(50), ids[9])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterDateTimeMinMax("DateTime", newNow, newNow)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 30))
	assert.Equal(t, uint64(20), total)
	assert.Len(t, ids, 20)
	assert.Equal(t, uint64(21), ids[0])
	assert.Equal(t, uint64(40), ids[19])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterDateTimeNull("DateTimeNullable")
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 30))
	assert.Equal(t, uint64(20), total)
	assert.Len(t, ids, 20)
	assert.Equal(t, uint64(1), ids[0])
	assert.Equal(t, uint64(20), ids[19])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterDateTimeGreaterEqual("DateTime", newNow)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 30))
	assert.Equal(t, uint64(30), total)
	assert.Len(t, ids, 30)
	assert.Equal(t, uint64(21), ids[0])
	assert.Equal(t, uint64(50), ids[29])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterDateTimeLessEqual("DateTime", newNow)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(40), total)
	assert.Len(t, ids, 40)
	assert.Equal(t, uint64(1), ids[0])
	assert.Equal(t, uint64(40), ids[39])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterDateTimeLess("DateTime", newNow)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(20), total)
	assert.Len(t, ids, 20)
	assert.Equal(t, uint64(1), ids[0])
	assert.Equal(t, uint64(20), ids[19])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterDateNull("DateNullable")
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 30))
	assert.Equal(t, uint64(20), total)
	assert.Len(t, ids, 20)
	assert.Equal(t, uint64(1), ids[0])
	assert.Equal(t, uint64(20), ids[19])

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.FilterInt("Ref", 32)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 30))
	assert.Equal(t, uint64(3), total)
	assert.Len(t, ids, 3)
	assert.Equal(t, uint64(3), ids[0])
	assert.Equal(t, uint64(4), ids[1])
	assert.Equal(t, uint64(5), ids[2])

	entity = &redisSearchEntity{}
	engine.LoadByID(40, entity)
	engine.Delete(entity)

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(49), total)
	assert.Len(t, ids, 49)

	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	query.QueryRaw("(@Bool:{true})")
	query.AppendQueryRaw(" | (@Ref:[32 32])")
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(13), total)
	assert.Len(t, ids, 13)

	entity = &redisSearchEntity{}
	engine.LoadByID(1, entity)
	entity.Age = 100
	flusher = engine.NewFlusher()
	flusher.Track(entity)
	flusher.Flush()

	query = &RedisSearchQuery{}
	query.Sort("Age", false).FilterInt("Age", 100)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(1), total)
	assert.Len(t, ids, 1)
	assert.Equal(t, uint64(1), ids[0])

	entity.Age = 101
	engine.FlushLazy(entity)
	receiver := NewBackgroundConsumer(engine)
	receiver.DisableLoop()
	receiver.blockTime = time.Millisecond
	receiver.Digest(context.Background())

	query = &RedisSearchQuery{}
	query.Sort("Age", false).FilterInt("Age", 101)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 50))
	assert.Equal(t, uint64(1), total)
	assert.Len(t, ids, 1)
	assert.Equal(t, uint64(1), ids[0])

	engine.GetRedis("search").FlushDB()
	for _, alter := range engine.GetRedisSearchIndexAlters() {
		alter.Execute()
	}
	indexer.Digest(context.Background())
	query = &RedisSearchQuery{}
	query.Sort("Age", false)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 10))
	assert.Equal(t, uint64(49), total)
	assert.Len(t, ids, 10)

	entities := make([]*redisSearchEntity, 0)
	total = engine.RedisSearch(&entities, query, NewPager(1, 10))
	assert.Equal(t, uint64(49), total)
	assert.Len(t, entities, 10)
	assert.Equal(t, "dog 2", entities[0].Name)
	assert.Equal(t, "dog 11", entities[9].Name)

	query.FilterInt("Age", 10)
	assert.True(t, engine.RedisSearchOne(entity, query))
	assert.Equal(t, "dog 10", entity.Name)

	query.FilterInt("Balance", 700)
	assert.False(t, engine.RedisSearchOne(entity, query))

	engine.LoadByID(40, entity)
	engine.ForceDelete(entity)
	query = NewRedisSearchQuery()
	total = engine.RedisSearchCount(entity, query)
	assert.Equal(t, uint64(49), total)

	entity = &redisSearchEntity{}
	engine.LoadByID(29, entity)
	entity.Name = "test coming soon to"
	engine.Flush(entity)
	query = &RedisSearchQuery{}
	query.QueryFieldPrefixMatch("Name", "test coming soon to")
	total = engine.RedisSearch(&entities, query, NewPager(1, 100))
	assert.Equal(t, uint64(1), total)
	assert.Equal(t, uint(29), entities[0].ID)

	entity = &redisSearchEntity{}
	engine.LoadByID(1, entity)
	entity.Age = 120
	entity.Name = ""
	engine.Flush(entity)
	query = &RedisSearchQuery{}
	query.FilterString("Name", "")
	assert.True(t, engine.RedisSearchOne(entity, query))
	assert.PanicsWithError(t, "unknown field Name2", func() {
		query = &RedisSearchQuery{}
		query.FilterString("Name2", "")
		engine.RedisSearchOne(entity, query)
	})
	assert.PanicsWithError(t, "missing `searchable` tag for field Another", func() {
		query = &RedisSearchQuery{}
		query.FilterString("Another", "")
		engine.RedisSearchOne(entity, query)
	})
	assert.PanicsWithError(t, "string filter on fields Weight with type NUMERIC not allowed", func() {
		query = &RedisSearchQuery{}
		query.FilterString("Weight", "")
		engine.RedisSearchOne(entity, query)
	})
	assert.PanicsWithError(t, "unknown field Name2", func() {
		query = &RedisSearchQuery{}
		query.FilterInt("Name2", 23)
		engine.RedisSearchOne(entity, query)
	})
	assert.PanicsWithError(t, "missing `searchable` tag for field AnotherNumeric", func() {
		query = &RedisSearchQuery{}
		query.FilterInt("AnotherNumeric", 23)
		engine.RedisSearchOne(entity, query)
	})
	assert.PanicsWithError(t, "numeric filter on fields Name with type TEXT not allowed", func() {
		query = &RedisSearchQuery{}
		query.FilterInt("Name", 23)
		engine.RedisSearchOne(entity, query)
	})

	assert.PanicsWithError(t, "unknown field Name2", func() {
		query = &RedisSearchQuery{}
		query.FilterTag("Name2", "test")
		engine.RedisSearchOne(entity, query)
	})
	assert.PanicsWithError(t, "missing `searchable` tag for field AnotherTag", func() {
		query = &RedisSearchQuery{}
		query.FilterTag("AnotherTag", "test")
		engine.RedisSearchOne(entity, query)
	})
	assert.PanicsWithError(t, "tag filter on fields Name with type TEXT not allowed", func() {
		query = &RedisSearchQuery{}
		query.FilterTag("Name", "test")
		engine.RedisSearchOne(entity, query)
	})

	assert.PanicsWithError(t, "integer too high for redis search sort field", func() {
		entity = &redisSearchEntity{}
		engine.LoadByID(9, entity)
		entity.Balance = math.MaxInt64
		engine.Flush(entity)
	})

	assert.PanicsWithError(t, "entity 'string' is not registered", func() {
		query = &RedisSearchQuery{}
		engine.RedisSearch(&[]*string{}, query, NewPager(1, 100))
	})

	assert.PanicsWithError(t, "entity beeorm.redisNoSearchEntity is not searchable", func() {
		query = &RedisSearchQuery{}
		engine.RedisSearch(&[]*redisNoSearchEntity{}, query, NewPager(1, 100))
	})

	engine.Flush(&redisSearchEntity{Age: 133})
	schema := engine.GetRegistry().GetTableSchemaForEntity(entity)
	schema.ReindexRedisSearchIndex(engine)
	indexer.Digest(context.Background())
	query = NewRedisSearchQuery()
	query.FilterInt("Age", 133)
	found := engine.RedisSearchOne(entity, query)
	assert.True(t, found)

	engine.LoadByID(6, entity)
	entity.Enum = "b"
	engine.Flush(entity)
	query = NewRedisSearchQuery()
	query.QueryField("Name", "dog")
	query.FilterNotTag("Enum", "b")
	total = engine.RedisSearch(&[]*redisSearchEntity{}, query, NewPager(1, 100))
	assert.Equal(t, uint64(18), total)

	query = NewRedisSearchQuery()
	query.FilterNotTag("EnumNullable", "")
	total = engine.RedisSearch(&[]*redisSearchEntity{}, query, NewPager(1, 100))
	assert.Equal(t, uint64(29), total)

	entitySearch, has := schema.GetRedisSearch(engine)
	assert.True(t, has)
	assert.Equal(t, "search", entitySearch.GetPoolConfig().GetCode())

	type redisSearchEntity2 struct {
		ORM `orm:"redisSearch=invalid"`
		ID  uint `orm:"searchable;sortable"`
	}
	registry = NewRegistry()
	registry.RegisterEntity(&redisSearchEntity2{})
	registry.RegisterMySQLPool("root:root@tcp(localhost:3312)/test")
	_, _, err := registry.Validate()
	assert.EqualError(t, err, "redis pool 'invalid' not found")

	assert.PanicsWithError(t, "integer too high for redis search sort field", func() {
		engine.Flush(&redisSearchEntity{Age: math.MaxInt32 + 1})
	})
	assert.PanicsWithError(t, "integer too high for redis search sort field", func() {
		v := uint64(math.MaxInt32 + 1)
		engine.Flush(&redisSearchEntity{AgeNullable: &v})
	})
	assert.PanicsWithError(t, "integer too high for redis search sort field", func() {
		v := int64(math.MaxInt32 + 1)
		engine.Flush(&redisSearchEntity{BalanceNullable: &v})
	})

	type redisSearchEntity3 struct {
		ORM
		ID uint
	}
	registry = NewRegistry()
	registry.RegisterMySQLPool("root:root@tcp(localhost:3312)/test")
	registry.RegisterEntity(&redisSearchEntity3{})
	def()
	vRegistry, def, err := registry.Validate()
	defer def()
	assert.NoError(t, err)
	schema = vRegistry.GetTableSchemaForEntity(&redisSearchEntity3{})
	entitySearch, has = schema.GetRedisSearch(engine)
	assert.Nil(t, entitySearch)
	assert.False(t, has)
}

func TestEntityRedisAggregate(t *testing.T) {
	var entity *redisSearchAggregateEntity
	registry := &Registry{}
	engine, def := prepareTables(t, registry, 5, "", "2.0", entity)
	defer def()
	flusher := engine.NewFlusher()
	for i := 1; i <= 50; i++ {
		e := &redisSearchAggregateEntity{}
		age := 18
		size := 10
		name := "Adam"
		if i > 20 {
			age = 39
			size = 20
		}
		if i > 30 {
			size = 30
		}
		if i > 35 {
			name = "John"
		}
		e.Age = age
		e.Size = size
		e.Name = name
		flusher.Track(e)
	}
	flusher.Flush()
	query := &RedisSearchAggregate{}
	query.GroupByFields([]string{"@Age", "@Size"}, NewAggregateReduceCount("rows"))
	query.Sort(RedisSearchAggregateSort{"@rows", true}, RedisSearchAggregateSort{"@Size", false})
	res, totalRows := engine.RedisSearchAggregate(entity, query, NewPager(1, 100))
	assert.Equal(t, uint64(3), totalRows)
	assert.Len(t, res, 3)
	assert.Equal(t, "18", res[0]["Age"])
	assert.Equal(t, "10", res[0]["Size"])
	assert.Equal(t, "20", res[0]["rows"])
	assert.Equal(t, "39", res[1]["Age"])
	assert.Equal(t, "30", res[1]["Size"])
	assert.Equal(t, "20", res[1]["rows"])
	assert.Equal(t, "39", res[2]["Age"])
	assert.Equal(t, "20", res[2]["Size"])
	assert.Equal(t, "10", res[2]["rows"])

	query = &RedisSearchAggregate{}
	query.GroupByFields([]string{"@Age"}, NewAggregateReduceCountDistinct("@Size", "sizes", false))
	query.Sort(RedisSearchAggregateSort{"@sizes", false})
	res, totalRows = engine.RedisSearchAggregate(entity, query, NewPager(1, 100))
	assert.Equal(t, uint64(2), totalRows)
	assert.Len(t, res, 2)
	assert.Equal(t, "18", res[0]["Age"])
	assert.Equal(t, "1", res[0]["sizes"])
	assert.Equal(t, "39", res[1]["Age"])
	assert.Equal(t, "2", res[1]["sizes"])

	query = &RedisSearchAggregate{}
	query.GroupByFields([]string{"@Age"}, NewAggregateReduceCountDistinct("@Size", "sizes", true))
	query.Sort(RedisSearchAggregateSort{"@sizes", false})
	res, totalRows = engine.RedisSearchAggregate(entity, query, NewPager(1, 100))
	assert.Equal(t, uint64(2), totalRows)
	assert.Len(t, res, 2)

	query = &RedisSearchAggregate{}
	query.GroupByField("@Age", NewAggregateReduceSum("@Size", "total"))
	query.Sort(RedisSearchAggregateSort{"@total", false})
	res, totalRows = engine.RedisSearchAggregate(entity, query, NewPager(1, 100))
	assert.Equal(t, uint64(2), totalRows)
	assert.Len(t, res, 2)
	assert.Equal(t, "18", res[0]["Age"])
	assert.Equal(t, "200", res[0]["total"])
	assert.Equal(t, "39", res[1]["Age"])
	assert.Equal(t, "800", res[1]["total"])

	query = &RedisSearchAggregate{}
	query.GroupByFields([]string{"@Age"}, NewAggregateReduceMin("@Size", "total"))
	query.Sort(RedisSearchAggregateSort{"@total", false})
	res, totalRows = engine.RedisSearchAggregate(entity, query, NewPager(1, 100))
	assert.Equal(t, uint64(2), totalRows)
	assert.Len(t, res, 2)
	assert.Equal(t, "18", res[0]["Age"])
	assert.Equal(t, "10", res[0]["total"])
	assert.Equal(t, "39", res[1]["Age"])
	assert.Equal(t, "20", res[1]["total"])

	query = &RedisSearchAggregate{}
	query.GroupByFields([]string{"@Age"}, NewAggregateReduceMax("@Size", "total"))
	query.Sort(RedisSearchAggregateSort{"@total", false})
	res, totalRows = engine.RedisSearchAggregate(entity, query, NewPager(1, 100))
	assert.Equal(t, uint64(2), totalRows)
	assert.Len(t, res, 2)
	assert.Equal(t, "18", res[0]["Age"])
	assert.Equal(t, "10", res[0]["total"])
	assert.Equal(t, "39", res[1]["Age"])
	assert.Equal(t, "30", res[1]["total"])

	query = &RedisSearchAggregate{}
	query.GroupByFields([]string{"@Age"}, NewAggregateReduceAvg("@Size", "total"))
	query.Sort(RedisSearchAggregateSort{"@total", false})
	res, totalRows = engine.RedisSearchAggregate(entity, query, NewPager(1, 100))
	assert.Equal(t, uint64(2), totalRows)
	assert.Len(t, res, 2)
	assert.Equal(t, "18", res[0]["Age"])
	assert.Equal(t, "10", res[0]["total"])
	assert.Equal(t, "39", res[1]["Age"])
	assert.Equal(t, "26.6666666667", res[1]["total"])

	query = &RedisSearchAggregate{}
	query.GroupByFields([]string{"@Age"}, NewAggregateReduceStdDev("@Size", "total"))
	query.Sort(RedisSearchAggregateSort{"@total", false})
	res, totalRows = engine.RedisSearchAggregate(entity, query, NewPager(1, 100))
	assert.Equal(t, uint64(2), totalRows)
	assert.Len(t, res, 2)
	assert.Equal(t, "18", res[0]["Age"])
	assert.Equal(t, "0", res[0]["total"])
	assert.Equal(t, "39", res[1]["Age"])
	assert.Equal(t, "4.79463301485", res[1]["total"])

	query = &RedisSearchAggregate{}
	query.GroupByFields([]string{"@Age"}, NewAggregateReduceQuantile("@Size", "0.1", "total"))
	query.Sort(RedisSearchAggregateSort{"@total", false})
	res, totalRows = engine.RedisSearchAggregate(entity, query, NewPager(1, 100))
	assert.Equal(t, uint64(2), totalRows)
	assert.Len(t, res, 2)
	assert.Equal(t, "18", res[0]["Age"])
	assert.Equal(t, "10", res[0]["total"])
	assert.Equal(t, "39", res[1]["Age"])
	assert.Equal(t, "20", res[1]["total"])

	query = &RedisSearchAggregate{}
	query.GroupByFields([]string{"@Age"}, NewAggregateReduceToList("@Size", "list"))
	query.Sort(RedisSearchAggregateSort{"@Age", false})
	res, totalRows = engine.RedisSearchAggregate(entity, query, NewPager(1, 100))
	assert.Equal(t, uint64(2), totalRows)
	assert.Len(t, res, 2)
	assert.Equal(t, "18", res[0]["Age"])
	assert.Equal(t, "10", res[0]["list"])
	assert.Equal(t, "39", res[1]["Age"])
	assert.True(t, "20,30" == res[1]["list"] || "30,20" == res[1]["list"])

	query = &RedisSearchAggregate{}
	query.GroupByFields([]string{"@Age"}, NewAggregateReduceFirstValue("@Size", "first"))
	query.Sort(RedisSearchAggregateSort{"@Age", false})
	res, totalRows = engine.RedisSearchAggregate(entity, query, NewPager(1, 100))
	assert.Equal(t, uint64(2), totalRows)
	assert.Len(t, res, 2)
	assert.Equal(t, "18", res[0]["Age"])
	assert.Equal(t, "39", res[1]["Age"])

	query = &RedisSearchAggregate{}
	query.GroupByFields([]string{"@Age"}, NewAggregateReduceFirstValueBy("@Size", "@Age", "first", true))
	query.Sort(RedisSearchAggregateSort{"@Age", true})
	res, totalRows = engine.RedisSearchAggregate(entity, query, NewPager(1, 100))
	assert.Equal(t, uint64(2), totalRows)
	assert.Len(t, res, 2)
	assert.Equal(t, "39", res[0]["Age"])
	assert.Equal(t, "18", res[1]["Age"])

	query = &RedisSearchAggregate{}
	query.GroupByFields([]string{"@Age"}, NewAggregateReduceRandomSample("@Size", "res", 2))
	query.Sort(RedisSearchAggregateSort{"@Age", true})
	res, totalRows = engine.RedisSearchAggregate(entity, query, NewPager(1, 100))
	assert.Equal(t, uint64(2), totalRows)
	assert.Len(t, res, 2)
	assert.Equal(t, "39", res[0]["Age"])
	assert.Equal(t, "18", res[1]["Age"])

	query = &RedisSearchAggregate{}
	query.GroupByFields([]string{"@Name"}, NewAggregateReduceCountDistinct("@Age", "ages", false),
		NewAggregateReduceCountDistinct("@Size", "sizes", false))
	query.Apply("upper(@Name)", "upperName")
	query.Sort(RedisSearchAggregateSort{"@Name", false})
	res, totalRows = engine.RedisSearchAggregate(entity, query, NewPager(1, 100))
	assert.Equal(t, uint64(2), totalRows)
	assert.Len(t, res, 2)
	assert.Equal(t, "adam", res[0]["Name"])
	assert.Equal(t, "john", res[1]["Name"])
	assert.Equal(t, "ADAM", res[0]["upperName"])
	assert.Equal(t, "JOHN", res[1]["upperName"])
	assert.Equal(t, "2", res[0]["ages"])
	assert.Equal(t, "1", res[1]["ages"])
	assert.Equal(t, "3", res[0]["sizes"])
	assert.Equal(t, "1", res[1]["sizes"])

	query = &RedisSearchAggregate{}
	query.GroupByFields([]string{"@Age"}, NewAggregateReduceCountDistinct("@Size", "sizes", false))
	query.Filter("@Age > 18")
	query.Sort(RedisSearchAggregateSort{"@Age", false})
	res, totalRows = engine.RedisSearchAggregate(entity, query, NewPager(1, 100))
	assert.Equal(t, uint64(2), totalRows)
	assert.Len(t, res, 2)
	assert.Equal(t, "39", res[0]["Age"])
	assert.Equal(t, "", res[1]["Age"])

	q := &RedisSearchQuery{}
	q.Sort("ID", true)
	q.FilterInt("ID", 4, 6, 2)
	query = q.Aggregate()
	query.GroupByFields([]string{"@Age"}, NewAggregateReduceCountDistinct("@Size", "sizes", false))
	res, totalRows = engine.RedisSearchAggregate(entity, query, NewPager(1, 100))
	assert.Len(t, res, 1)
	assert.Equal(t, "18", res[0]["Age"])
	assert.Equal(t, "1", res[0]["sizes"])
}

func TestEntityOnlySortPKRedisSearch(t *testing.T) {
	var entity *redisSearchOnlySortPKEntity
	registry := &Registry{}
	engine, def := prepareTables(t, registry, 5, "", "2.0", entity)
	defer def()
	flusher := engine.NewFlusher()
	for i := 1; i <= 50; i++ {
		e := &redisSearchOnlySortPKEntity{}
		flusher.Track(e)
	}
	flusher.Flush()

	query := &RedisSearchQuery{}
	query.Sort("ID", false)
	ids, total := engine.RedisSearchIds(entity, query, NewPager(1, 100))
	assert.Equal(t, uint64(50), total)
	assert.Len(t, ids, 50)
	assert.Equal(t, uint64(1), ids[0])
	assert.Equal(t, uint64(30), ids[29])
	assert.Equal(t, uint64(50), ids[49])

	e := &redisSearchOnlySortPKEntity{}
	e.SetOnDuplicateKeyUpdate(Bind{"ID": 60})
	engine.Flush(e)

	query = &RedisSearchQuery{}
	query.Sort("ID", true)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 100))
	assert.Equal(t, uint64(51), total)
	assert.Len(t, ids, 51)
	assert.Equal(t, uint64(51), ids[0])
	assert.Equal(t, uint64(1), ids[50])

	e = &redisSearchOnlySortPKEntity{}
	engine.LoadByID(2, e)
	engine.Delete(e)

	query = &RedisSearchQuery{}
	query.Sort("ID", false)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 3))
	assert.Equal(t, uint64(50), total)
	assert.Len(t, ids, 3)
	assert.Equal(t, uint64(1), ids[0])
	assert.Equal(t, uint64(3), ids[1])
	assert.Equal(t, uint64(4), ids[2])
}

func TestEntityFakeDeleteRedisSearch(t *testing.T) {
	var entity *redisSearchFakeDeleteEntity
	registry := &Registry{}
	engine, def := prepareTables(t, registry, 5, "", "2.0", entity)
	defer def()
	flusher := engine.NewFlusher()
	for i := 1; i <= 50; i++ {
		e := &redisSearchFakeDeleteEntity{}
		flusher.Track(e)
	}
	flusher.Flush()

	query := &RedisSearchQuery{}
	query.Sort("ID", false)
	ids, total := engine.RedisSearchIds(entity, query, NewPager(1, 100))
	assert.Equal(t, uint64(50), total)
	assert.Len(t, ids, 50)
	assert.Equal(t, uint64(1), ids[0])
	assert.Equal(t, uint64(30), ids[29])
	assert.Equal(t, uint64(50), ids[49])

	entity = &redisSearchFakeDeleteEntity{}
	engine.LoadByID(10, entity)
	engine.Delete(entity)

	query = &RedisSearchQuery{}
	query.Sort("ID", false)
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 100))
	assert.Equal(t, uint64(49), total)
	assert.Len(t, ids, 49)

	query = &RedisSearchQuery{}
	query.Sort("ID", false)
	query.WithFakeDeleteRows()
	ids, total = engine.RedisSearchIds(entity, query, NewPager(1, 100))
	assert.Equal(t, uint64(50), total)
	assert.Len(t, ids, 50)
}
