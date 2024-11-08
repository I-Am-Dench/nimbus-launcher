package patcher

type Logger interface {
	Print(v ...any)
	Printf(format string, v ...any)
	Println(v ...any)
}
