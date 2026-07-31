package cache

import (
	"context"
	"github.com/allegro/bigcache/v3"
	"time"
)

// 验证码信息缓存3分钟
var RbacCache, _ = bigcache.New(context.Background(), bigcache.DefaultConfig(3*time.Minute))
