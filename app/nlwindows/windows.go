// Patch Review and Prerequisites windows are not included, even though they should be,
// because they are special cases within the current implementation.
//
// The Patch Review window should theoretically be opened for each configuration receiving patches,
// but to reduce clutter for automatic checks, only the first patch shows up. This is NOT an optimal
// implementation and WILL be fixed before version 1.0.0. The launcher should collect all patches
// first, before showing a window with all available patches.
//
// The Prerequisites window is only created once when the launcher is started, so making it an
// instanced window would not be necessary. The window is put in its own subpackage for consistency
// and because the complexity of checking for/automatically installing prerequisites may grow in
// future versions.
package nlwindows

const (
	Settings = iota
	Info
)
