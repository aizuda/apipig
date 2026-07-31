package config

type Upload struct {
	Path              string `mapstructure:"path" json:"path" yaml:"path"`                                            // 上传文件路径
	ImageQuality      uint   `mapstructure:"image-quality" json:"imageQuality" yaml:"image-quality"`                  // 图片质量
	ImageCacheControl string `mapstructure:"image-cache-control" json:"imageCacheControl" yaml:"image-cache-control"` // 图片缓存控制

}
