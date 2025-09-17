package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"sync"
	"time"
)

var _ slog.Handler = (*Handler)(nil)

type Handler struct {
	opts *slog.HandlerOptions

	group string
	attrs []slog.Attr

	mu *sync.Mutex
	w  io.Writer
}

func NewHandler(w io.Writer, opts *slog.HandlerOptions) *Handler {
	return &Handler{
		opts: opts,
		mu:   &sync.Mutex{},
		w:    w,
	}
}

func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *Handler) appendAttr(buf []byte, a slog.Attr, group string) []byte {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return buf
	}

	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		if len(attrs) == 0 {
			return buf
		}

		groupName := a.Key
		if len(groupName) == 0 {
			groupName = group
		} else if len(group) > 0 {
			groupName = group + "." + groupName
		}

		for _, ga := range attrs {
			buf = h.appendAttr(buf, ga, groupName)
		}

		return buf
	}

	buf = append(buf, ' ')
	if len(group) == 0 {
		buf = append(buf, a.Key...)
	} else {
		buf = append(buf, group...)
		buf = append(buf, '.')
		buf = append(buf, a.Key...)
	}
	buf = append(buf, '=')

	switch a.Value.Kind() {
	case slog.KindString:
		buf = strconv.AppendQuote(buf, a.Value.String())
	case slog.KindTime:
		buf = strconv.AppendQuote(buf, a.Value.Time().Format(time.DateTime))
	case slog.KindBool:
		buf = strconv.AppendBool(buf, a.Value.Bool())
	case slog.KindFloat64:
		buf = strconv.AppendFloat(buf, a.Value.Float64(), 'g', -1, 64)
	case slog.KindInt64:
		buf = strconv.AppendInt(buf, a.Value.Int64(), 10)
	case slog.KindUint64:
		buf = strconv.AppendUint(buf, a.Value.Uint64(), 10)
	default:
		switch v := a.Value.Any().(type) {
		case error:
			buf = strconv.AppendQuote(buf, v.Error())
		case fmt.Stringer:
			buf = strconv.AppendQuote(buf, v.String())
		default:
			buf = fmt.Append(buf, v)
		}
	}

	return buf
}

func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	const padding = "     "

	buf := make([]byte, 0, 256)
	if !r.Time.IsZero() {
		buf = fmt.Append(buf, r.Time.Format(time.DateTime), " ")
	}

	level := r.Level.String()
	buf = append(buf, level...)
	buf = append(buf, padding[:max(len(padding)-len(level), 0)]...)
	buf = append(buf, ' ')
	buf = append(buf, r.Message...)

	if r.NumAttrs()+len(h.attrs) > 0 {
		buf = append(buf, " ~"...)

		for _, a := range h.attrs {
			buf = h.appendAttr(buf, a, h.group)
		}

		r.Attrs(func(a slog.Attr) bool {
			buf = h.appendAttr(buf, a, h.group)
			return true
		})
	}

	buf = append(buf, '\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(buf)
	return err
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		opts: h.opts,

		group: h.group,
		attrs: append(h.attrs, attrs...),

		mu: h.mu,
		w:  h.w,
	}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	group := name
	if len(h.group) > 0 {
		group = name + "." + h.group
	}

	return &Handler{
		opts: h.opts,

		group: group,
		attrs: h.attrs,

		mu: h.mu,
		w:  h.w,
	}
}
