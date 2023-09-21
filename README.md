# gotool

Provides a fully statically compiled Go toolchain meant for use in environments without glibc or other libc implementations.

This was used in https://github.com/anupcshan/gokrazyci, which was run on a Gokrazy device.

As of Go 1.21, default Go toolchain is already statically compiled, making this package unnecessary.
