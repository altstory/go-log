package log

import (
	"context"
)

type logTag struct{}
type logMoreInfo struct{}

// Info 代表一个 k=v 键值对。
type Info struct {
	Key   string
	Value interface{}
}

type moreInfo struct {
	infoList []Info
}

var (
	keyLogTag      logTag
	keyLogMoreInfo logMoreInfo
)

// WithTag 在 ctx 里面存一个日志 tag 信息，用于日志输出。
func WithTag(ctx context.Context, tag string) context.Context {
	return context.WithValue(ctx, keyLogTag, tag)
}

func tag(ctx context.Context) string {
	v := ctx.Value(keyLogTag)

	if v == nil {
		return ""
	}

	return v.(string)
}

// WithMoreInfo 在 ctx 里面保存更多的信息，可以自动在输出 log 时候将这些信息以 k=v 形式输出。
func WithMoreInfo(ctx context.Context, info ...Info) context.Context {
	if len(info) == 0 {
		return ctx
	}

	var infoList []Info

	if more := ctx.Value(keyLogMoreInfo); more != nil {
		old := more.(moreInfo)
		infoList = make([]Info, 0, len(old.infoList)+len(info))
		infoList = append(infoList, old.infoList...)
	}

	infoList = append(infoList, info...)
	return context.WithValue(ctx, keyLogMoreInfo, moreInfo{
		infoList: infoList,
	})
}

func findMoreInfo(ctx context.Context) []Info {
	more := ctx.Value(keyLogMoreInfo)

	if more == nil {
		return nil
	}

	return more.(moreInfo).infoList
}
