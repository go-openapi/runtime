// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2014 Naoya Inada <naoina@kuune.org>
// SPDX-License-Identifier: MIT

package denco_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"

	"github.com/go-openapi/runtime/middleware/denco"
)

func BenchmarkRouterLookupStatic100(b *testing.B) {
	benchmarkRouterLookupStatic(b, 100)
}

func BenchmarkRouterLookupStatic300(b *testing.B) {
	benchmarkRouterLookupStatic(b, 300)
}

func BenchmarkRouterLookupStatic700(b *testing.B) {
	benchmarkRouterLookupStatic(b, 700)
}

func BenchmarkRouterLookupSingleParam100(b *testing.B) {
	records := makeTestSingleParamRecords(100)
	benchmarkRouterLookupSingleParam(b, records)
}

func BenchmarkRouterLookupSingleParam300(b *testing.B) {
	records := makeTestSingleParamRecords(300)
	benchmarkRouterLookupSingleParam(b, records)
}

func BenchmarkRouterLookupSingleParam700(b *testing.B) {
	records := makeTestSingleParamRecords(700)
	benchmarkRouterLookupSingleParam(b, records)
}

func BenchmarkRouterLookupSingle2Param100(b *testing.B) {
	records := makeTestSingle2ParamRecords(100)
	benchmarkRouterLookupSingleParam(b, records)
}

func BenchmarkRouterLookupSingle2Param300(b *testing.B) {
	records := makeTestSingle2ParamRecords(300)
	benchmarkRouterLookupSingleParam(b, records)
}

func BenchmarkRouterLookupSingle2Param700(b *testing.B) {
	records := makeTestSingle2ParamRecords(700)
	benchmarkRouterLookupSingleParam(b, records)
}

func BenchmarkRouterBuildStatic100(b *testing.B) {
	records := makeTestStaticRecords(100)
	benchmarkRouterBuild(b, records)
}

func BenchmarkRouterBuildStatic300(b *testing.B) {
	records := makeTestStaticRecords(300)
	benchmarkRouterBuild(b, records)
}

func BenchmarkRouterBuildStatic700(b *testing.B) {
	records := makeTestStaticRecords(700)
	benchmarkRouterBuild(b, records)
}

func BenchmarkRouterBuildSingleParam100(b *testing.B) {
	records := makeTestSingleParamRecords(100)
	benchmarkRouterBuild(b, records)
}

func BenchmarkRouterBuildSingleParam300(b *testing.B) {
	records := makeTestSingleParamRecords(300)
	benchmarkRouterBuild(b, records)
}

func BenchmarkRouterBuildSingleParam700(b *testing.B) {
	records := makeTestSingleParamRecords(700)
	benchmarkRouterBuild(b, records)
}

func BenchmarkRouterBuildSingle2Param100(b *testing.B) {
	records := makeTestSingle2ParamRecords(100)
	benchmarkRouterBuild(b, records)
}

func BenchmarkRouterBuildSingle2Param300(b *testing.B) {
	records := makeTestSingle2ParamRecords(300)
	benchmarkRouterBuild(b, records)
}

func BenchmarkRouterBuildSingle2Param700(b *testing.B) {
	records := makeTestSingle2ParamRecords(700)
	benchmarkRouterBuild(b, records)
}

func benchmarkRouterLookupStatic(b *testing.B, n int) {
	b.StopTimer()
	router := denco.New()
	records := makeTestStaticRecords(n)
	if err := router.Build(records); err != nil {
		b.Fatal(err)
	}
	record := pickTestRecord(records)
	b.StartTimer()
	for range b.N {
		if r, _, _ := router.Lookup(record.Key); r != record.Value {
			b.Fail()
		}
	}
}

func benchmarkRouterLookupSingleParam(b *testing.B, records []denco.Record) {
	router := denco.New()
	if err := router.Build(records); err != nil {
		b.Fatal(err)
	}
	record := pickTestRecord(records)
	b.ResetTimer()
	for range b.N {
		if _, _, found := router.Lookup(record.Key); !found {
			b.Fail()
		}
	}
}

func benchmarkRouterBuild(b *testing.B, records []denco.Record) {
	for range b.N {
		router := denco.New()
		if err := router.Build(records); err != nil {
			b.Fatal(err)
		}
	}
}

func makeTestStaticRecords(n int) []denco.Record {
	records := make([]denco.Record, n)
	for i := range n {
		records[i] = denco.NewRecord("/"+randomString(50), fmt.Sprintf("testroute%d", i))
	}
	return records
}

func makeTestSingleParamRecords(n int) []denco.Record {
	records := make([]denco.Record, n)
	for i := range records {
		records[i] = denco.NewRecord(fmt.Sprintf("/user%d/:name", i), fmt.Sprintf("testroute%d", i))
	}
	return records
}

func makeTestSingle2ParamRecords(n int) []denco.Record {
	records := make([]denco.Record, n)
	for i := range records {
		records[i] = denco.NewRecord(fmt.Sprintf("/user%d/:name/comment/:id", i), fmt.Sprintf("testroute%d", i))
	}
	return records
}

func pickTestRecord(records []denco.Record) denco.Record {
	return records[len(records)/2]
}

func randomString(n int) string {
	const srcStrings = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789/"
	var buf bytes.Buffer
	for range n {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(srcStrings)-1)))
		if err != nil {
			panic(err)
		}
		buf.WriteByte(srcStrings[num.Int64()])
	}
	return buf.String()
}
