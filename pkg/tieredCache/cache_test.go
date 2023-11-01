package tieredCache

import (
	"context"
	"errors"
	"github.com/patrickmn/go-cache"
	"testing"
	"time"
)

type cacheTestCase struct {
	Name           string
	Cache          Cache
	Key            string
	Value          string
	ExpectedOutput string
	ExpectedErr    error
}

func TestTieredCache(t *testing.T) {
	testCases := []cacheTestCase{
		{
			Name:           "go cache",
			Cache:          NewGoCache(cache.New(time.Minute, time.Minute), time.Minute),
			Key:            "test_cache",
			Value:          "test",
			ExpectedOutput: "test",
			ExpectedErr:    nil,
		},
		{
			Name:           "go cache",
			Cache:          NewGoCache(cache.New(time.Minute, time.Minute), time.Minute),
			Key:            "test_cache_fail",
			Value:          "",
			ExpectedOutput: "",
			ExpectedErr:    ErrCacheMiss,
		},
		{
			Name:           "go cache tiered",
			Cache:          NewTieredCache(nil, NewGoCache(cache.New(time.Minute, time.Minute), time.Minute)),
			Key:            "test_cache_fail",
			Value:          "",
			ExpectedOutput: "",
			ExpectedErr:    ErrCacheMiss,
		},
		{
			Name:           "go cache tiered",
			Cache:          NewTieredCache(nil, NewGoCache(cache.New(time.Minute, time.Minute), time.Minute)),
			Key:            "test_cache",
			Value:          "test",
			ExpectedOutput: "test",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Value != "" {
				err := tc.Cache.SetCache(context.Background(), tc.Key, tc.Value)
				if err != nil {
					t.Errorf("failed setting cache:%s", err.Error())
					return
				}
			}
			value, err := tc.Cache.GetCache(context.Background(), tc.Key)
			if err != nil && !errors.Is(err, tc.ExpectedErr) {
				t.Errorf("failed getting cache:%s", err.Error())
				return
			}
			if tc.ExpectedErr != nil {
				return
			}
			if string(value) != tc.ExpectedOutput {
				t.Errorf("does not match expected output: %s != %s", tc.ExpectedOutput, string(value))
			}

		})
	}

}
