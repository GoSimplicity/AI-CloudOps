package cache

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

// Length-prefixed fields preserve arbitrary names and label values without
// delimiter collisions. The version also refreshes previously generated caches.
func calculateRecordPoolHash(pool *model.MonitorScrapePool, rules []*model.MonitorRecordRule) string {
	hash := sha256.New()
	writeField := func(value string) {
		var size [8]byte
		binary.LittleEndian.PutUint64(size[:], uint64(len(value)))
		_, _ = hash.Write(size[:])
		_, _ = hash.Write([]byte(value))
	}
	writeField("recording-rules-v2")
	writeField(strconv.Itoa(pool.ID))
	writeField(pool.Name)
	writeField(strconv.Itoa(len(pool.PrometheusInstances)))
	for _, instance := range pool.PrometheusInstances {
		writeField(instance)
	}
	writeField(strconv.Itoa(len(rules)))
	for _, rule := range rules {
		if rule == nil {
			writeField("nil")
			continue
		}
		writeField("rule")
		writeField(rule.Name)
		writeField(rule.Expr)
		writeField(strconv.Itoa(len(rule.Labels)))
		for _, label := range rule.Labels {
			writeField(label)
		}
	}
	return hex.EncodeToString(hash.Sum(nil))
}
