package main

// main() exists to satisfy the linker when building this package as a
// c-shared DLL. All real entry points are CGO exports elsewhere in the
// package.
func main() {}
